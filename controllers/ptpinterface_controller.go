/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022-2023 Wind River Systems, Inc. */

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinstances"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinterfaces"
	perrors "github.com/pkg/errors"
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

var logPtpInterface = log.Log.WithName("controller").WithName("ptpinterface")

const PtpInterfaceControllerName = "ptpinterface-controller"

const PtpInterfaceFinalizerName = "ptpinterface.finalizers.windriver.com"

// PtpInterfaceReconciler reconciles a PtpInterface object
type PtpInterfaceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	cloudManager.CloudManager
	common.ReconcilerErrorHandler
	common.ReconcilerEventLogger
}

func interfaceUpdateRequired(instance *starlingxv1.PtpInterface, i *ptpinterfaces.PTPInterface) (result bool) {
	if instance.Name != i.Name {
		result = true
	}

	spec := instance.Spec
	if spec.PtpInstance != i.PTPInstanceName {
		result = true
	}

	return result
}

func intefaceParameterUpdateRequired(instance *starlingxv1.PtpInterface, i *ptpinterfaces.PTPInterface, r *PtpInterfaceReconciler) (added []string, removed []string, result bool) {
	configured := instance.Spec.InterfaceParameters
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
		logPtpInterface.Info(fmt.Sprintf("delta configuration:%s\n", deltaString))
	}
	instance.Status.Delta = deltaString
	err := r.Client.Status().Update(context.TODO(), instance)
	if err != nil {
		logPtpInterface.Info(fmt.Sprintf("failed to update status:  %s\n", err))
	}

	return added, removed, result
}

// ReconcileParamAdded is a method which handles adding new Parameters to
// associate with an existing PTP interface
func (r *PtpInterfaceReconciler) ReconcileParamAdded(client *gophercloud.ServiceClient, params []string, i *ptpinterfaces.PTPInterface) (*ptpinterfaces.PTPInterface, error) {

	id := i.UUID
	for _, param := range params {
		opts := ptpinterfaces.PTPParamToPTPIntOpts{Parameter: &param}

		logPtpInterface.Info("adding ptp parameter", "opts", opts)

		new, err := ptpinterfaces.AddPTPParamToPTPInt(client, id, opts).Extract()

		if err != nil {
			return i, err
		}

		*i = *new
	}

	return i, nil
}

// ReconcileParamRemoved is a method which handles removing Parameters from
// an existing PTP
func (r *PtpInterfaceReconciler) ReconcileParamRemoved(client *gophercloud.ServiceClient, params []string, i *ptpinterfaces.PTPInterface) (*ptpinterfaces.PTPInterface, error) {

	id := i.UUID
	for _, param := range params {
		opts := ptpinterfaces.PTPParamToPTPIntOpts{Parameter: &param}

		logPtpInterface.Info("adding ptp parameter", "opts", opts)

		new, err := ptpinterfaces.RemovePTPParamFromPTPInt(client, id, opts).Extract()

		if err != nil {
			return i, err
		}

		*i = *new
	}

	return i, nil
}

// findPTPInstanceByName is to search for a PTP instance by its name,
// this instance may or may not associate with the current host.
func findPTPInstanceByName(client *gophercloud.ServiceClient, name string) (*ptpinstances.PTPInstance, error) {
	founds, err := ptpinstances.ListPTPInstances(client)
	if err != nil {
		return nil, err
	}
	for _, found := range founds {
		if found.Name == name {
			return &found, nil
		}
	}
	return nil, nil
}

