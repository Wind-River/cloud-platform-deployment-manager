/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/datanetworks"
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

var logDataNetwork = log.Log.WithName("controller").WithName("datanetwork")

const DataNetworkControllerName = "datanetwork-controller"

const DataNetworkFinalizerName = "datanetwork.finalizers.windriver.com"

var _ reconcile.Reconciler = &DataNetworkReconciler{}

// DataNetworkReconciler reconciles a DataNetwork object
type DataNetworkReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	cloudManager.CloudManager
	common.ReconcilerErrorHandler
	common.ReconcilerEventLogger
}

// dataNetworkUpdateRequired is a utility function which determines whether an
// update is needed to a data network system resource in order to reconcile
// with the latest stored configuration.
func dataNetworkUpdateRequired(instance *starlingxv1.DataNetwork, n *datanetworks.DataNetwork) (opts datanetworks.DataNetworkOpts, result bool) {
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
func (r *DataNetworkReconciler) ReconcileNew(client *gophercloud.ServiceClient, instance *starlingxv1.DataNetwork) (*datanetworks.DataNetwork, error) {
	if instance.Status.Reconciled && r.StopAfterInSync() {
		// Do not process any further changes once we have reached a
		// synchronized state unless there is an annotation on the resource.
		if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
			msg := common.NoProvisioningAfterReconciled
			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, msg)
			return nil, common.NewChangeAfterInSync(msg)
		} else {
			logDataNetwork.Info(common.ProvisioningAllowedAfterReconciled)
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

	logDataNetwork.Info("creating data network", "opts", opts)

	network, err := datanetworks.Create(client, opts).Extract()
	if err != nil {
		err = perrors.Wrapf(err, "failed to create: %s", common.FormatStruct(opts))
		return nil, err
	}

	r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceCreated,
		"data network has been created")

	return network, nil
}

// ReconcileUpdated is a method which handles reconciling an existing data
// resource and updates the corresponding system resource thru the system API to
// match the desired state of the resource.
func (r *DataNetworkReconciler) ReconcileUpdated(client *gophercloud.ServiceClient, instance *starlingxv1.DataNetwork, network *datanetworks.DataNetwork) error {
	// Update existing network
	if opts, ok := dataNetworkUpdateRequired(instance, network); ok {
		if instance.Status.Reconciled && r.StopAfterInSync() {
			// Do not process any further changes once we have reached a
			// synchronized state unless there is an annotation on the resource.
			if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
				msg := common.NoChangesAfterReconciled
				r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, msg)
				return common.NewChangeAfterInSync(msg)
			} else {
				logDataNetwork.Info(common.ChangedAllowedAfterReconciled)
			}
		}

		logDataNetwork.Info("updating data network", "uuid", network.ID, "opts", opts)

		result, err := datanetworks.Update(client, network.ID, opts).Extract()
		if err != nil {
			err = perrors.Wrapf(err, "failed to update: %s", common.FormatStruct(opts))
			return err
		}

		*network = *result

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
			"data network has been updated")
	}

	return nil
}

// ReconcileNew is a method which handles reconciling a new data resource and
// creates the corresponding system resource thru the system API.
func (r *DataNetworkReconciler) ReconciledDeleted(client *gophercloud.ServiceClient, instance *starlingxv1.DataNetwork, network *datanetworks.DataNetwork) error {
	if utils.ContainsString(instance.ObjectMeta.Finalizers, DataNetworkFinalizerName) {
		if network != nil {
			// Unless it was already deleted go ahead and attempt to delete it.
			err := datanetworks.Delete(client, network.ID).ExtractErr()
			if err != nil {
				if code, ok := err.(gophercloud.ErrDefault400); !ok {
					err = perrors.Wrap(err, "failed to delete data network")
					return err
				} else if code.Actual == 400 {
					logDataNetwork.Info("data network is still in use; deleting local resource anyway")
					// NOTE: there is no way to block the kubernetes delete beyond
					//  delaying it a little while we delete an external resource so
					//  since we can't fail this then log it and allow it to
					//  continue for now.
				} else {
					err = perrors.Wrap(err, "unexpected response code on data network delete")
					return err
				}
			}

			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceDeleted, "data network has been deleted")
		}

		// Remove the finalizer so the kubernetes delete operation can continue.
		instance.ObjectMeta.Finalizers = utils.RemoveString(instance.ObjectMeta.Finalizers, DataNetworkFinalizerName)
		if err := r.Client.Update(context.Background(), instance); err != nil {
			return err
		}

	}

	return nil
}

