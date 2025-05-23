/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022-2024 Wind River Systems, Inc. */

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinstances"
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

var logPtpInstance = log.Log.WithName("controller").WithName("ptpinstance")

const PtpInstanceControllerName = "ptpinstance-controller"

const PtpInstanceFinalizerName = "ptpinstance.finalizers.windriver.com"

var _ reconcile.Reconciler = &PtpInstanceReconciler{}

// PtpInstanceReconciler reconciles a PtpInstance object
type PtpInstanceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	cloudManager.CloudManager
	common.ReconcilerErrorHandler
	common.ReconcilerEventLogger
}

func instanceUpdateRequired(instance *starlingxv1.PtpInstance, i *ptpinstances.PTPInstance) (result bool) {
	if instance.Name != i.Name {
		result = true
	}

	spec := instance.Spec
	if spec.Service != i.Service {
		result = true
	}

	return result
}

func instanceParameterUpdateRequired(instance *starlingxv1.PtpInstance, i *ptpinstances.PTPInstance, r *PtpInstanceReconciler) (added []string, removed []string, result bool) {
	configured := instance.Spec.InstanceParameters
	current := i.Parameters
	result = false

	// Diff the lists to determine if changes need to be applied
	added, removed, _ = utils.ListDelta(current, configured)
	var delta strings.Builder

	if len(added) > 0 || len(removed) > 0 {
		result = true

		for _, a := range added {
			delta.WriteString("\t+ ")
			delta.WriteString(a)
			delta.WriteString("\n")
		}

		for _, r := range removed {
			delta.WriteString("\t- ")
			delta.WriteString(r)
			delta.WriteString("\n")
		}
	}
	deltaString := delta.String()
	if deltaString != "" {
		deltaString = "\n" + strings.TrimSuffix(deltaString, "\n")
		logPtpInstance.Info(fmt.Sprintf("delta configuration:%s\n", deltaString))
	}
	instance.Status.Delta = deltaString

	err := r.Client.Status().Update(context.TODO(), instance)
	if err != nil {
		logPtpInstance.Info(fmt.Sprintf("failed to update status:  %s\n", err))
	}

	return added, removed, result
}

// ReconcileParamAdded is a method which handles adding new Parameters to
// associate with an existing PTP instance
func (r *PtpInstanceReconciler) ReconcileParamAdded(client *gophercloud.ServiceClient, params []string, i *ptpinstances.PTPInstance) (*ptpinstances.PTPInstance, error) {

	id := i.UUID
	for _, param := range params {
		opts := ptpinstances.PTPParamToPTPInstOpts{Parameter: &param}

		logPtpInstance.Info("adding ptp parameter", "opts", opts)

		new, err := ptpinstances.AddPTPParamToPTPInst(client, id, opts).Extract()

		if err != nil {
			return i, err
		}

		*i = *new
	}

	return i, nil
}

// ReconcileParamAdded is a method which handles removing new Parameters from
// an existing PTP instance
func (r *PtpInstanceReconciler) ReconcileParamRemoved(client *gophercloud.ServiceClient, params []string, i *ptpinstances.PTPInstance) (*ptpinstances.PTPInstance, error) {

	id := i.UUID
	for _, param := range params {
		opts := ptpinstances.PTPParamToPTPInstOpts{Parameter: &param}

		logPtpInstance.Info("removing ptp parameter", "opts", opts)

		new, err := ptpinstances.RemovePTPParamFromPTPInst(client, id, opts).Extract()

		if err != nil {
			return i, err
		}

		*i = *new
	}

	return i, nil
}