// ReconcileNew is a method which handles reconciling a new data resource and
// creates the corresponding system resource thru the system API.
func (r *PtpInterfaceReconciler) ReconcileNew(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInterface) (*ptpinterfaces.PTPInterface, error) {
	if instance.Status.Reconciled && r.StopAfterInSync() {
		// Do not process any further changes once we have reached a
		// synchronized state unless there is an annotation on the resource.
		if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
			msg := common.NoProvisioningAfterReconciled
			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, msg)
			return nil, common.NewChangeAfterInSync(msg)
		} else {
			logPtpInterface.Info(common.ProvisioningAllowedAfterReconciled)
		}
	}

	// Get the UUID of the PTP instance that associated with this PTP interface
	found, err := findPTPInstanceByName(client, instance.Spec.PtpInstance)
	if err != nil {
		err = perrors.Wrapf(err, "failed to find PTP instance for PTP interface: %s", instance.Name)
		return nil, err
	}

	if found == nil {
		return nil, common.NewResourceStatusDependency("PTP instance is not created, waiting for the creation")
	}

	// Create a new PTP interface
	opts := ptpinterfaces.PTPInterfaceOpts{
		Name:            &instance.Name,
		PTPInstanceUUID: &found.UUID,
	}

	logPtpInterface.Info("creating ptp interface", "opts", opts)

	new, err := ptpinterfaces.Create(client, opts).Extract()
	if err != nil {
		err = perrors.Wrapf(err, "failed to create: %s", common.FormatStruct(opts))
		return nil, err
	}

	new, err = r.ReconcileParamAdded(client, instance.Spec.InterfaceParameters, new)
	if err != nil {
		err = perrors.Wrapf(err, "failed to add parameter to: %s", common.FormatStruct(opts))
		return nil, err
	}

	r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceCreated,
		"ptp interface has been created")

	return new, nil
}

// ReconciledDeleted is a method which handles reconciling a new data resource and
// creates the corresponding system resource thru the system API.
func (r *PtpInterfaceReconciler) ReconciledDeleted(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInterface, i *ptpinterfaces.PTPInterface) error {
	if utils.ContainsString(instance.ObjectMeta.Finalizers, PtpInterfaceFinalizerName) {
		if i != nil {
			// Unless it was already deleted go ahead and attempt to delete it.
			err := ptpinterfaces.Delete(client, i.UUID).ExtractErr()
			if err != nil {
				if code, ok := err.(gophercloud.ErrDefault400); !ok {
					err = perrors.Wrap(err, "failed to delete ptp interface")
					return err
				} else if code.Actual == 400 {
					logPtpInterface.Info("PTP interface is still in use; deleting local resource anayway")
					// NOTE: there is no way to block the kubernetes delete beyond
					//  delaying it a little while we delete an external resource so
					//  since we can't fail this then log it and allow it to
					//  continue for now.
				} else {
					err = perrors.Wrap(err, "unexpected response code on PTP interface deletion")
					return err
				}
			}

			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceDeleted, "PTP interface has been deleted")
		}

		// Remove the finalizer so the kubernetes delete operation can continue.
		instance.ObjectMeta.Finalizers = utils.RemoveString(instance.ObjectMeta.Finalizers, PtpInterfaceFinalizerName)
		if err := r.Client.Update(context.Background(), instance); err != nil {
			return err
		}

	}

	return nil
}

// statusUpdateRequired is a utility function which determines whether an update
// is required to the host status attribute.  Updating this unnecessarily
// will result in an infinite reconciliation loop.
func (r *PtpInterfaceReconciler) statusUpdateRequired(instance *starlingxv1.PtpInterface, existing *ptpinterfaces.PTPInterface, inSync bool) (result bool) {
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
			r.CloudManager.SetResourceInfo(cloudManager.ResourcePtpinterface, "", instance.Name, status.Reconciled, status.StrategyRequired)
		}
		result = true
	}

	return result
}

