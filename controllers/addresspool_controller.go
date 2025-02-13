/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024 Wind River Systems, Inc. */

package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers/common"
	cloudManager "github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var logAddressPool = log.Log.WithName("controller").WithName("addresspool")

const AddressPoolControllerName = "addresspool-controller"

const AddressPoolFinalizerName = "addresspool.finalizers.windriver.com"

var _ reconcile.Reconciler = &AddressPoolReconciler{}

// AddressPoolReconciler reconciles a AddressPool object
type AddressPoolReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	cloudManager.CloudManager
	common.ReconcilerErrorHandler
	common.ReconcilerEventLogger
}

// statusUpdateRequired is a utility method to determine if the status needs
// to be updated at the API.
func (r *AddressPoolReconciler) statusUpdateRequired(instance *starlingxv1.AddressPool, status *starlingxv1.AddressPoolStatus) bool {
	return !instance.Status.DeepEqual(status)
}

func (r *AddressPoolReconciler) UpdateInsyncStatus(client *gophercloud.ServiceClient, instance *starlingxv1.AddressPool, oldStatus *starlingxv1.AddressPoolStatus) error {
	if r.statusUpdateRequired(instance, oldStatus) {
		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "synchronization has changed to: %t", instance.Status.InSync)
		err2 := r.Client.Status().Update(context.TODO(), instance)
		if err2 != nil {
			logAddressPool.Error(err2, "failed to update platform network status")
			return err2
		}
	}
	return nil
}

// Fetch active controller's host instance and increment annotation value of
// deployment-manager/notifications only if Generation != ObservedGeneration
// or there is a deploymentScope change.
// Note that for deletion we are just cleaning up the finalizer and we
// are not specifically deleting addresspool object on the system.
func (r *AddressPoolReconciler) ReconcileResource(client *gophercloud.ServiceClient, instance *starlingxv1.AddressPool, request_namespace string) (err error) {
	if instance.DeletionTimestamp.IsZero() {
		if instance.Status.ObservedGeneration != instance.ObjectMeta.Generation {
			if instance.Status.Reconciled {
				instance.Status.Reconciled = false
				err = r.Client.Status().Update(context.TODO(), instance)
				if err != nil {
					msg := fmt.Sprintf("Failed to reset reconciled status of the addresspool '%s'", instance.Name)
					logAddressPool.Error(err, msg)
					return common.NewResourceConfigurationDependency(msg)
				}
			}

			host_instance, _, err := r.CloudManager.GetHostByPersonality(request_namespace, client, cloudManager.ActiveController)
			if err != nil {
				msg := "failed to get active host"
				return common.NewUserDataError(msg)
			}

			err = r.CloudManager.NotifyResource(host_instance)
			if err != nil {
				msg := fmt.Sprintf("Failed to notify '%s' active host instance", host_instance.Name)
				logAddressPool.Error(err, msg)
				return common.NewResourceConfigurationDependency(msg)
			}

			r.ReconcilerEventLogger.NormalEvent(host_instance,
				common.ResourceNotified,
				"Host has been notified due to '%s' addresspool update.",
				instance.Name)

			// Set Generation = ObservedGeneration only when active
			// host controller is successfully notified.
			instance.Status.ObservedGeneration = instance.ObjectMeta.Generation
			err = r.Client.Status().Update(context.TODO(), instance)
			if err != nil {
				msg := fmt.Sprintf(
					"Failed to update '%s' addresspool instance observed generation, attempting retry.",
					instance.Name)
				logAddressPool.Error(err, msg)
				return common.NewResourceConfigurationDependency(msg)
			}
		}
	} else {
		// Remove the finalizer so the kubernetes delete operation can
		// continue.
		instance.ObjectMeta.Finalizers = utils.RemoveString(instance.ObjectMeta.Finalizers, AddressPoolFinalizerName)
		if err := r.Client.Update(context.Background(), instance); err != nil {
			return err
		}
	}

	return nil
}

