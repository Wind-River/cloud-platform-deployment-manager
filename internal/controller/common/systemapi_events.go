/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2026 Wind River Systems, Inc. */

package common

import (
	"fmt"
	"os"
	"strings"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reasons of the Warning events generated when a system API request fails.
// They mirror the classification used to log and retry the request.
const (
	// SystemAPIRejected is used when the system API rejected the request
	// (4xx), usually because the configuration must be corrected.
	SystemAPIRejected = "SystemAPIRejected"

	// SystemAPITransient is used when the request is likely to succeed if
	// retried (no response, request timeout, rate limiting).
	SystemAPITransient = "SystemAPITransient"

	// SystemAPIServerError is used when the system API failed to process the
	// request (5xx).
	SystemAPIServerError = "SystemAPIServerError"
)

const (
	// DeploymentName is the name of the deployment of the deployment manager,
	// the object the system API failure events are recorded on.  The playbooks
	// use the same name to read its logs.
	DeploymentName = "platform-deployment-manager"

	// NamespaceEnv is the environment variable that overrides the namespace
	// of the deployment manager, e.g., to run it outside of a pod.  It is not
	// set by the chart.
	NamespaceEnv = "POD_NAMESPACE"
)

// namespaceFile holds the namespace of the pod.  Kubernetes mounts it, with
// the service account token, in every pod that has one, which is also where
// controller-runtime reads the namespace of the leader election from.
var namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

// getNamespace returns the namespace of the deployment manager: the one of the
// pod, or the value of NamespaceEnv if it is set.  It is empty if neither is
// available.
func getNamespace() string {
	if namespace := os.Getenv(NamespaceEnv); namespace != "" {
		return namespace
	}
	data, err := os.ReadFile(namespaceFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

const (
	// systemAPIEventsComponent is the source of the system API failure events.
	systemAPIEventsComponent = "platform-deployment-manager"

	// maxEventMessageLen is the maximum message length accepted by the
	// Kubernetes API server for an event.
	maxEventMessageLen = 1024
)

// SystemAPIEvents records the requests that failed on the system API as
// Warning events on the deployment manager deployment, whatever the resource
// whose request failed.  They are therefore all found in the namespace of the
// deployment, e.g. with:
//
//	kubectl -n platform-deployment-manager get events
//
// The deployment is used instead of the pod because it outlives the pod: the
// pod is replaced when the host is unlocked, and the events recorded by the
// previous pod are still found under the same object.
// Kubernetes folds identical events into one, with a count and the last time
// it was seen, so a request that fails on every retry does not flood the
// cluster.
type SystemAPIEvents struct {
	record.EventRecorder

	// Target is the object the events are recorded on, the deployment of the
	// deployment manager.
	Target *v1.ObjectReference
}

// systemAPIEventsCorrelatorOptions tunes the client side event correlation
// for the events which are all recorded on the same object (the deployment).  By
// default the events of an object are rate limited together, and the ones
// with the same reason are combined into a single event once they have more
// than 10 different messages: either would drop or hide failures that are
// unrelated to each other.  Each failure, identified by its reason and
// message, is therefore rate limited and folded on its own.
func systemAPIEventsCorrelatorOptions() record.CorrelatorOptions {
	return record.CorrelatorOptions{
		KeyFunc: func(event *v1.Event) (string, string) {
			key, _ := record.EventAggregatorByReasonFunc(event)
			return key + event.Message, event.Message
		},
		SpamKeyFunc: func(event *v1.Event) string {
			key, _ := record.EventAggregatorByReasonFunc(event)
			return key + event.Message
		},
	}
}

var (
	systemAPIEventsOnce     sync.Once
	systemAPIEventsInstance *SystemAPIEvents
)

// GetSystemAPIEvents returns the recorder shared by all controllers for the
// system API failures, creating it on first use.  It returns nil, and no event
// is recorded, if the namespace of the manager is unknown (i.e., when it does
// not run in a pod and NamespaceEnv is not set) or the recorder cannot be
// created.
func GetSystemAPIEvents(mgr ctrl.Manager) *SystemAPIEvents {
	systemAPIEventsOnce.Do(func() {
		log := ctrl.Log.WithName("system-api-events")

		namespace := getNamespace()
		if namespace == "" {
			log.Info("system API failures are not recorded as events, the namespace is unknown",
				"file", namespaceFile, "variable", NamespaceEnv)
			return
		}

		clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
		if err != nil {
			log.Error(err, "system API failures are not recorded as events")
			return
		}

		// The UID is not known (reading it would need the permission to get
		// deployments): the events are found by the kind, name and namespace.
		systemAPIEventsInstance = newSystemAPIEvents(clientset, mgr.GetScheme(), &v1.ObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Namespace:  namespace,
			Name:       DeploymentName,
		})
	})
	return systemAPIEventsInstance
}

// newSystemAPIEvents returns a recorder of the system API failures that
// creates its events in the namespace of the target object.
func newSystemAPIEvents(clientset kubernetes.Interface, scheme *runtime.Scheme, target *v1.ObjectReference) *SystemAPIEvents {
	broadcaster := record.NewBroadcasterWithCorrelatorOptions(systemAPIEventsCorrelatorOptions())
	broadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: clientset.CoreV1().Events(target.Namespace)})

	return &SystemAPIEvents{
		EventRecorder: broadcaster.NewRecorder(scheme, v1.EventSource{Component: systemAPIEventsComponent}),
		Target:        target,
	}
}

// systemAPIEventReason returns the reason of the event recorded for a failed
// system API request.
func systemAPIEventReason(apiErr ErrSystemAPI) string {
	switch {
	case apiErr.Transient():
		return SystemAPITransient
	case apiErr.ServerError():
		return SystemAPIServerError
	default:
		return SystemAPIRejected
	}
}

// Record generates a Warning event on the deployment describing a failed system API
// request: the kind, namespace and name of the resource being reconciled, the
// operation, the reason reported by the server and the status code.
func (in *SystemAPIEvents) Record(kind string, request reconcile.Request, apiErr ErrSystemAPI) {
	msg := fmt.Sprintf("%s %s: %s", kind, request.NamespacedName, apiErr.Error())
	if len(msg) > maxEventMessageLen {
		msg = msg[:maxEventMessageLen-3] + "..."
	}
	in.Event(in.Target, v1.EventTypeWarning, systemAPIEventReason(apiErr), msg)
}