// FindExistingPTPInterface attempts to find the existing PTP interface referenced by
// the ID value stored in the status or to find another resource with a matching
// name.
func (r *PtpInterfaceReconciler) FindExistingPTPInterface(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInterface) (found *ptpinterfaces.PTPInterface, err error) {
	id := instance.Status.ID
	if id != nil {
		// This PTP interface was previously provisioned.
		found, err = ptpinterfaces.Get(client, *id).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); !ok {
				err = perrors.Wrapf(err, "failed to get: %s", *id)
				return nil, err
			}

			// The resource may have been deleted by the system or operator
			// therefore continue and attempt to recreate it.
			logPtpInterface.Info("resource no longer exists", "id", *id)
			return nil, nil
		}

	} else {
		// This PTP interface needs to be provisioned if it doesn't already exist.
		results, err := ptpinterfaces.ListPTPInterfaces(client)
		if err != nil {
			err = perrors.Wrap(err, "failed to list")
			return nil, err
		}

		for _, result := range results {
			if result.Name == instance.Name {
				if result.PTPInstanceName == instance.Spec.PtpInstance {
					logPtpInterface.Info("found existing ptp interface", "uuid", result.UUID)
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
func (r *PtpInterfaceReconciler) ReconcileUpdated(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInterface, existing *ptpinterfaces.PTPInterface) error {
	if ok := interfaceUpdateRequired(instance, existing); ok {
		if instance.Status.Reconciled && r.StopAfterInSync() {
			// Do not process any further changes once we have reached a
			// synchronized state unless there is an annotation on the resource.
			if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
				msg := common.NoProvisioningAfterReconciled
				r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, msg)
				return common.NewChangeAfterInSync(msg)
			} else {
				logPtpInterface.Info(common.ProvisioningAllowedAfterReconciled)
			}
		}
		// As there's not sysinv API to update the name and service type of a
		// PTP interface, delete the existing and create a new one.
		logPtpInterface.Info("deleting PTP interface", "status", instance.Status)
		err := r.ReconciledDeleted(client, instance, existing)
		if err != nil {
			logPtpInterface.Info("failed to delete PTP interface", "status", instance.Status)
		}

		logPtpInterface.Info("creating new PTP interface", "status", instance.Status)
		new, err2 := r.ReconcileNew(client, instance)

		if err2 != nil {
			return err2
		}

		*existing = *new

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
			"ptp interface has been updated")

	} else if added, removed, required := intefaceParameterUpdateRequired(instance, existing, r); required {
		if instance.Status.Reconciled && r.StopAfterInSync() {
			// Do not process any further changes once we have reached a
			// synchronized state unless there is an annotation on the resource.
			if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
				msg := common.NoProvisioningAfterReconciled
				r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, msg)
				return common.NewChangeAfterInSync(msg)
			} else {
				logPtpInterface.Info(common.ProvisioningAllowedAfterReconciled)
			}
		}

		// Update PTP parameters associated with PTP interface
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
			"ptp interface has been updated")
	}

	return nil
}

