/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package datanetwork

import (
	"context"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/datanetworks"
	perrors "github.com/pkg/errors"
	starlingxv1beta1 "github.com/wind-river/cloud-platform-deployment-manager/pkg/apis/starlingx/v1beta1"
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

var log = logf.Log.WithName("controller").WithName("datanetwork")

const ControllerName = "datanetwork-controller"

const FinalizerName = "datanetwork.finalizers.windriver.com"

// Add creates a new DataNetwork Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	tMgr := cloudManager.GetInstance(mgr)
	return &ReconcileDataNetwork{
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

	// Watch for changes to DataNetwork
	err = c.Watch(&source.Kind{Type: &starlingxv1beta1.DataNetwork{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileDataNetwork{}

// ReconcileResource reconciles a DataNetwork object
type ReconcileDataNetwork struct {
	client.Client
	scheme *runtime.Scheme
	cloudManager.CloudManager
	common.ReconcilerErrorHandler
	common.ReconcilerEventLogger
}

// networkUpdateRequired is a utility function which determines whether an
// update is needed to a data network system resource in order to reconcile
// with the latest stored configuration.
func networkUpdateRequired(instance *starlingxv1beta1.DataNetwork, n *datanetworks.DataNetwork) (opts datanetworks.DataNetworkOpts, result bool) {
	if instance.Name != n.Name {
		opts.Name = &instance.Name
		result = true
	}

	spec := instance.Spec
	if spec.Type != n.Type {
		opts.Type = &spec.Type
		result = true
	}

	if spec.MTU != nil && *spec.MTU != n.MTU {
		opts.MTU = spec.MTU
		result = true
	}

	if spec.Description != nil && *spec.Description != n.Description {
		opts.Description = spec.Description
		result = true
	}

	if spec.Type == datanetworks.TypeVxLAN && spec.VxLAN != nil {
		vxlan := spec.VxLAN
		if vxlan.EndpointMode != nil && n.Mode != nil {
			if *vxlan.EndpointMode != *n.Mode {
				opts.Mode = vxlan.EndpointMode
				result = true
			}
		}

		if vxlan.UDPPortNumber != nil && n.UDPPortNumber != nil {
			if *vxlan.UDPPortNumber != *n.UDPPortNumber {
				opts.PortNumber = vxlan.UDPPortNumber
				result = true
			}
		}

		if vxlan.TTL != nil && n.TTL != nil {
			if *vxlan.TTL != *n.TTL {
				opts.TTL = vxlan.TTL
				result = true
			}
		}

		if vxlan.MulticastGroup != nil && n.MulticastGroup != nil {
			if *vxlan.MulticastGroup != *n.MulticastGroup {
				opts.MulticastGroup = vxlan.MulticastGroup
				result = true
			}
		}
	}

	return opts, result
}

// ReconcileNew is a method which handles reconciling a new data resource and
// creates the corresponding system resource thru the system API.
func (r *ReconcileDataNetwork) ReconcileNew(client *gophercloud.ServiceClient, instance *starlingxv1beta1.DataNetwork) (*datanetworks.DataNetwork, error) {
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

	// Create a new network
	opts := datanetworks.DataNetworkOpts{
		Name:        &instance.Name,
		Type:        &instance.Spec.Type,
		Description: instance.Spec.Description,
		MTU:         instance.Spec.MTU,
	}

	if instance.Spec.Type == datanetworks.TypeVxLAN && instance.Spec.VxLAN != nil {
		// VxLAN requires special attributes.
		opts.Mode = instance.Spec.VxLAN.EndpointMode
		opts.MulticastGroup = instance.Spec.VxLAN.MulticastGroup
		opts.TTL = instance.Spec.VxLAN.TTL
		opts.PortNumber = instance.Spec.VxLAN.UDPPortNumber
	}

	log.Info("creating data network", "opts", opts)

	network, err := datanetworks.Create(client, opts).Extract()
	if err != nil {
		err = perrors.Wrapf(err, "failed to create: %s", common.FormatStruct(opts))
		return nil, err
	}

	r.NormalEvent(instance, common.ResourceCreated,
		"data network has been created")

	return network, nil
}

// ReconcileUpdated is a method which handles reconciling an existing data
// resource and updates the corresponding system resource thru the system API to
// match the desired state of the resource.
func (r *ReconcileDataNetwork) ReconcileUpdated(client *gophercloud.ServiceClient, instance *starlingxv1beta1.DataNetwork, network *datanetworks.DataNetwork) error {
	// Update existing network
	if opts, ok := networkUpdateRequired(instance, network); ok {
		if instance.Status.Reconciled && r.StopAfterInSync() {
			// Do not process any further changes once we have reached a
			// synchronized state unless there is an annotation on the resource.
			if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
				msg := common.NoChangesAfterReconciled
				r.NormalEvent(instance, common.ResourceUpdated, msg)
				return common.NewChangeAfterInSync(msg)
			} else {
				log.Info(common.ChangedAllowedAfterReconciled)
			}
		}

		log.Info("updating data network", "uuid", network.ID, "opts", opts)

		result, err := datanetworks.Update(client, network.ID, opts).Extract()
		if err != nil {
			err = perrors.Wrapf(err, "failed to update: %s", common.FormatStruct(opts))
			return err
		}

		*network = *result

		r.NormalEvent(instance, common.ResourceUpdated,
			"data network has been updated")
	}

	return nil
}

// ReconcileNew is a method which handles reconciling a new data resource and
// creates the corresponding system resource thru the system API.
func (r *ReconcileDataNetwork) ReconciledDeleted(client *gophercloud.ServiceClient, instance *starlingxv1beta1.DataNetwork, network *datanetworks.DataNetwork) error {
	if utils.ContainsString(instance.ObjectMeta.Finalizers, FinalizerName) {
		if network != nil {
			// Unless it was already deleted go ahead and attempt to delete it.
			err := datanetworks.Delete(client, network.ID).ExtractErr()
			if err != nil {
				if code, ok := err.(gophercloud.ErrDefault400); !ok {
					err = perrors.Wrap(err, "failed to delete data network")
					return err
				} else if code.Actual == 400 {
					log.Info("data network is still in use; deleting local resource anyway")
					// NOTE: there is no way to block the kubernetes delete beyond
					//  delaying it a little while we delete an external resource so
					//  since we can't fail this then log it and allow it to
					//  continue for now.
				} else {
					err = perrors.Wrap(err, "unexpected response code on data network delete")
					return err
				}
			}

			r.NormalEvent(instance, common.ResourceDeleted, "data network has been deleted")
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
func (r *ReconcileDataNetwork) statusUpdateRequired(instance *starlingxv1beta1.DataNetwork, network *datanetworks.DataNetwork, inSync bool) (result bool) {
	status := &instance.Status

	if network != nil {
		if status.ID == nil || *status.ID != network.ID {
			status.ID = &network.ID
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

// FindExistingResource attempts to re-use the existing resource referenced by
// the ID value stored in the status or to find another resource with a matching
// name.
func (r *ReconcileDataNetwork) FindExistingResource(client *gophercloud.ServiceClient, instance *starlingxv1beta1.DataNetwork) (network *datanetworks.DataNetwork, err error) {
	id := instance.Status.ID
	if id != nil {
		// This network was previously provisioned.
		network, err = datanetworks.Get(client, *id).Extract()
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
		results, err := datanetworks.ListDataNetworks(client)
		if err != nil {
			err = perrors.Wrap(err, "failed to list")
			return nil, err
		}

		for _, net := range results {
			if net.Name == instance.Name {
				if net.Type == instance.Spec.Type {
					log.Info("found existing data network", "uuid", net.ID)
					network = &net
					break
				}
			}
		}
	}

	return network, err
}

// ReconcileResource interacts with the system API in order to reconcile the
// state of a data network with the state stored in the k8s database.
func (r *ReconcileDataNetwork) ReconcileResource(client *gophercloud.ServiceClient, instance *starlingxv1beta1.DataNetwork) error {
	network, err := r.FindExistingResource(client, instance)
	if err != nil {
		return err
	}

	if !instance.DeletionTimestamp.IsZero() {
		err = r.ReconciledDeleted(client, instance, network)

	} else {
		if network == nil {
			network, err = r.ReconcileNew(client, instance)
		} else {
			err = r.ReconcileUpdated(client, instance, network)
		}

		inSync := err == nil

		if instance.Status.InSync != inSync {
			r.NormalEvent(instance, common.ResourceUpdated, "synchronization has changed to: %t", inSync)
		}

		if r.statusUpdateRequired(instance, network, inSync) {
			// Update the resource status to link it to the system object.
			log.Info("updating data network", "status", instance.Status)

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
func (r *ReconcileDataNetwork) StopAfterInSync() bool {
	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return config.GetReconcilerOptionBool(config.DataNetwork, config.StopAfterInSync, true)
}

// Reconcile reads that state of the cluster for a DataNetwork object and makes changes based on the state read
// and what is in the DataNetwork.Spec
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=datanetworks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=datanetworks/status,verbs=get;update;patch
func (r *ReconcileDataNetwork) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	savedLog := log
	log = log.WithName(request.NamespacedName.String())
	defer func() { log = savedLog }()

	// Fetch the DataNetwork instance
	instance := &starlingxv1beta1.DataNetwork{}
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

	if !config.IsReconcilerEnabled(config.DataNetwork) {
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
