/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2024 Wind River Systems, Inc. */

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

// ReconcileResource interacts with the system API in order to reconcile the
// state of a data network with the state stored in the k8s database.
// TODO(sriram-gn): Fetch active controller's host instance and set
// Reconciled/Insync to false only if Generation != ObservedGeneration.
func (r *PlatformNetworkReconciler) ReconcileResource(client *gophercloud.ServiceClient, instance *starlingxv1.PlatformNetwork, request_namespace string) (err error) {
	if instance.DeletionTimestamp.IsZero() {
		host_instance := &starlingxv1.Host{}
		host_namespace := types.NamespacedName{Namespace: instance.Namespace, Name: "controller-0"}
		err := r.Client.Get(context.TODO(), host_namespace, host_instance)
		if err != nil {
			logAddressPool.Error(err, "Failed to get host resource from namespace")
		}

		host_instance.Status.InSync = false
		host_instance.Status.Reconciled = false
		err = r.Client.Status().Update(context.TODO(), host_instance)
		if err != nil {
			logAddressPool.Error(err, "Failed to update status")
			return err
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

	return err
}

// StopAfterInSync determines whether the reconciler should continue processing
// change requests after the configuration has been reconciled a first time.
func (r *PlatformNetworkReconciler) StopAfterInSync() bool {
	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return utils.GetReconcilerOptionBool(utils.PlatformNetwork, utils.StopAfterInSync, true)
}

// Obtain deploymentScope value from configuration
// Taking this value from annotation in instacne
// (It seems Client.Get does not update Status value from configuration)
// "bootstrap" if "bootstrap" in configuration or deploymentScope not specified
// "principal" if "principal" in configuration
func (r *PlatformNetworkReconciler) GetScopeConfig(instance *starlingxv1.PlatformNetwork) (scope string, err error) {
	// Set default value for deployment scope
	deploymentScope := cloudManager.ScopeBootstrap
	// Set DeploymentScope from configuration
	annotation := instance.GetObjectMeta().GetAnnotations()
	if annotation != nil {
		config, ok := annotation["kubectl.kubernetes.io/last-applied-configuration"]
		if ok {
			status_config := &starlingxv1.PlatformNetwork{}
			err := json.Unmarshal([]byte(config), &status_config)
			if err == nil {
				if status_config.Status.DeploymentScope != "" {
					lowerCaseScope := strings.ToLower(status_config.Status.DeploymentScope)
					switch lowerCaseScope {
					case cloudManager.ScopeBootstrap:
						deploymentScope = cloudManager.ScopeBootstrap
					case cloudManager.ScopePrincipal:
						deploymentScope = cloudManager.ScopePrincipal
					default:
						err = fmt.Errorf("Unsupported DeploymentScope: %s",
							status_config.Status.DeploymentScope)
						return deploymentScope, err
					}
				}
			} else {
				err = perrors.Wrapf(err, "failed to Unmarshal annotaion last-applied-configuration")
				return deploymentScope, err
			}
		}
	}
	return deploymentScope, nil
}

// Update deploymentScope and ReconcileAfterInSync in instance
// ReconcileAfterInSync value will be:
// "true"  if deploymentScope is "principal" because it is day 2 operation (update configuration)
// "false" if deploymentScope is "bootstrap"
// Then reflect these values to cluster object
func (r *PlatformNetworkReconciler) UpdateConfigStatus(instance *starlingxv1.PlatformNetwork) (err error) {
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := r.Client.Get(context.TODO(), types.NamespacedName{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		}, instance)
		if err != nil {
			return err
		}
		deploymentScope, err := r.GetScopeConfig(instance)
		if err != nil {
			return err
		}
		logPlatformNetwork.V(2).Info("deploymentScope in configuration", "deploymentScope", deploymentScope)

		// Put ReconcileAfterInSync values depends on scope
		// "true"  if scope is "principal" because it is day 2 operation (update configuration)
		// "false" if scope is "bootstrap" or None
		afterInSync, ok := instance.Annotations[cloudManager.ReconcileAfterInSync]
		if deploymentScope == cloudManager.ScopePrincipal {
			if !ok || afterInSync != "true" {
				instance.Annotations[cloudManager.ReconcileAfterInSync] = "true"
			}
		} else {
			if ok && afterInSync == "true" {
				delete(instance.Annotations, cloudManager.ReconcileAfterInSync)
			}
		}
		// if instance.Spec.NetworkType == "mgmt" {
		// 	instance.Spec.NetworkType = "oam"
		// 	instance.Spec.IPAllocationType = "static"
		// }
		logPlatformNetwork.Info(fmt.Sprintf("%+v", instance))
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

		// Update scope status
		deploymentScope, err := r.GetScopeConfig(instance)
		if err != nil {
			return err
		}
		logPlatformNetwork.V(2).Info("deploymentScope in configuration", "deploymentScope", deploymentScope)
		instance.Status.DeploymentScope = deploymentScope

		// Check if the configuration is updated
		if instance.Status.ObservedGeneration != instance.ObjectMeta.Generation {
			if instance.Status.ObservedGeneration == 0 &&
				instance.Status.Reconciled {
				// Case: DM upgrade in reconciled node
				instance.Status.ConfigurationUpdated = false
			} else {
				// Case: Fresh install or Day-2 operation
				instance.Status.ConfigurationUpdated = true
				if instance.Status.DeploymentScope == cloudManager.ScopePrincipal {
					instance.Status.Reconciled = false
				}
			}
			instance.Status.ObservedGeneration = instance.ObjectMeta.Generation
		}
		logPlatformNetwork.Info(fmt.Sprintf("%+v", instance))
		return r.Client.Status().Update(context.TODO(), instance)
	})

	if err != nil {
		err = perrors.Wrapf(err, "failed to update status: %s",
			common.FormatStruct(instance.Status))
		return err
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

	logPlatformNetwork.Info("Triggered Reconcile AddressPool")
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

	if instance.Status.ObservedGeneration == instance.ObjectMeta.Generation &&
		instance.Status.Reconciled {
		return ctrl.Result{}, nil
	}

	// Update scope from configuration
	logPlatformNetwork.V(2).Info("before UpdateConfigStatus", "instance", instance)
	err = r.UpdateConfigStatus(instance)
	if err != nil {
		logPlatformNetwork.Error(err, "unable to update scope")
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

	// SetPlatformNetworkReconciling(true) so that host reconciler waits for platform
	// network reconciler to complete its reconciliation before Host reconciler
	// propagates unlock_required strategy update.
	r.CloudManager.SetPlatformNetworkReconciling(true)

	err = r.ReconcileResource(platformClient, instance, request.NamespacedName.Namespace)
	if err != nil {
		return r.ReconcilerErrorHandler.HandleReconcilerError(request, err)
	}

	// SetPlatformNetworkReconciling(false) if platform network reconciliation is successful.
	// This would allow Host reconciler to send unlock_required strategy update.
	r.CloudManager.SetPlatformNetworkReconciling(false)

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
			if restoreStatus.InSync != nil {
				instance.Status.InSync = *restoreStatus.InSync
			}
			instance.Status.Reconciled = true
			instance.Status.ObservedGeneration = instance.ObjectMeta.Generation
			instance.Status.DeploymentScope = "bootstrap"
			instance.Status.StrategyRequired = "not_required"
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