// ReconcileResource interacts with the system API in order to reconcile the
// state of a PTP interface with the state stored in the k8s database.
func (r *PtpInterfaceReconciler) ReconcileResource(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInterface) error {
	logPtpInterface.Info("ptp interface reconcile resource")
	found, err := r.FindExistingPTPInterface(client, instance)
	if err != nil {
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
			logPtpInterface.Info("updating PTP interface", "status", instance.Status)

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
func (r *PtpInterfaceReconciler) StopAfterInSync() bool {
	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return utils.GetReconcilerOptionBool(utils.PTPInterface, utils.StopAfterInSync, true)
}

// Obtain deploymentScope value from configuration
// Taking this value from annotation in instacne
// (It seems Client.Get does not update Status value from configuration)
// "bootstrap" if "bootstrap" in configuration or deploymentScope not specified
// "principal" if "principal" in configuration
func (r *PtpInterfaceReconciler) GetScopeConfig(instance *starlingxv1.PtpInterface) (scope string, err error) {
	// Set default value for deployment scope
	deploymentScope := cloudManager.ScopeBootstrap
	// Set DeploymentScope from configuration
	annotation := instance.GetObjectMeta().GetAnnotations()
	if annotation != nil {
		config, ok := annotation["kubectl.kubernetes.io/last-applied-configuration"]
		if ok {
			status_config := &starlingxv1.Host{}
			err := json.Unmarshal([]byte(config), &status_config)
			if err == nil {
				if status_config.Status.DeploymentScope != "" {
					switch scope := status_config.Status.DeploymentScope; scope {
					case cloudManager.ScopeBootstrap:
						deploymentScope = scope
					case cloudManager.ScopePrincipal:
						deploymentScope = scope
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
// Then reflrect these values to cluster object
func (r *PtpInterfaceReconciler) UpdateConfigStatus(instance *starlingxv1.PtpInterface) (err error) {
	deploymentScope, err := r.GetScopeConfig(instance)
	if err != nil {
		return err
	}
	logPtpInterface.V(2).Info("deploymentScope in configuration", "deploymentScope", deploymentScope)

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

	// Update annotation
	err = r.Client.Update(context.TODO(), instance)
	if err != nil {
		err = perrors.Wrapf(err, "failed to update profile annotation ReconcileAfterInSync")
		return err
	}

	// Update scope status
	instance.Status.DeploymentScope = deploymentScope

	// Set default value for StrategyRequired
	if instance.Status.StrategyRequired == "" {
		instance.Status.StrategyRequired = cloudManager.StrategyNotRequired
	}

	// Check configration is updated
	if instance.Status.ObservedGeneration != instance.ObjectMeta.Generation {
		if instance.Status.ObservedGeneration == 0 &&
			instance.Status.Reconciled {
			// Case: DM upgrade in reconceiled node
			instance.Status.ConfigurationUpdated = false
		} else {
			// Case: Fresh install or Day-2 operation
			instance.Status.ConfigurationUpdated = true
			if instance.Status.DeploymentScope == cloudManager.ScopePrincipal {
				instance.Status.Reconciled = false
				// Update storategy required status for strategy monitor
				r.CloudManager.UpdateConfigVersion()
				r.CloudManager.SetResourceInfo(cloudManager.ResourcePtpinterface, "", instance.Name, instance.Status.Reconciled, cloudManager.StrategyNotRequired)
			}
		}
		instance.Status.ObservedGeneration = instance.ObjectMeta.Generation
		// Reset strategy when new configration is applied
		instance.Status.StrategyRequired = cloudManager.StrategyNotRequired
	}

	err = r.Client.Status().Update(context.TODO(), instance)
	if err != nil {
		err = perrors.Wrapf(err, "failed to update status: %s",
			common.FormatStruct(instance.Status))
		return err
	}

	return nil
}

// Reconcile reads that state of the cluster for a PTPInterface object and makes
// changes based on the state read and what is in the PtpInterface.Spec
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=ptpinterfaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=ptpinterfaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=ptpinterfaces/finalizers,verbs=update
func (r *PtpInterfaceReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	savedLog := logPtpInterface
	logPtpInterface = logPtpInterface.WithName(request.NamespacedName.String())
	defer func() { logPtpInterface = savedLog }()

	logPtpInterface.Info("PTP interface reconcile called")

	// Fetch the PTPInterface instance
	instance := &starlingxv1.PtpInterface{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically
			// garbage collected. For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}

		logPtpInterface.Error(err, "unable to read object: %v", request)
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Update scope from configuration
	logPtpInterface.V(2).Info("before UpdateConfigStatus", "instance", instance)
	err = r.UpdateConfigStatus(instance)
	if err != nil {
		logPtpInterface.Error(err, "unable to update scope")
		return reconcile.Result{}, err
	}
	logPtpInterface.V(2).Info("after UpdateConfigStatus", "instance", instance)

	if instance.DeletionTimestamp.IsZero() {
		// Ensure that the object has a finalizer setup as a pre-delete hook so
		// that we can delete any system resources that we previously added.
		if !utils.ContainsString(instance.ObjectMeta.Finalizers, PtpInterfaceFinalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, PtpInterfaceFinalizerName)
			if err := r.Client.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}

			// Might as well return immediately as the update is going to cause
			// another reconcile event for this resource and we don't want to
			// access the system API more than necessary.
			return reconcile.Result{}, nil
		}
	}

	if !utils.IsReconcilerEnabled(utils.PTPInterface) {
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

// SetupWithManager sets up the controller with the Manager.
func (r *PtpInterfaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	tMgr := cloudManager.GetInstance(mgr)
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.CloudManager = tMgr
	r.ReconcilerErrorHandler = &common.ErrorHandler{
		CloudManager: tMgr,
		Logger:       logPtpInterface}
	r.ReconcilerEventLogger = &common.EventLogger{
		EventRecorder: mgr.GetEventRecorderFor(PtpInterfaceControllerName),
		Logger:        logPtpInterface}
	return ctrl.NewControllerManagedBy(mgr).
		For(&starlingxv1.PtpInterface{}).
		Complete(r)
}