// ReconcileNew is a method which handles reconciling a new data resource and
// creates the corresponding system resource thru the system API.
func (r *PtpInstanceReconciler) ReconcileNew(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInstance) (*ptpinstances.PTPInstance, error) {
	if instance.Status.Reconciled && r.StopAfterInSync() {
		// Do not process any further changes once we have reached a
		// synchronized state unless there is an annotation on the resource.
		if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
			msg := common.NoProvisioningAfterReconciled
			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, msg)
			return nil, common.NewChangeAfterInSync(msg)
		} else {
			logPtpInstance.Info(common.ProvisioningAllowedAfterReconciled)
		}
	}

	// Create a new PTP instance
	opts := ptpinstances.PTPInstanceOpts{
		Name:    &instance.Name,
		Service: &instance.Spec.Service,
	}

	logPtpInstance.Info("creating ptp instance", "opts", opts)

	new, err := ptpinstances.Create(client, opts).Extract()
	if err != nil {
		err = perrors.Wrapf(err, "failed to create: %s", common.FormatStruct(opts))
		return nil, err
	}

	new, err = r.ReconcileParamAdded(client, instance.Spec.InstanceParameters, new)
	if err != nil {
		err = perrors.Wrapf(err, "failed to add parameter to: %s", common.FormatStruct(opts))
		return nil, err
	}

	r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceCreated,
		"ptp instance has been created")

	return new, nil
}

// Remove finalizer
func (r *PtpInstanceReconciler) removePtpInstanceFinalizer(instance *starlingxv1.PtpInstance) {
	// Remove the finalizer so the kubernetes delete operation can continue.
	instance.ObjectMeta.Finalizers = utils.RemoveString(instance.ObjectMeta.Finalizers, PtpInstanceFinalizerName)
	if err := r.Client.Update(context.Background(), instance); err != nil {
		logPtpInstance.Error(err, "failed to remove the finalizer in the ptpInstance because of the error:%v")
	}
}

// ReconciledDeleted is a method which handles reconciling a new data resource and
// creates the corresponding system resource thru the system API.
func (r *PtpInstanceReconciler) ReconciledDeleted(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInstance, i *ptpinstances.PTPInstance) error {
	if utils.ContainsString(instance.ObjectMeta.Finalizers, PtpInstanceFinalizerName) {
		defer r.removePtpInstanceFinalizer(instance)
		if i != nil {
			// Unless it was already deleted go ahead and attempt to delete it.
			err := ptpinstances.Delete(client, i.UUID).ExtractErr()
			if err != nil {
				if code, ok := err.(gophercloud.ErrDefault400); !ok {
					err = perrors.Wrap(err, "failed to delete ptp instance")
					return err
				} else if code.Actual == 400 {
					logPtpInstance.Info("PTP instance is still in use; deleting local resource anayway")
					// NOTE: there is no way to block the kubernetes delete beyond
					//  delaying it a little while we delete an external resource so
					//  since we can't fail this then log it and allow it to
					//  continue for now.
				} else {
					err := perrors.Wrap(err, "unexpected response code on PTP instance deletion")
					return err
				}
			}

			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceDeleted, "PTP instance has been deleted")
		}
	}

	return nil
}

// statusUpdateRequired is a utility function which determines whether an update
// is required to the host status attribute.  Updating this unnecessarily
// will result in an infinite reconciliation loop.
func (r *PtpInstanceReconciler) statusUpdateRequired(instance *starlingxv1.PtpInstance, existing *ptpinstances.PTPInstance, inSync bool) (result bool) {
	status := &instance.Status

	if existing != nil {
		if status.ID == nil || *status.ID != existing.UUID {
			status.ID = &existing.UUID
			result = true
		}
	} else {
		status.ID = nil
	}

	if status.InSync != inSync {
		status.InSync = inSync
		result = true
	}

	if status.InSync && !status.Reconciled {
		// Record the fact that we have reached inSync at least once.
		status.Reconciled = true
		status.ConfigurationUpdated = false
		status.StrategyRequired = cloudManager.StrategyNotRequired
		if instance.Status.DeploymentScope == cloudManager.ScopePrincipal {
			r.CloudManager.SetResourceInfo(cloudManager.ResourcePtpinstance, "", instance.Name, status.Reconciled, status.StrategyRequired)
		}
		result = true
	}

	return result
}

