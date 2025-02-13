/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2024 Wind River Systems, Inc. */

package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	perrors "github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers/common"
	cloudManager "github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var logPlatformNetwork = log.Log.WithName("controller").WithName("platformnetwork")

const PlatformNetworkControllerName = "platformnetwork-controller"

const PlatformNetworkFinalizerName = "platformnetwork.finalizers.windriver.com"

const (
	// Defines commonly used network allocation type values.
	AllocationTypeDynamic = "dynamic"
)

var _ reconcile.Reconciler = &PlatformNetworkReconciler{}

// PlatformNetworkReconciler reconciles a PlatformNetwork object
type PlatformNetworkReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	cloudManager.CloudManager
	common.ReconcilerErrorHandler
	common.ReconcilerEventLogger
}

// statusUpdateRequired is a utility method to determine if the status needs
// to be updated at the API.
func (r *PlatformNetworkReconciler) statusUpdateRequired(instance *starlingxv1.PlatformNetwork, status *starlingxv1.PlatformNetworkStatus) bool {
	return !instance.Status.DeepEqual(status)
}

func (r *PlatformNetworkReconciler) UpdateInsyncStatus(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork, oldStatus *starlingxv1.PlatformNetworkStatus) error {
	if r.statusUpdateRequired(instance, oldStatus) {
		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "synchronization has changed to: %t", instance.Status.InSync)
		err2 := r.Client.Status().Update(context.TODO(), instance)
		if err2 != nil {
			logPlatformNetwork.Error(err2, "failed to update platform network status")
			return err2
		}
	}
	return nil
}

// Fetch active controller's host instance and increment annotation value of
// deployment-manager/notifications only if Generation != ObservedGeneration
// or there is a deploymentScope change.
// Note that for deletion we are just cleaning up the finalizer and we
// are not specifically deleting network object on the system.
func (r *PlatformNetworkReconciler) ReconcileResource(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork, reqNs string, scope_updated bool) (err error) {
	if instance.DeletionTimestamp.IsZero() {
		if instance.Status.ObservedGeneration != instance.ObjectMeta.Generation ||
			scope_updated {
			host_instance, _, err := r.CloudManager.GetHostByPersonality(reqNs, client, cloudManager.ActiveController)
			if err != nil {
				msg := "failed to get active host"
				logPlatformNetwork.Error(err, msg)
				return common.NewUserDataError(msg)
			}

			err = r.CloudManager.NotifyResource(host_instance)
			if err != nil {
				msg := fmt.Sprintf("Failed to notify '%s' active host instance", host_instance.Name)
				logPlatformNetwork.Error(err, msg)
				return common.NewResourceConfigurationDependency(msg)
			}

			r.ReconcilerEventLogger.NormalEvent(host_instance,
				common.ResourceNotified,
				"Host has been notified due to '%s' platformnetwork update.",
				instance.Name)

			// Set Generation = ObservedGeneration only when active
			// host controller is successfully notified.
			instance.Status.ObservedGeneration = instance.ObjectMeta.Generation
			err = r.Client.Status().Update(context.TODO(), instance)
			if err != nil {
				msg := fmt.Sprintf(
					"Failed to update '%s' platformnetwork instance observed generation, attempting retry.",
					instance.Name)
				logPlatformNetwork.Error(err, msg)
				return common.NewResourceConfigurationDependency(msg)
			}
		}

		return nil
	} else {
		// Remove the finalizer so the kubernetes delete operation can
		// continue.
		instance.ObjectMeta.Finalizers = utils.RemoveString(instance.ObjectMeta.Finalizers, PlatformNetworkFinalizerName)
		if err := r.Client.Update(context.Background(), instance); err != nil {
			return err
		}
	}

	return nil
}

// StopAfterInSync determines whether the reconciler should continue processing
// change requests after the configuration has been reconciled a first time.
func (r *PlatformNetworkReconciler) StopAfterInSync() bool {
	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return utils.GetReconcilerOptionBool(utils.PlatformNetwork, utils.StopAfterInSync, true)
}

// UpdateDeploymentScope function is used to update the deployment scope for PlatformNetwork.
func (r *PlatformNetworkReconciler) UpdateDeploymentScope(instance *starlingxv1.PlatformNetwork) (error, bool) {
	updated, err := common.UpdateDeploymentScope(r.Client, instance)
	if err != nil {
		logPlatformNetwork.Error(err, "failed to update deploymentScope")
		return err, false
	}
	return nil, updated
}

