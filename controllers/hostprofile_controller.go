/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	perrors "github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers/common"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var logHostProfile = log.Log.WithName("controller").WithName("hostprofile")

const HostProfileControllerName = "hostprofile-controller"

// HostProfileReconciler reconciles a HostProfile object
type HostProfileReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	common.ReconcilerEventLogger
}

var _ reconcile.Reconciler = &HostProfileReconciler{}

// ProfileUses determines whether the 'base' profile references the 'target'
// profile either directly or indirectly via one of its parent/base profiles.
func (r *HostProfileReconciler) ProfileUses(namespace, base, target string) (bool, error) {
	profile := &starlingxv1.HostProfile{}

	for {
		// Retrieve the current level profile
		name := types.NamespacedName{Namespace: namespace, Name: base}
		err := r.Client.Get(context.TODO(), name, profile)
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
func (r *HostProfileReconciler) UpdateHosts(instance *starlingxv1.HostProfile) error {
	hosts := &starlingxv1.HostList{}
	opts := client.ListOptions{}
	opts.Namespace = instance.Namespace
	err := r.List(context.TODO(), hosts, &opts)
	if err != nil {
		err = perrors.Wrap(err, "failed to get host list")
		return err
	}

	for _, h := range hosts.Items {
		// If this host uses this profile the assume an update is required.
		updateRequired := h.Spec.Profile == instance.Name

		if !updateRequired {
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
			logHostProfile.Info("updating host to trigger reconciliation via profile update", "host", h.Name)

			err = r.Client.Update(context.TODO(), &h)
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
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=hostprofiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=hostprofiles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=hostprofiles/finalizers,verbs=update
func (r *HostProfileReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	savedLog := logHostProfile
	logHostProfile = logHostProfile.WithName(request.NamespacedName.String())
	defer func() { logHostProfile = savedLog }()

	logHostProfile.V(2).Info("reconcile called")

	// Fetch the HostProfile instance
	instance := &starlingxv1.HostProfile{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		logHostProfile.Error(err, "unable to read object: %v", request)
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Force an update to each of the hosts that reference this profile.
	err = r.UpdateHosts(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
		"host profile has been updated")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HostProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.ReconcilerEventLogger = &common.EventLogger{
		EventRecorder: mgr.GetEventRecorderFor(HostProfileControllerName),
		Logger:        logHostProfile}

	return ctrl.NewControllerManagedBy(mgr).
		For(&starlingxv1.HostProfile{}).
		Complete(r)
}
