/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package manager

import (
	"fmt"
	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	"github.com/pkg/errors"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"time"
)

// Monitor defines the interface which must be implemented by all concrete
// monitor structures.  A monitor object should be capable of spawning a Go
// routine on Start and stopping that routing on Stop.
type MonitorBody interface {
	// Run is responsible for starting a Go routine to monitor a resource
	Run(client *gophercloud.ServiceClient) (stop bool, err error)

	// State is how the monitor body reports its current state to the monitor.
	State() string
}

// CommonMonitorBody is a common struct that can be inherited by all
// MonitorBody implementations
type CommonMonitorBody struct {
	state string
}

// State retrieves the current state of the monitor body.
func (in *CommonMonitorBody) State() string {
	return in.state
}

// SetState sets the current state of the monitor body.
func (in *CommonMonitorBody) SetState(messageFmt string, args ...interface{}) {
	in.state = fmt.Sprintf(messageFmt, args...)
}

// MonitorManager defines an interface for monitors that need access to the
// manager reference.
type MonitorManager interface {
	SetManager(manager TitaniumManager)
}

// Monitor defines the common behaviour of all monitors.  Since the Monitor
// implementation is simple and can be written generically for all monitors
// that implementation is encapsulated within a base class type structure.
type Monitor struct {
	// MonitorBody is a reference to an actual monitored workload.  It is
	// responsible for all type specific monitoring actions while the Monitor
	// structure is responsible for the generic monitor framework.
	MonitorBody

	// Logger allows a base monitor to implement a logging interface so that
	// it can supply its own custom log "name" when generating logs that are
	// a subset of whatever log stream that is being used by the controller that
	// instantiated this monitor.
	logr.Logger

	// manager is a reference to the Titanium Manager that has been instantiated
	// to oversee all of the controller objects.
	Manager TitaniumManager

	// interval defines the number of seconds between each polling attempt.
	Interval time.Duration

	// object is the kubernetes resource object that is the source of the
	// monitoring event.
	Object runtime.Object

	// stopCh is the stop channel indirectly used by the caller to stop this
	// monitor from running.
	stopCh chan struct{}
}

// BuildMonitorKey is a utility function that formats a string to be used
// as a unique key for a monitor
func BuildMonitorKey(object runtime.Object) string {
	accessor := meta.NewAccessor()

	id, err := accessor.UID(object)
	if err != nil {
		id = "unknown"
	}

	// The intent of this function was to generate a unique key based on:
	// kind+namespace+name, but the GVK on the object is not always populated
	// properly.  This PR is supposed to fix a similar issue but at the time
	// of writing this code we are using v0.1.9 which should contain that fix
	// and we are still seeing the error, so we are going to use the object's
	// UID value since it always seems to be populated properly.  This should
	// revisited as using the kind/namespace/name would be safer for handling
	// deleted and recreated objects.
	//   https://github.com/kubernetes-sigs/controller-runtime/pull/212
	//
	return string(id)
}

// BuildMonitorKey is a utility function that formats a string to be used
// as a unique key for a monitor
func (m *Monitor) GetKey() string {
	return BuildMonitorKey(m.Object)
}

// GetNamespace returns the namespace to which the object being monitored is
// associated.
func (m *Monitor) GetNamespace() string {
	accessor := meta.NewAccessor()

	namespace, err := accessor.Namespace(m.Object)
	if err != nil {
		namespace = "unknown"
	}

	return namespace
}

// Start is responsible for stating the Go routine that will monitor a resource
// or set of resources.
func (m *Monitor) Start(manager TitaniumManager) {
	m.V(2).Info("starting")

	if mgr, ok := m.MonitorBody.(MonitorManager); ok {
		mgr.SetManager(manager)
	}

	m.Manager = manager
	m.stopCh = make(chan struct{})

	go func(stopCh <-chan struct{}) {
		// Set initial interval to immediately run once on startup
		interval := time.Nanosecond

		for {
			select {
			case <-stopCh:
				m.V(2).Info("terminated", "key", m.GetKey())
				return

			case <-time.After(interval):
				// Get the latest client
				client := m.Manager.GetPlatformClient(m.GetNamespace())
				if client == nil {
					// Wait for a client to be created by the system controller.
					m.V(2).Info("platform client not available")
					continue
				}

				stop, err := m.Run(client)

				m.V(1).Info(m.State())

				if stop {
					m.V(2).Info("completed", "key", m.GetKey())
					if m.notify() == nil {
						m.V(2).Info("exiting", "key", m.GetKey())
						return
					}

				} else if err != nil {
					if stop := m.handleClientError(err); stop {
						m.V(2).Info("exiting on error", "key", m.GetKey())
						return
					}
				}

				// Use the configured value on the next iteration.
				interval = m.Interval
			}
		}
	}(m.stopCh)
}

// Stop is responsible for stopping the monitor Go routine.  It does so by
// signaling thru the stop channel.
func (m *Monitor) Stop() {
	if m.stopCh != nil {
		close(m.stopCh)
		m.stopCh = nil
	}
}

// notify is a utility function that updates a monitored object to force
// a reconciliation event that triggers the reconciler.
func (m *Monitor) notify() error {
	err := m.Manager.NotifyResource(m.Object)
	if err != nil {
		err = errors.Cause(err)
		if errors2.IsNotFound(err) {
			// The resource no longer exists so there is no need to continue
			// to run this monitor therefore return "nil" so that it stops.
			m.V(1).Info("resource no longer exists; stopping")
			return nil
		}

		m.Error(err, "failed to notify controller")
		return err
	}

	return nil
}

// handleClientError is responsible for providing custom error handling for
// specific error types.  Currently, it only determines whether or not the Go
// routine should continue or exit based on whether it was able to force an
// update to the object being monitored.
func (m *Monitor) handleClientError(err error) (stop bool) {
	// We were not able to access the client api.  In order to recover we need
	// to transfer control back to the reconciler so that it can decide the best
	// next step.  The best way to do that is to send a notification to the
	// object being monitored.

	// If we successfully notified the reconciler then we can stop monitoring;
	// otherwise we should keep going since there is no other way to ensure that
	// the reconciler will continue.
	return m.notify() == nil
}
