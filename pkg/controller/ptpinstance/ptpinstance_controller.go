/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package ptpinstance

import (
	"context"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinstances"
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

var log = logf.Log.WithName("controller").WithName("ptpinstance")

const ControllerName = "ptpinstance-controller"

const FinalizerName = "ptpinstance.finalizers.windriver.com"

// Add creates a new PTPInstance Controller and adds it to the Manager with default RBAC.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	tMgr := cloudManager.GetInstance(mgr)
	return &ReconcilePtpInstance{
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

	// Watch for changes to PtpInstance
	err = c.Watch(&source.Kind{Type: &starlingxv1.PtpInstance{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcilePtpInstance{}

// ReconcilePtpInstance reconciles a PtpInstance object
type ReconcilePtpInstance struct {
	client.Client
	scheme *runtime.Scheme
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

func instanceParameterUpdateRequired(instance *starlingxv1.PtpInstance, i *ptpinstances.PTPInstance) (added []string, removed []string, result bool) {
	configured := instance.Spec.InstanceParameters
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
// associate with an existing PTP instance
func (r *ReconcilePtpInstance) ReconcileParamAdded(client *gophercloud.ServiceClient, params []string, i *ptpinstances.PTPInstance) (*ptpinstances.PTPInstance, error) {

	id := i.UUID
	for _, param := range params {
		opts := ptpinstances.PTPParamToPTPInstOpts{Parameter: &param}

		log.Info("adding ptp parameter", "opts", opts)

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
func (r *ReconcilePtpInstance) ReconcileParamRemoved(client *gophercloud.ServiceClient, params []string, i *ptpinstances.PTPInstance) (*ptpinstances.PTPInstance, error) {

	id := i.UUID
	for _, param := range params {
		opts := ptpinstances.PTPParamToPTPInstOpts{Parameter: &param}

		log.Info("removing ptp parameter", "opts", opts)

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
func (r *ReconcilePtpInstance) ReconcileNew(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInstance) (*ptpinstances.PTPInstance, error) {
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

	// Create a new PTP instance
	opts := ptpinstances.PTPInstanceOpts{
		Name:    &instance.Name,
		Service: &instance.Spec.Service,
	}

	log.Info("creating ptp instance", "opts", opts)

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

	r.NormalEvent(instance, common.ResourceCreated,
		"ptp instance has been created")

	return new, nil
}

// ReconciledDeleted is a method which handles reconciling a new data resource and
// creates the corresponding system resource thru the system API.
func (r *ReconcilePtpInstance) ReconciledDeleted(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInstance, i *ptpinstances.PTPInstance) error {
	if utils.ContainsString(instance.ObjectMeta.Finalizers, FinalizerName) {
		if i != nil {
			// Unless it was already deleted go ahead and attempt to delete it.
			err := ptpinstances.Delete(client, i.UUID).ExtractErr()
			if err != nil {
				if code, ok := err.(gophercloud.ErrDefault400); !ok {
					err = perrors.Wrap(err, "failed to delete ptp instance")
					return err
				} else if code.Actual == 400 {
					log.Info("PTP instance is still in use; deleting local resource anayway")
					// NOTE: there is no way to block the kubernetes delete beyond
					//  delaying it a little while we delete an external resource so
					//  since we can't fail this then log it and allow it to
					//  continue for now.
				} else {
					err := perrors.Wrap(err, "unexpected response code on PTP instance deletion")
					return err
				}
			}

			r.NormalEvent(instance, common.ResourceDeleted, "PTP instance has been deleted")
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
func (r *ReconcilePtpInstance) statusUpdateRequired(instance *starlingxv1.PtpInstance, existing *ptpinstances.PTPInstance, inSync bool) (result bool) {
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

// FindExistingPTPInstance attempts to find the existing PTP instance referenced by
// the ID value stored in the status or to find another resource with a matching
// name.
func (r *ReconcilePtpInstance) FindExistingPTPInstance(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInstance) (found *ptpinstances.PTPInstance, err error) {
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
			log.Info("resource no longer exists", "id", *id)
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
					log.Info("found existing ptp instance", "uuid", result.UUID)
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
func (r *ReconcilePtpInstance) ReconcileUpdated(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInstance, existing *ptpinstances.PTPInstance) error {
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
		log.Info("deleting PTP instance", "status", instance.Status)
		err := r.ReconciledDeleted(client, instance, existing)
		if err != nil {
			log.Info("failed to delete PTP instance", "status", instance.Status)
		}

		log.Info("creating new PTP instance", "status", instance.Status)
		new, err2 := r.ReconcileNew(client, instance)

		if err2 != nil {
			return err2
		}

		*existing = *new

		r.NormalEvent(instance, common.ResourceUpdated,
			"ptp instance has been updated")

	} else if added, removed, required := instanceParameterUpdateRequired(instance, existing); required {
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

		r.NormalEvent(instance, common.ResourceUpdated,
			"ptp instance has been updated")
	}

	return nil
}

// ReconcileResource interacts with the system API in order to reconcile the
// state of a ptp instance with the state stored in the k8s database.
func (r *ReconcilePtpInstance) ReconcileResource(client *gophercloud.ServiceClient, instance *starlingxv1.PtpInstance) error {
	found, err := r.FindExistingPTPInstance(client, instance)

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
			log.Info("updating PTP instance", "status", instance.Status)

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
func (r *ReconcilePtpInstance) StopAfterInSync() bool {
	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return config.GetReconcilerOptionBool(config.PTPInstance, config.StopAfterInSync, true)
}

// Reconcile reads that state of the cluster for a PTPInstance object and makes
// changes based on the state read and what is in the PtpInstance.Spec
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=ptpinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=ptpinstances/status,verbs=get;update;patch
func (r *ReconcilePtpInstance) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	savedLog := log
	log = log.WithName(request.NamespacedName.String())
	defer func() { log = savedLog }()

	log.Info("PTP instance reconcile called")

	// Fetch the PTPInstance instance
	instance := &starlingxv1.PtpInstance{}
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

	if !config.IsReconcilerEnabled(config.PTPInstance) {
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
