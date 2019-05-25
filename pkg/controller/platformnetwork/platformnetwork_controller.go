/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package platformnetwork

import (
	"context"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresspools"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/networks"
	perrors "github.com/pkg/errors"
	starlingxv1beta1 "github.com/wind-river/titanium-deployment-manager/pkg/apis/starlingx/v1beta1"
	"github.com/wind-river/titanium-deployment-manager/pkg/controller/common"
	titaniumManager "github.com/wind-river/titanium-deployment-manager/pkg/manager"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
)

var log = logf.Log.WithName("controller").WithName("platformnetwork")

const ControllerName = "platformnetwork-controller"

const FinalizerName = "platformnetwork.finalizers.windriver.com"

const (
	// Defines commonly used network allocation type values.
	AllocationTypeDynamic = "dynamic"
)

// Add creates a new PlatformNetwork Controller and adds it to the Manager with
// default RBAC. The Manager will set fields on the Controller and Start it when
// the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	tMgr := titaniumManager.GetInstance(mgr)
	return &ReconcilePlatformNetwork{
		Client:          mgr.GetClient(),
		scheme:          mgr.GetScheme(),
		TitaniumManager: tMgr,
		ReconcilerErrorHandler: &common.ErrorHandler{
			TitaniumManager: tMgr,
			Logger:          log},
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

	// Watch for changes to PlatformNetwork
	err = c.Watch(&source.Kind{Type: &starlingxv1beta1.PlatformNetwork{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcilePlatformNetwork{}

// ReconcileResource reconciles a PlatformNetwork object
type ReconcilePlatformNetwork struct {
	client.Client
	scheme *runtime.Scheme
	titaniumManager.TitaniumManager
	common.ReconcilerErrorHandler
	common.ReconcilerEventLogger
}

// makeRangeArray converts an array of range structs to an array of arrays where
// the inner array contains two elements.  The first element is the range start
// address and the second element is the range end address.  This is to align
// with the system API formatting which represents a pair as an array of two
// elements.
func makeRangeArray(ranges []starlingxv1beta1.AllocationRange) [][]string {
	result := make([][]string, len(ranges))
	for index, r := range ranges {
		result[index] = []string{r.Start, r.End}
	}

	return result
}

// compareRangeArrays compares two range arrays and returns true if they are
// equal.
func compareRangeArrays(x, y [][]string) bool {
	if len(x) != len(y) {
		return false
	}

	count := 0
	for _, o := range x {
		for _, i := range y {
			if strings.EqualFold(o[0], i[0]) && strings.EqualFold(o[1], i[1]) {
				count++
			}
		}
	}

	if len(x) != count {
		return false
	}

	return true
}

// poolUpdateRequired determines whether a system address pool resource must
// be updated to align with the stored value.  Only the updated fields are
//// include in the request options to minimum churn and to ease debugging.
func poolUpdateRequired(instance *starlingxv1beta1.PlatformNetwork, p *addresspools.AddressPool) (opts addresspools.AddressPoolOpts, result bool) {
	if instance.Name != p.Name {
		opts.Name = &instance.Name
		result = true
	}

	spec := instance.Spec
	if !strings.EqualFold(spec.Subnet, p.Network) {
		opts.Network = &spec.Subnet
		result = true
	}

	if spec.Prefix != p.Prefix {
		opts.Prefix = &spec.Prefix
		result = true
	}

	if instance.Spec.Type != networks.NetworkTypeOther {
		// TODO(alegacy): There is a sysinv bug in how the gateway address
		//  gets registered in the database.  It doesn't have a "name" and
		//  so causes an exception when a related route is added.
		if spec.Gateway != nil && (p.Gateway == nil || !strings.EqualFold(*spec.Gateway, *p.Gateway)) {
			opts.Gateway = spec.Gateway
			result = true
		}
	}

	if spec.Allocation.Order != nil && *spec.Allocation.Order != p.Order {
		opts.Order = spec.Allocation.Order
		result = true
	}

	if len(spec.Allocation.Ranges) > 0 {
		ranges := makeRangeArray(spec.Allocation.Ranges)
		if !compareRangeArrays(ranges, p.Ranges) {
			opts.Ranges = &ranges
			result = true
		}
	}

	return opts, result
}

// ReconcileNew is a method which handles reconciling a new data resource and
// creates the corresponding system resource thru the system API.
func (r *ReconcilePlatformNetwork) ReconcileNewAddressPool(client *gophercloud.ServiceClient, instance *starlingxv1beta1.PlatformNetwork) (*addresspools.AddressPool, error) {
	opts := addresspools.AddressPoolOpts{
		Name:    &instance.Name,
		Network: &instance.Spec.Subnet,
		Prefix:  &instance.Spec.Prefix,
		Order:   instance.Spec.Allocation.Order,
	}

	if instance.Spec.Type != networks.NetworkTypeOther {
		// TODO(alegacy): There is a sysinv bug in how the gateway address
		//  gets registered in the database.  It doesn't have a "name" and
		//  so causes an exception when a related route is added.  So to
		// avoid the issue only set the gateway for all other network types.
		opts.Gateway = instance.Spec.Gateway
	}

	if len(instance.Spec.Allocation.Ranges) > 0 {
		ranges := makeRangeArray(instance.Spec.Allocation.Ranges)
		opts.Ranges = &ranges
	}

	log.Info("creating address pool", "opts", opts)

	pool, err := addresspools.Create(client, opts).Extract()
	if err != nil {
		err = perrors.Wrapf(err, "failed to create pool: %s", common.FormatStruct(opts))
		return nil, err
	}

	r.NormalEvent(instance, common.ResourceCreated,
		"address pool has been created")

	return pool, nil
}

// ReconcileUpdated is a method which handles reconciling an existing data
// resource and updates the corresponding system resource thru the system API to
// match the desired state of the resource.
func (r *ReconcilePlatformNetwork) ReconcileUpdatedAddressPool(client *gophercloud.ServiceClient, instance *starlingxv1beta1.PlatformNetwork, pool *addresspools.AddressPool) error {
	if opts, ok := poolUpdateRequired(instance, pool); ok {
		// Update existing pool
		log.Info("updating address pool", "uuid", pool.ID, "opts", opts)

		result, err := addresspools.Update(client, pool.ID, opts).Extract()
		if err != nil {
			err = perrors.Wrapf(err, "failed to update pool: %+v", opts)
			return err
		}

		*pool = *result

		r.NormalEvent(instance, common.ResourceUpdated,
			"address pool has been updated")
	}

	return nil
}

// ReconcileNew is a method which handles reconciling a new data resource and
// creates the corresponding system resource thru the system API.
func (r *ReconcilePlatformNetwork) ReconciledDeletedAddressPool(client *gophercloud.ServiceClient, instance *starlingxv1beta1.PlatformNetwork, pool *addresspools.AddressPool) error {
	if pool != nil {
		// Unless it was already deleted go ahead and attempt to delete it.
		err := addresspools.Delete(client, pool.ID).ExtractErr()
		if err != nil {
			if code, ok := err.(gophercloud.ErrUnexpectedResponseCode); !ok {
				err = perrors.Wrap(err, "failed to delete address pool")
				return err
			} else if code.Actual == 409 {
				log.Info("address pool is still in use; deleting local resource anyway")
				// NOTE: there is no way to block the kubernetes delete beyond
				//  delaying it a little while we delete an external resource so
				//  since we can't fail this then log it and allow it to
				//  continue for now.
			} else {
				err = perrors.Wrap(err, "unexpected response code on address pool delete")
				return err
			}
		}

		r.NormalEvent(instance, common.ResourceDeleted, "address pool has been deleted")
	}

	return nil
}

// FindExistingAddressPool attempts to re-use the existing resource referenced
// by the ID value stored in the status or to find another resource with a
// matching name.
func (r *ReconcilePlatformNetwork) FindExistingAddressPool(client *gophercloud.ServiceClient, instance *starlingxv1beta1.PlatformNetwork) (pool *addresspools.AddressPool, err error) {
	id := instance.Status.ID
	if id != nil {
		// This network was previously provisioned.
		pool, err = addresspools.Get(client, *id).Extract()
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
		// This network needs to be provisioned if it doesn't already exist.
		results, err := addresspools.ListAddressPools(client)
		if err != nil {
			err = perrors.Wrap(err, "failed to list pools")
			return nil, err
		}

		for _, p := range results {
			if p.Name == instance.Name {
				pool = &p
				break
			}
		}
	}

	return pool, err
}

// ReconcileAddressPool determines whether the stored address pool instance
// needs to be created or updated in the system.  This is done independently of
// the network resource since at the system level these are two independent
// resources.
func (r *ReconcilePlatformNetwork) ReconcileAddressPool(client *gophercloud.ServiceClient, instance *starlingxv1beta1.PlatformNetwork) error {
	pool, err := r.FindExistingAddressPool(client, instance)
	if err != nil {
		return err
	}

	if instance.DeletionTimestamp.IsZero() == false {
		if common.ContainsString(instance.ObjectMeta.Finalizers, FinalizerName) {
			return r.ReconciledDeletedAddressPool(client, instance, pool)
		}

	} else {
		if pool == nil {
			pool, err = r.ReconcileNewAddressPool(client, instance)
		} else {
			err = r.ReconcileUpdatedAddressPool(client, instance, pool)
		}

		if err == nil && pool != nil {
			// Update the resource status to link it to the system object.
			if instance.Status.PoolUUID == nil || *instance.Status.PoolUUID != pool.ID {
				instance.Status.PoolUUID = &pool.ID
			}
		}
	}

	return nil
}

// networkUpdateRequired determines whether a system network resource must be
// updated to align with the stored value.  Only the updated fields are
// include in the request options to minimum churn and to ease debugging.
func networkUpdateRequired(instance *starlingxv1beta1.PlatformNetwork, n *networks.Network) (opts networks.NetworkOpts, result bool) {
	if instance.Name != n.Name {
		opts.Name = &instance.Name
		result = true
	}

	spec := instance.Spec
	if spec.Type != n.Type {
		opts.Type = &spec.Type
		result = true
	}

	dynamic := bool(spec.Allocation.Type != AllocationTypeDynamic)
	if dynamic != n.Dynamic {
		opts.Dynamic = &dynamic
		result = true
	}

	status := instance.Status
	if status.PoolUUID != nil && *status.PoolUUID != n.PoolUUID {
		opts.PoolUUID = status.PoolUUID
		result = true
	}

	return opts, result
}

// networkResourceRequired determines whether the network instance requires both
// an address pool resource and a network resource or whether it requires only
// an address pool resource. Networks that are used purely to instantiate an
// address pool do not require an actual network resource to be created.  In
// the future we will hopefully change the system API to require this or even
// change it so that address pools are not explicitly provisioned, but for now
// we simply manage this detail here to make the UX better.
func networkResourceRequired(instance *starlingxv1beta1.PlatformNetwork) bool {
	switch instance.Spec.Type {
	case networks.NetworkTypeOther:
		return false
	default:
		return true
	}
}

// ReconcileNewNetwork is a method which handles reconciling a new data resource
// and creates the corresponding system resource thru the system API.
func (r *ReconcilePlatformNetwork) ReconcileNewNetwork(client *gophercloud.ServiceClient, instance *starlingxv1beta1.PlatformNetwork) (*networks.Network, error) {
	dynamic := bool(instance.Spec.Allocation.Type == AllocationTypeDynamic)

	opts := networks.NetworkOpts{
		Name:     &instance.Name,
		Type:     &instance.Spec.Type,
		Dynamic:  &dynamic,
		PoolUUID: instance.Status.PoolUUID,
	}

	log.Info("creating platform network", "opts", opts)

	network, err := networks.Create(client, opts).Extract()
	if err != nil {
		err = perrors.Wrapf(err, "failed to create: %s", common.FormatStruct(opts))
		return nil, err
	}

	r.NormalEvent(instance, common.ResourceCreated,
		"platform network has been created")

	return network, nil
}

// ReconcileUpdated is a method which handles reconciling an existing data
// resource and updates the corresponding system resource thru the system API to
// match the desired state of the resource.
func (r *ReconcilePlatformNetwork) ReconcileUpdatedNetwork(client *gophercloud.ServiceClient, instance *starlingxv1beta1.PlatformNetwork, network *networks.Network) error {
	if opts, ok := networkUpdateRequired(instance, network); ok {
		log.Info("updating platform network", "uuid", network.UUID, "opts", opts)

		result, err := networks.Update(client, network.UUID, opts).Extract()
		if err != nil {
			err = perrors.Wrapf(err, "failed to update: %s, %s",
				network.UUID, common.FormatStruct(opts))
			return err
		}

		*network = *result

		r.NormalEvent(instance, common.ResourceUpdated,
			"platform network has been updated")
	}

	return nil
}

// ReconcileNew is a method which handles reconciling a new data resource and
// creates the corresponding system resource thru the system API.
func (r *ReconcilePlatformNetwork) ReconciledDeletedNetwork(client *gophercloud.ServiceClient, instance *starlingxv1beta1.PlatformNetwork, network *networks.Network) error {
	if network != nil {
		// Unless it was already deleted go ahead and attempt to delete it.
		err := networks.Delete(client, network.UUID).ExtractErr()
		if err != nil {
			err = perrors.Wrap(err, "failed to delete network pool")
			return err
		}

		r.NormalEvent(instance, common.ResourceDeleted, "network has been deleted")
	}

	return nil
}

// FindExistingNetwork attempts to re-use the existing resource referenced
// by the ID value stored in the status or to find another resource with a
// matching name.
func (r *ReconcilePlatformNetwork) FindExistingNetwork(client *gophercloud.ServiceClient, instance *starlingxv1beta1.PlatformNetwork) (network *networks.Network, err error) {
	id := instance.Status.ID
	if id != nil {
		// This network was previously provisioned.
		network, err = networks.Get(client, *id).Extract()
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
		// This network needs to be provisioned if it doesn't already exist.
		results, err := networks.ListNetworks(client)
		if err != nil {
			err = perrors.Wrap(err, "failed to list networks")
			return nil, err
		}

		for _, p := range results {
			if p.Name == instance.Name {
				network = &p
				break
			}
		}
	}

	return network, err
}

// ReconcileNetwork determines whether the stored network instance is needs to
// be created or updated in the system.  This is done independently of the
// address pool resource since at the system level these are two independent
// resources.
func (r *ReconcilePlatformNetwork) ReconcileNetwork(client *gophercloud.ServiceClient, instance *starlingxv1beta1.PlatformNetwork) error {
	network, err := r.FindExistingNetwork(client, instance)
	if err != nil {
		return err
	}

	if instance.DeletionTimestamp.IsZero() == false {
		if common.ContainsString(instance.ObjectMeta.Finalizers, FinalizerName) {
			return r.ReconciledDeletedNetwork(client, instance, network)
		}

	} else {
		if network == nil {
			network, err = r.ReconcileNewNetwork(client, instance)
		} else {
			err = r.ReconcileUpdatedNetwork(client, instance, network)
		}

		if err == nil && network != nil {
			// Update the resource status to link it to the system object.
			if instance.Status.ID == nil || *instance.Status.ID != network.UUID {
				instance.Status.ID = &network.UUID
			}
		}
	}

	return nil
}

// statusUpdateRequired is a utility method to determine if the status needs
// to be updated at the API.
func (r *ReconcilePlatformNetwork) statusUpdateRequired(instance *starlingxv1beta1.PlatformNetwork, status *starlingxv1beta1.PlatformNetworkStatus) bool {
	return instance.Status.DeepEqual(status) == false
}

// ReconcileResource interacts with the system API in order to reconcile the
// state of a data network with the state stored in the k8s database.
func (r *ReconcilePlatformNetwork) ReconcileResource(client *gophercloud.ServiceClient, instance *starlingxv1beta1.PlatformNetwork) (err error) {
	oldStatus := instance.Status.DeepCopy()

	if instance.DeletionTimestamp.IsZero() {
		err = r.ReconcileAddressPool(client, instance)
		if err == nil {
			if networkResourceRequired(instance) {
				err = r.ReconcileNetwork(client, instance)
			}
		}

		inSync := err == nil

		if inSync != instance.Status.InSync {
			r.NormalEvent(instance, common.ResourceUpdated, "synchronization has changed to: %t", inSync)
		}

		instance.Status.InSync = inSync

		if r.statusUpdateRequired(instance, oldStatus) {
			err2 := r.Status().Update(context.TODO(), instance)
			if err2 != nil {
				log.Error(err2, "failed to update platform network status")
				return err2
			}
		}

	} else {
		// Reverse the order of operations for deletes since there is a built-in
		// dependency between the two.
		if networkResourceRequired(instance) {
			err = r.ReconcileNetwork(client, instance)
		} else {
			err = nil
		}

		if err == nil {
			err = r.ReconcileAddressPool(client, instance)
		}

		if err == nil {
			// Remove the finalizer so the kubernetes delete operation can
			// continue.
			instance.ObjectMeta.Finalizers = common.RemoveString(instance.ObjectMeta.Finalizers, FinalizerName)
			if err := r.Update(context.Background(), instance); err != nil {
				return err
			}
		}

	}

	return err
}

// Reconcile reads that state of the cluster for a PlatformNetwork object and makes changes based on the state read
// and what is in the PlatformNetwork.Spec
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=platformnetworks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=platformnetworks/status,verbs=get;update;patch
func (r *ReconcilePlatformNetwork) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// To reduce the repitition of adding the resource name to every log we
	// replace the logger with one that includes the resource name and then
	// restore it at the end of the reconcile function.

	savedLog := log
	log = log.WithName(request.NamespacedName.String())
	defer func() { log = savedLog }()

	// Fetch the DataNetwork instance
	instance := &starlingxv1beta1.PlatformNetwork{}
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
		if !common.ContainsString(instance.ObjectMeta.Finalizers, FinalizerName) {
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

	if r.IsReconcilerEnabled(titaniumManager.PlatformNetwork) == false {
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

	if r.GetSystemReady(request.Namespace) == false {
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
