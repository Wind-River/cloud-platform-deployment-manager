/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024 Wind River Systems, Inc. */

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers/common"
	cloudManager "github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

// ReconcileResource interacts with the system API in order to reconcile the
// state of a data network with the state stored in the k8s database.
// TODO(sriram-gn): Fetch active controller's host instance and set
// Reconciled/Insync to false only if Generation != ObservedGeneration.
func (r *AddressPoolReconciler) ReconcileResource(client *gophercloud.ServiceClient, instance *starlingxv1.AddressPool, request_namespace string) (err error) {
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
		instance.ObjectMeta.Finalizers = utils.RemoveString(instance.ObjectMeta.Finalizers, AddressPoolFinalizerName)
		if err := r.Client.Update(context.Background(), instance); err != nil {
			return err
		}
	}

	return err
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

	err = r.ReconcileResource(platformClient, instance, request.NamespacedName.Namespace)
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
