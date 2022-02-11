/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package ptpinterface

import (
	"context"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinstances"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinterfaces"
	perrors "github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/pkg/apis/starlingx/v1"
	utils "github.com/wind-river/cloud-platform-deployment-manager/pkg/common"
	"github.com/wind-river/cloud-platform-deployment-manager/pkg/config"
	"github.com/wind-river/cloud-platform-deployment-manager/pkg/controller/common"
	cloudManager "github.com/wind-river/cloud-platform-deployment-manager/pkg/manager"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller").WithName("ptpinterface")

const ControllerName = "ptpinterface-controller"

const FinalizerName = "ptpinterface.finalizers.windriver.com"

// Add creates a new PTPInterface Controller and adds it to the Manager with default RBAC.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	tMgr := cloudManager.GetInstance(mgr)
	return &ReconcilePtpInterface{
		Client:       mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		CloudManager: tMgr,
		ReconcilerErrorHandler: &common.ErrorHandler{
			CloudManager: tMgr,
			Logger:       log},
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

	// Watch for changes to PtpInterface
	err = c.Watch(&source.Kind{Type: &starlingxv1.PtpInterface{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcilePtpInterface{}

// ReconcileResource reconciles a PtpInterface object
type ReconcilePtpInterface struct {
	client.Client
	scheme *runtime.Scheme
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

func intefaceParameterUpdateRequired(instance *starlingxv1.PtpInterface, i *ptpinterfaces.PTPInterface) (added []string, removed []string, result bool) {
	configured := instance.Spec.InterfaceParameters
	current := i.Parameters
	result = false

	// Diff the lists to determine if changes need to be applied
	added, removed, _ = utils.ListDelta(current, configured)

	if len(added) > 0 || len(removed) > 0 {
		result = true
	}

	return added, removed, result
}

// ReconcileParamAdded is a method which handles adding new Parameters to
// associate with an existing PTP interface
func (r *ReconcilePtpInterface) ReconcileParamAdded(client *gophercloud.ServiceClient, params []string, i *ptpinterfaces.PTPInterface) (*ptpinterfaces.PTPInterface, error) {

	id := i.UUID
	for _, param := range params {
		opts := ptpinterfaces.PTPParamToPTPIntOpts{Parameter: &param}

		log.Info("adding ptp parameter", "opts", opts)

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
func (r *ReconcilePtpInterface) ReconcileParamRemoved(client *gophercloud.ServiceClient, params []string, i *ptpinterfaces.PTPInterface) (*ptpinterfaces.PTPInterface, error) {

	id := i.UUID
	for _, param := range params {
		opts := ptpinterfaces.PTPParamToPTPIntOpts{Parameter: &param}

		log.Info("adding ptp parameter", "opts", opts)

		new, err := ptpinterfaces.RemovePTPParamFromPTPInt(client, id, opts).Extract()

		if err != nil {
			return i, err
		}

		*i = *new
	}

	return i, nil
}

func (r *ReconcilePtpInterface) GetPTPInstanceUUIDByName(client *gophercloud.ServiceClient, name string) (string, error) {

	associatedInst, err := ptpinstances.Get(client, name).Extract()

	if err != nil {
		err = perrors.Wrapf(err, "Failed to get UUID of ptp instance: %s", name)
		return "", err
	}

	if associatedInst == nil {
		err = perrors.Wrapf(err, "No ptp instance provisioned as: %s", name)
		return "", err
	}

	return associatedInst.UUID, nil
}

// ReconcileNew is a method which handles reconciling a new data resource and
// creates the corresponding system resource thru the system API.
func (r *ReconcilePtpInterface) ReconcileNew(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInterface) (*ptpinterfaces.PTPInterface, error) {
	if instance.Status.Reconciled && r.StopAfterInSync() {
		// Do not process any further changes once we have reached a
		// synchronized state unless there is an annotation on the resource.
		if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
			msg := common.NoProvisioningAfterReconciled
			r.NormalEvent(instance, common.ResourceUpdated, msg)
			return nil, common.NewChangeAfterInSync(msg)
		} else {
			log.Info(common.ProvisioningAllowedAfterReconciled)
		}
	}

	// Get the UUID of the PTP instance that associated with this PTP interface
	uuid, err := r.GetPTPInstanceUUIDByName(client, instance.Spec.PtpInstance)
	if err != nil {
		return nil, err
	}

	// Create a new PTP instance
	opts := ptpinterfaces.PTPInterfaceOpts{
		Name:            &instance.Name,
		PTPInstanceUUID: &uuid,
	}

	log.Info("creating ptp interface", "opts", opts)

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

	r.NormalEvent(instance, common.ResourceCreated,
		"ptp interface has been created")

	return new, nil
}

// ReconciledDeleted is a method which handles reconciling a new data resource and
// creates the corresponding system resource thru the system API.
func (r *ReconcilePtpInterface) ReconciledDeleted(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInterface, i *ptpinterfaces.PTPInterface) error {
	if utils.ContainsString(instance.ObjectMeta.Finalizers, FinalizerName) {
		if i != nil {
			// Unless it was already deleted go ahead and attempt to delete it.
			err := ptpinterfaces.Delete(client, i.UUID).ExtractErr()
			if err != nil {
				if code, ok := err.(gophercloud.ErrDefault400); !ok {
					err = perrors.Wrap(err, "failed to delete ptp interface")
					return err
				} else if code.Actual == 400 {
					log.Info("PTP interface is still in use; deleting local resource anayway")
					// NOTE: there is no way to block the kubernetes delete beyond
					//  delaying it a little while we delete an external resource so
					//  since we can't fail this then log it and allow it to
					//  continue for now.
				} else {
					err = perrors.Wrap(err, "unexpected response code on PTP interface deletion")
					return err
				}
			}

			r.NormalEvent(instance, common.ResourceDeleted, "PTP interface has been deleted")
		}

		// Remove the finalizer so the kubernetes delete operation can continue.
		instance.ObjectMeta.Finalizers = utils.RemoveString(instance.ObjectMeta.Finalizers, FinalizerName)
		if err := r.Update(context.Background(), instance); err != nil {
			return err
		}

	}

	return nil
}

// statusUpdateRequired is a utility function which determines whether an update
// is required to the host status attribute.  Updating this unnecessarily
// will result in an infinite reconciliation loop.
func (r *ReconcilePtpInterface) statusUpdateRequired(instance *starlingxv1.PtpInterface, existing *ptpinterfaces.PTPInterface, inSync bool) (result bool) {
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
		result = true
	}

	return result
}

// FindExistingPTPInterface attempts to find the existing PTP interface referenced by
// the ID value stored in the status or to find another resource with a matching
// name.
func (r *ReconcilePtpInterface) FindExistingPTPInterface(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInterface) (found *ptpinterfaces.PTPInterface, err error) {
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
			log.Info("resource no longer exists", "id", *id)
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
					log.Info("found existing ptp interface", "uuid", result.UUID)
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
func (r *ReconcilePtpInterface) ReconcileUpdated(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInterface, existing *ptpinterfaces.PTPInterface) error {
	if ok := interfaceUpdateRequired(instance, existing); ok {
		// As there's not sysinv API to update the name and service type of a
		// PTP interface, delete the existing and create a new one.
		log.Info("deleting PTP interface", "status", instance.Status)
		err := r.ReconciledDeleted(client, instance, existing)
		if err != nil {
			log.Info("failed to delete PTP interface", "status", instance.Status)
		}

		log.Info("creating new PTP interface", "status", instance.Status)
		new, err2 := r.ReconcileNew(client, instance)

		if err2 != nil {
			return err2
		}

		*existing = *new

		r.NormalEvent(instance, common.ResourceUpdated,
			"ptp interface has been updated")

	} else if added, removed, required := intefaceParameterUpdateRequired(instance, existing); required {
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

		r.NormalEvent(instance, common.ResourceUpdated,
			"ptp interface has been updated")
	}

	return nil
}

// ReconcileResource interacts with the system API in order to reconcile the
// state of a data network with the state stored in the k8s database.
func (r *ReconcilePtpInterface) ReconcileResource(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInterface) error {
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
			r.NormalEvent(instance, common.ResourceUpdated, "synchronization has changed to: %t", inSync)
		}

		if r.statusUpdateRequired(instance, found, inSync) {
			// update the resource status to link it to the system object.
			log.Info("updating PTP interface", "status", instance.Status)

			err2 := r.Status().Update(context.TODO(), instance)
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
func (r *ReconcilePtpInterface) StopAfterInSync() bool {
	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return config.GetReconcilerOptionBool(config.PTPInterface, config.StopAfterInSync, true)
}

// Reconcile reads that state of the cluster for a PTPInterface object and makes
// changes based on the state read and what is in the PtpInterface.Spec
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=ptpinterfaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=ptpinterfaces/status,verbs=get;update;patch
func (r *ReconcilePtpInterface) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	savedLog := log
	log = log.WithName(request.NamespacedName.String())
	defer func() { log = savedLog }()

	// Fetch the PTPInterface instance
	instance := &starlingxv1.PtpInterface{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically
			// garbage collected. For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}

		log.Error(err, "unable to read object: %v", request)
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.DeletionTimestamp.IsZero() {
		// Ensure that the object has a finalizer setup as a pre-delete hook so
		// that we can delete any system resources that we previously added.
		if !utils.ContainsString(instance.ObjectMeta.Finalizers, FinalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, FinalizerName)
			if err := r.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}

			// Might as well return immediately as the update is going to cause
			// another reconcile event for this resource and we don't want to
			// access the system API more than necessary.
			return reconcile.Result{}, nil
		}
	}

	if !config.IsReconcilerEnabled(config.PTPInterface) {
		return reconcile.Result{}, nil
	}

	platformClient := r.GetPlatformClient(request.Namespace)
	if platformClient == nil {
		// The client has not been authenticated by the system controller so
		// wait.
		r.WarningEvent(instance, common.ResourceDependency,
			"waiting for platform client creation")
		return common.RetryMissingClient, nil
	}

	if !r.GetSystemReady(request.Namespace) {
		r.WarningEvent(instance, common.ResourceDependency,
			"waiting for system reconciliation")
		return common.RetrySystemNotReady, nil
	}

	err = r.ReconcileResource(platformClient, instance)
	if err != nil {
		return r.HandleReconcilerError(request, err)
	}

	return reconcile.Result{}, nil
}