// statusUpdateRequired is a utility function which determines whether an update
// is required to the host status attribute.  Updating this unnecessarily
// will result in an infinite reconciliation loop.
func (r *DataNetworkReconciler) statusUpdateRequired(instance *starlingxv1.DataNetwork, network *datanetworks.DataNetwork, inSync bool) (result bool) {
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
func (r *DataNetworkReconciler) FindExistingResource(client *gophercloud.ServiceClient, instance *starlingxv1.DataNetwork) (network *datanetworks.DataNetwork, err error) {
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
			logDataNetwork.Info("resource no longer exists", "id", *id)
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
					logDataNetwork.Info("found existing data network", "uuid", net.ID)
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
func (r *DataNetworkReconciler) ReconcileResource(client *gophercloud.ServiceClient, instance *starlingxv1.DataNetwork) error {
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
			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "synchronization has changed to: %t", inSync)
		}

		if r.statusUpdateRequired(instance, network, inSync) {
			// Update the resource status to link it to the system object.
			logDataNetwork.Info("updating data network", "status", instance.Status)

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
func (r *DataNetworkReconciler) StopAfterInSync() bool {
	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return utils.GetReconcilerOptionBool(utils.DataNetwork, utils.StopAfterInSync, true)
}

// Reconcile reads that state of the cluster for a DataNetwork object and makes changes based on the state read
//+kubebuilder:rbac:groups=starlingx.windriver.com,resources=datanetworks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=starlingx.windriver.com,resources=datanetworks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=starlingx.windriver.com,resources=datanetworks/finalizers,verbs=update
func (r *DataNetworkReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	savedLog := logDataNetwork
	logDataNetwork = logDataNetwork.WithName(request.NamespacedName.String())
	defer func() { logDataNetwork = savedLog }()

	// Fetch the DataNetwork instance
	instance := &starlingxv1.DataNetwork{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically
			// garbage collected. For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}

		logDataNetwork.Error(err, "unable to read object: %v", request)
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.DeletionTimestamp.IsZero() {
		// Ensure that the object has a finalizer setup as a pre-delete hook so
		// that we can delete any system resources that we previously added.
		if !utils.ContainsString(instance.ObjectMeta.Finalizers, DataNetworkFinalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, DataNetworkFinalizerName)
			if err := r.Client.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}

			// Might as well return immediately as the update is going to cause
			// another reconcile event for this resource and we don't want to
			// access the system API more than necessary.
			return reconcile.Result{}, nil
		}
	}

	if !utils.IsReconcilerEnabled(utils.DataNetwork) {
		return reconcile.Result{}, nil
	}

	platformClient := r.GetPlatformClient(request.Namespace)
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
func (r *DataNetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	tMgr := cloudManager.GetInstance(mgr)
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.CloudManager = tMgr
	r.ReconcilerErrorHandler = &common.ErrorHandler{
		CloudManager: tMgr,
		Logger:       logDataNetwork}
	r.ReconcilerEventLogger = &common.EventLogger{
		EventRecorder: mgr.GetEventRecorderFor(DataNetworkControllerName),
		Logger:        logDataNetwork}
	return ctrl.NewControllerManagedBy(mgr).
		For(&starlingxv1.DataNetwork{}).
		Complete(r)
}