// Update ReconcileAfterInSync in instance
// ReconcileAfterInSync value will be:
// "true"  if deploymentScope is "principal" because it is day 2 operation (update configuration)
// "false" if deploymentScope is "bootstrap"
// Then reflect these values to cluster object
// It is expected that instance.Status.Deployment scope is already updated by
// UpdateDeploymentScope at this point.
func (r *PlatformNetworkReconciler) UpdateConfigStatus(instance *starlingxv1.PlatformNetwork, ns string) (err error) {
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := r.Client.Get(context.TODO(), types.NamespacedName{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		}, instance)
		if err != nil {
			return err
		}

		// Put ReconcileAfterInSync values depends on scope
		// "true"  if scope is "principal" because it is day 2 operation (update configuration)
		// "false" if scope is "bootstrap" or None
		afterInSync, ok := instance.Annotations[cloudManager.ReconcileAfterInSync]
		if instance.Status.DeploymentScope == cloudManager.ScopePrincipal {
			if !ok || afterInSync != "true" {
				instance.Annotations[cloudManager.ReconcileAfterInSync] = "true"
			}
		} else {
			if ok && afterInSync == "true" {
				delete(instance.Annotations, cloudManager.ReconcileAfterInSync)
			}
		}
		return r.Client.Update(context.TODO(), instance)
	})
	if err != nil {
		err = perrors.Wrapf(err, "failed to update profile annotation ReconcileAfterInSync")
		return err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := r.Client.Get(context.TODO(), types.NamespacedName{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		}, instance)
		if err != nil {
			return err
		}

		if instance.Status.ObservedGeneration != instance.ObjectMeta.Generation {
			if instance.Status.ObservedGeneration == 0 &&
				instance.Status.Reconciled {
				// Case: DM upgrade in reconciled node
				instance.Status.ConfigurationUpdated = false
			} else {
				// Case: Fresh install or Day-2 operation
				instance.Status.ConfigurationUpdated = true
				instance.Status.Reconciled = false
			}
		}
		return r.Client.Status().Update(context.TODO(), instance)
	})

	if err != nil {
		err = perrors.Wrapf(err, "failed to update status: %s",
			common.FormatStruct(instance.Status))
		return err
	}

	return nil
}

// During factory install, the reconciled status is expected to be updated to
// false to unblock the configuration as the day 1 configuration.
// UpdateStatusForFactoryInstall updates the status by checking the factory
// install data.
func (r *PlatformNetworkReconciler) UpdateStatusForFactoryInstall(
	ns string,
	instance *starlingxv1.PlatformNetwork,
) error {
	reconciledUpdated, err := r.CloudManager.GetFactoryResourceDataUpdated(
		ns,
		instance.Name,
		"reconciled",
	)
	if err != nil {
		return err
	}

	if !reconciledUpdated {
		instance.Status.Reconciled = false
		err = r.Client.Status().Update(context.TODO(), instance)
		if err != nil {
			return err
		}
		err = r.CloudManager.SetFactoryResourceDataUpdated(
			ns,
			instance.Name,
			"reconciled",
			true,
		)
		if err != nil {
			return err
		}
		r.ReconcilerEventLogger.NormalEvent(
			instance,
			common.ResourceUpdated,
			"Set Reconciled false for factory install",
		)
	}
	return nil
}

