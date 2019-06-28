/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package hostprofile

import (
	"context"
	"fmt"
	perrors "github.com/pkg/errors"
	starlingxv1beta1 "github.com/wind-river/titanium-deployment-manager/pkg/apis/starlingx/v1beta1"
	"github.com/wind-river/titanium-deployment-manager/pkg/controller/common"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller").WithName("hostprofile")

const ControllerName = "hostprofile-controller"

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new HostProfile Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileHostProfile{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		ReconcilerEventLogger: &common.EventLogger{
			EventRecorder: mgr.GetRecorder(ControllerName),
			Logger:        log},
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to HostProfile
	err = c.Watch(&source.Kind{Type: &starlingxv1beta1.HostProfile{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileHostProfile{}

// ReconcileHostProfile reconciles a HostProfile object
type ReconcileHostProfile struct {
	client.Client
	scheme *runtime.Scheme
	common.ReconcilerEventLogger
}

// ProfileUses determines whether the 'base' profile references the 'target'
// profile either directly or indirectly via one of its parent/base profiles.
func (r *ReconcileHostProfile) ProfileUses(namespace, base, target string) (bool, error) {
	profile := &starlingxv1beta1.HostProfile{}

	for {
		// Retrieve the current level profile
		name := types.NamespacedName{Namespace: namespace, Name: base}
		err := r.Get(context.TODO(), name, profile)
		if err != nil {
			if !errors.IsNotFound(err) {
				err = perrors.Wrapf(err, "failed to lookup profile: %s", base)
				return false, err
			}

			return false, nil
		}

		if profile.Spec.Base != nil && *profile.Spec.Base == target {
			// If it references the target profile then return true
			return true, nil
		} else if profile.Spec.Base == nil {
			// If we have reached the top of the profile chain return false
			return false, nil
		}

		// Otherwise, repeat with the next profile in the chain
		base = *profile.Spec.Base
	}
}

// UpdateHosts will force a update to each host that references this profile.
// This is to ensure that hosts get reconciled whenever any of their profiles
// get updated.
func (r *ReconcileHostProfile) UpdateHosts(instance *starlingxv1beta1.HostProfile) error {
	hosts := &starlingxv1beta1.HostList{}
	opts := client.ListOptions{}
	opts.InNamespace(instance.Namespace)
	err := r.List(context.TODO(), &opts, hosts)
	if err != nil {
		err = perrors.Wrap(err, "failed to get host list")
		return err
	}

	for _, h := range hosts.Items {
		// If this host uses this profile the assume an update is required.
		updateRequired := h.Spec.Profile == instance.Name

		if updateRequired == false {
			// Otherwise, look at the profile chain to figure out if its used
			updateRequired, err = r.ProfileUses(instance.Namespace, h.Spec.Profile, instance.Name)
			if err != nil {
				return err
			}
		}

		if updateRequired {
			// Check that the host hasn't already been updated for this profile
			key := fmt.Sprintf("profile/%s", instance.Name)
			value := instance.ResourceVersion

			if x, ok := h.Annotations[key]; ok {
				updateRequired = updateRequired && (x != value)
			}
			h.Annotations[key] = value
		}

		if updateRequired {
			log.Info("updating host to trigger reconciliation via profile update", "host", h.Name)

			err = r.Update(context.TODO(), &h)
			if err != nil {
				err = perrors.Wrapf(err, "failed to update profile annotation on host %s", h.Name)
				return err
			}
		}
	}

	return nil
}

// Reconcile reads that state of the cluster for a HostProfile object and makes changes based on the state read
// and what is in the HostProfile.Spec
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=hostprofiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=hostprofiles/status,verbs=get;update;patch
func (r *ReconcileHostProfile) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	savedLog := log
	log = log.WithName(request.NamespacedName.String())
	defer func() { log = savedLog }()

    log.V(2).Info("reconcile called")

	// Fetch the HostProfile instance
	instance := &starlingxv1beta1.HostProfile{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		log.Error(err, "unable to read object: %v", request)
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Force an update to each of the hosts that reference this profile.
	err = r.UpdateHosts(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	r.NormalEvent(instance, common.ResourceUpdated,
		"host profile has been updated")

	return reconcile.Result{}, nil
}