// FindExistingPTPInstance attempts to find the existing PTP instance referenced by
// the ID value stored in the status or to find another resource with a matching
// name.
func (r *PtpInstanceReconciler) FindExistingPTPInstance(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInstance) (found *ptpinstances.PTPInstance, err error) {
	id := instance.Status.ID
	if id != nil {
		// This PTP instance was previously provisioned.
		found, err = ptpinstances.Get(client, *id).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); !ok {
				err = perrors.Wrapf(err, "failed to get: %s", *id)
				return nil, err
			}

			// The resource may have been deleted by the system or operator
			// therefore continue and attempt to recreate it.
			logPtpInstance.Info("resource no longer exists", "id", *id)
			return nil, nil
		}

	} else {
		// This PTP instance needs to be provisioned if it doesn't already exist.
		results, err := ptpinstances.ListPTPInstances(client)
		if err != nil {
			err = perrors.Wrap(err, "failed to list")
			return nil, err
		}

		for _, result := range results {
			if result.Name == instance.Name {
				if result.Service == instance.Spec.Service {
					logPtpInstance.Info("found existing ptp instance", "uuid", result.UUID)
					found = &result
					break
				}
			}
		}
	}

	return found, err
}

// ReconcileUpdated is a method which handles reconciling an existing data
// resource and updates the corresponding system resource thru the system API to
// match the desired state of the resource.
func (r *PtpInstanceReconciler) ReconcileUpdated(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInstance, existing *ptpinstances.PTPInstance) error {
	if ok := instanceUpdateRequired(instance, existing); ok {
		if instance.Status.Reconciled && r.StopAfterInSync() {
			// Do not process any further changes once we have reached a
			// synchronized state unless there is an annotation on the resource.
			if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
				msg := common.NoProvisioningAfterReconciled
				r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, msg)
				return common.NewChangeAfterInSync(msg)
			} else {
				logPtpInstance.Info(common.ProvisioningAllowedAfterReconciled)
			}
		}

		// As there's not sysinv API to update the name and service type of a
		// PTP instance, delete the existing and create a new one.
		logPtpInstance.Info("deleting PTP instance", "status", instance.Status)
		err := r.ReconciledDeleted(client, instance, existing)
		if err != nil {
			logPtpInstance.Info("failed to delete PTP instance", "status", instance.Status)
		}

		logPtpInstance.Info("creating new PTP instance", "status", instance.Status)
		new, err2 := r.ReconcileNew(client, instance)

		if err2 != nil {
			return err2
		}

		*existing = *new

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
			"ptp instance has been updated")

	} else if added, removed, required := instanceParameterUpdateRequired(instance, existing, r); required {
		if instance.Status.Reconciled && r.StopAfterInSync() {
			// Do not process any further changes once we have reached a
			// synchronized state unless there is an annotation on the resource.
			if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
				msg := common.NoProvisioningAfterReconciled
				r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, msg)
				return common.NewChangeAfterInSync(msg)
			} else {
				logPtpInstance.Info(common.ProvisioningAllowedAfterReconciled)
			}
		}

		// Update PTP parameters associated with PTP instance
		if len(added) > 0 {
			new, err := r.ReconcileParamAdded(client, added, existing)
			if err != nil {
				return err
			}

			*existing = *new
		}

		if len(removed) > 0 {
			new2, err2 := r.ReconcileParamRemoved(client, removed, existing)
			if err2 != nil {
				return err2
			}

			*existing = *new2
		}

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
			"ptp instance has been updated")
	}

	return nil
}