// Reconcile reads that state of the cluster for a PlatformNetwork object and makes changes based on the state read
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=platformnetworks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=platformnetworks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=platformnetworks/finalizers,verbs=update
func (r *PlatformNetworkReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// To reduce the repitition of adding the resource name to every log we
	// replace the logger with one that includes the resource name and then
	// restore it at the end of the reconcile function.

	savedLog := logPlatformNetwork
	logPlatformNetwork = logPlatformNetwork.WithName(request.NamespacedName.String())
	defer func() { logPlatformNetwork = savedLog }()

	// Fetch the DataNetwork instance
	instance := &starlingxv1.PlatformNetwork{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically
			// garbage collected. For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}

		logPlatformNetwork.Error(err, "unable to read object")
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Restore the PlatformNetwork status
	if r.checkRestoreInProgress(instance) {
		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "Restoring '%s' PlatformNetwork resource status without doing actual reconciliation", instance.Name)
		if err := r.RestorePlatformNetworkStatus(instance); err != nil {
			return reconcile.Result{}, err
		}
		if err := r.ClearRestoreInProgress(instance); err != nil {
			return reconcile.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	err, scopeUpdated := r.UpdateDeploymentScope(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	factory, err := r.CloudManager.GetFactoryInstall(request.Namespace)
	if err != nil {
		return reconcile.Result{}, err
	}
	if factory {
		err := r.UpdateStatusForFactoryInstall(request.Namespace, instance)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Update ReconciledAfterInSync and ObservedGeneration.
	logPlatformNetwork.V(2).Info("before UpdateConfigStatus", "instance", instance)
	err = r.UpdateConfigStatus(instance, request.Namespace)
	if err != nil {
		logPlatformNetwork.Error(err, "unable to update ReconciledAfterInSync or ObservedGeneration.")
		return reconcile.Result{}, err
	}
	logPlatformNetwork.V(2).Info("after UpdateConfigStatus", "instance", instance)

	if instance.DeletionTimestamp.IsZero() {
		// Ensure that the object has a finalizer setup as a pre-delete hook so
		// that we can delete any system resources that we previously added.
		if !utils.ContainsString(instance.ObjectMeta.Finalizers, PlatformNetworkFinalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, PlatformNetworkFinalizerName)
			if err := r.Client.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}

			// Might as well return immediately as the update is going to cause
			// another reconcile event for this resource and we don't want to
			// access the system API more than necessary.
			return reconcile.Result{}, nil
		}
	}

	if !utils.IsReconcilerEnabled(utils.PlatformNetwork) {
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
		err = r.ReconcileResource(platformClient, instance, request.NamespacedName.Namespace, scopeUpdated)
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
func (r *PlatformNetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	tMgr := cloudManager.GetInstance(mgr)
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.CloudManager = tMgr
	r.ReconcilerErrorHandler = &common.ErrorHandler{
		CloudManager: tMgr,
		Logger:       logPlatformNetwork}
	r.ReconcilerEventLogger = &common.EventLogger{
		EventRecorder: mgr.GetEventRecorderFor(PlatformNetworkControllerName),
		Logger:        logPlatformNetwork}
	return ctrl.NewControllerManagedBy(mgr).
		For(&starlingxv1.PlatformNetwork{}).
		Complete(r)
}

// Verify whether we have annotation restore-in-progress
func (r *PlatformNetworkReconciler) checkRestoreInProgress(instance *starlingxv1.PlatformNetwork) bool {
	restoreInProgress, ok := instance.Annotations[cloudManager.RestoreInProgress]
	if ok && restoreInProgress != "" {
		return true
	}
	return false
}

// Update status
func (r *PlatformNetworkReconciler) RestorePlatformNetworkStatus(instance *starlingxv1.PlatformNetwork) error {
	annotation := instance.GetObjectMeta().GetAnnotations()
	config, ok := annotation[cloudManager.RestoreInProgress]
	if ok {
		restoreStatus := &cloudManager.RestoreStatus{}
		err := json.Unmarshal([]byte(config), &restoreStatus)
		if err == nil {
			instance.Status.InSync = true
			instance.Status.Reconciled = true
			instance.Status.ObservedGeneration = instance.ObjectMeta.Generation
			instance.Status.DeploymentScope = "bootstrap"
			err = r.Client.Status().Update(context.TODO(), instance)
			if err != nil {
				log_err_msg := fmt.Sprintf(
					"Failed to update platform network status while restoring '%s' resource. Error: %s",
					instance.Name,
					err)
				return common.NewResourceStatusDependency(log_err_msg)
			} else {
				StatusUpdate := fmt.Sprintf("Status updated for PlatformNetwork resource '%s' during restore with following values: Reconciled=%t InSync=%t DeploymentScope=%s",
					instance.Name, instance.Status.Reconciled, instance.Status.InSync, instance.Status.DeploymentScope)
				r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, StatusUpdate)

			}
		} else {
			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "Failed to unmarshal '%s'", err)
		}
	}
	return nil
}

// Clear annotation RestoreInProgress
func (r *PlatformNetworkReconciler) ClearRestoreInProgress(instance *starlingxv1.PlatformNetwork) error {
	delete(instance.Annotations, cloudManager.RestoreInProgress)
	if !utils.ContainsString(instance.ObjectMeta.Finalizers, PlatformNetworkFinalizerName) {
		instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, PlatformNetworkFinalizerName)
	}
	err := r.Client.Update(context.TODO(), instance)
	if err != nil {
		return common.NewResourceStatusDependency(fmt.Sprintf("Failed to update '%s' platform network resource after removing '%s' annotation during restoration.",
			instance.Name, cloudManager.RestoreInProgress))
	}
	return nil
}