// Reconcile reads that state of the cluster for a AddressPool object and makes changes based on the state read
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=addresspools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=addresspools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=addresspools/finalizers,verbs=update
func (r *AddressPoolReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// To reduce the repitition of adding the resource name to every log we
	// replace the logger with one that includes the resource name and then
	// restore it at the end of the reconcile function.

	savedLog := logAddressPool
	logAddressPool = logAddressPool.WithName(request.NamespacedName.String())
	defer func() { logAddressPool = savedLog }()

	// Fetch the DataNetwork instance
	instance := &starlingxv1.AddressPool{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically
			// garbage collected. For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}

		logAddressPool.Error(err, "unable to read object")
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Restore the address pool status
	if r.isRestoreInProgress(instance) {
		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "Restoring '%s' Addresspool resource status without doing actual reconciliation", instance.Name)
		if err := r.RestoreAddressPoolStatus(instance); err != nil {
			return reconcile.Result{}, err
		}
		if err := r.ClearRestoreInProgress(instance); err != nil {
			return reconcile.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if instance.DeletionTimestamp.IsZero() {
		// Ensure that the object has a finalizer setup as a pre-delete hook so
		// that we can delete any system resources that we previously added.
		if !utils.ContainsString(instance.ObjectMeta.Finalizers, AddressPoolFinalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, AddressPoolFinalizerName)
			if err := r.Client.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}

			// Might as well return immediately as the update is going to cause
			// another reconcile event for this resource and we don't want to
			// access the system API more than necessary.
			return reconcile.Result{}, nil
		}
	}

	if !utils.IsReconcilerEnabled(utils.AddressPool) {
		return reconcile.Result{}, nil
	}

	platformClient := r.CloudManager.GetPlatformClient(request.Namespace)
	if platformClient == nil {
		// The client has not been authenticated by the system controller so
		// wait.
		r.ReconcilerEventLogger.WarningEvent(instance, common.ResourceDependency,
			"waiting for platform client creation")
		return common.RetryMissingClient, nil
	}

	if !r.CloudManager.GetSystemReady(request.Namespace) {
		r.ReconcilerEventLogger.WarningEvent(instance, common.ResourceDependency,
			"waiting for system reconciliation")
		return common.RetrySystemNotReady, nil
	}

	if !r.CloudManager.IsNotifyingActiveHost() {
		r.CloudManager.SetNotifyingActiveHost(true)
		err = r.ReconcileResource(platformClient, instance, request.NamespacedName.Namespace)
		r.CloudManager.SetNotifyingActiveHost(false)
	} else {
		err = common.NewHostNotifyError("waiting to notify active host")
	}

	if err != nil {
		return r.ReconcilerErrorHandler.HandleReconcilerError(request, err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AddressPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	tMgr := cloudManager.GetInstance(mgr)
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.CloudManager = tMgr
	r.ReconcilerErrorHandler = &common.ErrorHandler{
		CloudManager: tMgr,
		Logger:       logAddressPool}
	r.ReconcilerEventLogger = &common.EventLogger{
		EventRecorder: mgr.GetEventRecorderFor(AddressPoolControllerName),
		Logger:        logAddressPool}
	return ctrl.NewControllerManagedBy(mgr).
		For(&starlingxv1.AddressPool{}).
		Complete(r)
}

// Verify whether we have annotation restore-in-progress
func (r *AddressPoolReconciler) isRestoreInProgress(instance *starlingxv1.AddressPool) bool {
	restoreInProgress, ok := instance.Annotations[cloudManager.RestoreInProgress]
	if ok && restoreInProgress != "" {
		return true
	}
	return false
}

// Updates the AddressPool status while restoring
func (r *AddressPoolReconciler) RestoreAddressPoolStatus(instance *starlingxv1.AddressPool) error {
	annotation := instance.GetObjectMeta().GetAnnotations()
	config, ok := annotation[cloudManager.RestoreInProgress]
	if ok {
		restoreStatus := &cloudManager.RestoreStatus{}
		err := json.Unmarshal([]byte(config), &restoreStatus)
		if err != nil {
			logAddressPool.Error(err, "failed to unmarshal  restore status")
			return nil
		} else {
			instance.Status.InSync = true
			instance.Status.Reconciled = true
			instance.Status.ObservedGeneration = instance.ObjectMeta.Generation

			err = r.Client.Status().Update(context.TODO(), instance)
			if err != nil {
				log_err_msg := fmt.Sprintf(
					"Failed to update AddressPool status while restoring '%s' resource. Error: %s",
					instance.Name,
					err)
				return common.NewResourceStatusDependency(log_err_msg)
			}
			StatusUpdate := fmt.Sprintf("Status updated for AddressPool resource '%s' during restore with following values: Reconciled=%t InSync=%t",
				instance.Name, instance.Status.Reconciled, instance.Status.InSync)
			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, StatusUpdate)
		}
	}
	return nil
}

// Clear annotation RestoreInProgress
func (r *AddressPoolReconciler) ClearRestoreInProgress(instance *starlingxv1.AddressPool) error {
	delete(instance.Annotations, cloudManager.RestoreInProgress)
	if !utils.ContainsString(instance.ObjectMeta.Finalizers, AddressPoolFinalizerName) {
		instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, AddressPoolFinalizerName)
	}
	err := r.Client.Update(context.TODO(), instance)
	if err != nil {
		return common.NewResourceStatusDependency(fmt.Sprintf("Failed to update '%s' addresspool resource after removing '%s' annotation during restoration.",
			instance.Name, cloudManager.RestoreInProgress))
	}
	return nil
}