// ReconcileResource interacts with the system API in order to reconcile the
// state of a ptp instance with the state stored in the k8s database.
func (r *PtpInstanceReconciler) ReconcileResource(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInstance) error {
	found, err := r.FindExistingPTPInstance(client, instance)
	if err != nil {
		if !instance.DeletionTimestamp.IsZero() {
			if utils.ContainsString(instance.ObjectMeta.Finalizers, PtpInstanceFinalizerName) {

				// Remove the PtpInstance finalizer
				r.removePtpInstanceFinalizer(instance)
			}
		}
		return err
	}

	if !instance.DeletionTimestamp.IsZero() {
		err = r.ReconciledDeleted(client, instance, found)

	} else {
		if found == nil {
			found, err = r.ReconcileNew(client, instance)
		} else {
			err = r.ReconcileUpdated(client, instance, found)
		}

		inSync := err == nil

		if instance.Status.InSync != inSync {
			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "synchronization has changed to: %t", inSync)
		}

		if r.statusUpdateRequired(instance, found, inSync) {
			// update the resource status to link it to the system object.
			logPtpInstance.Info("updating PTP instance", "status", instance.Status)

			err2 := r.Client.Status().Update(context.TODO(), instance)
			if err2 != nil {
				err2 = perrors.Wrapf(err2, "failed to update status: %s",
					instance.Name)
				return err2
			}
		}
	}

	return err
}

// StopAfterInSync determines whether the reconciler should continue processing
// change requests after the configuration has been reconciled a first time.
func (r *PtpInstanceReconciler) StopAfterInSync() bool {
	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return utils.GetReconcilerOptionBool(utils.PTPInstance, utils.StopAfterInSync, true)
}

// Update ReconcileAfterInSync in instance
// ReconcileAfterInSync value will be:
// "true"  if deploymentScope is "principal" because it is day 2 operation (update configuration)
// "false" if deploymentScope is "bootstrap"
// Then reflect these values to cluster object
// It is expected that instance.Status.Deployment scope is already updated by
// UpdateDeploymentScope at this point.
func (r *PtpInstanceReconciler) UpdateConfigStatus(instance *starlingxv1.PtpInstance, ns string) (err error) {
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

		// Set default value for StrategyRequired
		if instance.Status.StrategyRequired == "" {
			instance.Status.StrategyRequired = cloudManager.StrategyNotRequired
		}

		if instance.Status.ObservedGeneration != instance.ObjectMeta.Generation {
			// Configuration is updated
			if instance.Status.ObservedGeneration == 0 &&
				instance.Status.Reconciled {
				// Case: DM upgrade in reconciled node
				instance.Status.ConfigurationUpdated = false
			} else {
				// Case: Fresh install or Day-2 operation
				instance.Status.ConfigurationUpdated = true
				if instance.Status.DeploymentScope == cloudManager.ScopePrincipal {
					instance.Status.Reconciled = false
					// Update strategy required status for strategy monitor
					r.CloudManager.UpdateConfigVersion()
					r.CloudManager.SetResourceInfo(cloudManager.ResourcePtpinstance, "", instance.Name, instance.Status.Reconciled, cloudManager.StrategyNotRequired)
				}
			}
			instance.Status.ObservedGeneration = instance.ObjectMeta.Generation
			// Reset strategy when new configuration is applied
			instance.Status.StrategyRequired = cloudManager.StrategyNotRequired
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
func (r *PtpInstanceReconciler) UpdateStatusForFactoryInstall(
	ns string,
	instance *starlingxv1.PtpInstance,
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

// Reconcile reads that state of the cluster for a PTPInstance object and makes
// changes based on the state read and what is in the PtpInstance.Spec
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=ptpinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=ptpinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=ptpinstances/finalizers,verbs=update
func (r *PtpInstanceReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	savedLog := logPtpInstance
	logPtpInstance = logPtpInstance.WithName(request.NamespacedName.String())
	defer func() { logPtpInstance = savedLog }()

	logPtpInstance.Info("PTP instance reconcile called")

	// Fetch the PTPInstance instance
	instance := &starlingxv1.PtpInstance{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically
			// garbage collected. For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}

		logPtpInstance.Error(err, "unable to read object: %v", request)
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Restore the Ptp instance status
	if r.checkRestoreInProgress(instance) {
		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "Restoring '%s' Ptp Instance resource status without doing actual reconciliation", instance.Name)
		if err := r.RestorePtpInstanceStatus(instance); err != nil {
			return reconcile.Result{}, err
		}
		if err := r.ClearRestoreInProgress(instance); err != nil {
			return reconcile.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if err, _ := r.UpdateDeploymentScope(instance); err != nil {
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
	logPtpInstance.V(2).Info("before UpdateConfigStatus", "instance", instance)
	err = r.UpdateConfigStatus(instance, request.Namespace)
	if err != nil {
		logPtpInstance.Error(err, "unable to update ReconciledAfterInSync or ObservedGeneration.")
		return reconcile.Result{}, err
	}
	logPtpInstance.V(2).Info("after UpdateConfigStatus", "instance", instance)

	if instance.DeletionTimestamp.IsZero() {
		// Ensure that the object has a finalizer setup as a pre-delete hook so
		// that we can delete any system resources that we previously added.
		if !utils.ContainsString(instance.ObjectMeta.Finalizers, PtpInstanceFinalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, PtpInstanceFinalizerName)
			if err := r.Client.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}

			// Might as well return immediately as the update is going to cause
			// another reconcile event for this resource and we don't want to
			// access the system API more than necessary.
			return reconcile.Result{}, nil
		}
	}

	if !utils.IsReconcilerEnabled(utils.PTPInstance) {
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

	err = r.ReconcileResource(platformClient, instance)
	if err != nil {
		return r.ReconcilerErrorHandler.HandleReconcilerError(request, err)
	}

	return ctrl.Result{}, nil
}

// UpdateDeploymentScope function is used to update the deployment scope for PtpInstance.
func (r *PtpInstanceReconciler) UpdateDeploymentScope(instance *starlingxv1.PtpInstance) (error, bool) {
	updated, err := common.UpdateDeploymentScope(r.Client, instance)
	if err != nil {
		logPtpInstance.Error(err, "failed to update deploymentScope")
		return err, false
	}
	return nil, updated
}

// SetupWithManager sets up the controller with the Manager.
func (r *PtpInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	tMgr := cloudManager.GetInstance(mgr)
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.CloudManager = tMgr
	r.ReconcilerErrorHandler = &common.ErrorHandler{
		CloudManager: tMgr,
		Logger:       logPtpInstance}
	r.ReconcilerEventLogger = &common.EventLogger{
		EventRecorder: mgr.GetEventRecorderFor(PtpInstanceControllerName),
		Logger:        logPtpInstance}
	return ctrl.NewControllerManagedBy(mgr).
		For(&starlingxv1.PtpInstance{}).
		Complete(r)
}

// Verify whether we have annotation restore-in-progress
func (r *PtpInstanceReconciler) checkRestoreInProgress(instance *starlingxv1.PtpInstance) bool {
	restoreInProgress, ok := instance.Annotations[cloudManager.RestoreInProgress]
	if ok && restoreInProgress != "" {
		return true
	}
	return false
}

// Update status
func (r *PtpInstanceReconciler) RestorePtpInstanceStatus(instance *starlingxv1.PtpInstance) error {
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
					"Failed to update ptp instance status while restoring '%s' resource. Error: %s",
					instance.Name,
					err)
				return common.NewResourceStatusDependency(log_err_msg)
			} else {
				StatusUpdate := fmt.Sprintf("Status updated for PtpInstance resource '%s' during restore with following values: Reconciled=%t InSync=%t DeploymentScope=%s",
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
func (r *PtpInstanceReconciler) ClearRestoreInProgress(instance *starlingxv1.PtpInstance) error {
	delete(instance.Annotations, cloudManager.RestoreInProgress)
	if !utils.ContainsString(instance.ObjectMeta.Finalizers, PtpInstanceFinalizerName) {
		instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, PtpInstanceFinalizerName)
	}
	err := r.Client.Update(context.TODO(), instance)
	if err != nil {
		return common.NewResourceStatusDependency(fmt.Sprintf("Failed to update '%s' ptp instance resource after removing '%s' annotation during restoration.",
			instance.Name, cloudManager.RestoreInProgress))

	}
	return nil
}
