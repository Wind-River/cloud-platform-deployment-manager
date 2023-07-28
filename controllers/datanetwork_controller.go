/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2023 Wind River Systems, Inc. */

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
func dataNetworkUpdateRequired(instance *starlingxv1.DataNetwork, n *datanetworks.DataNetwork, r *DataNetworkReconciler) (opts datanetworks.DataNetworkOpts, result bool) {
	var delta strings.Builder
	if instance.Name != n.Name {
		opts.Name = &instance.Name
		delta.WriteString(fmt.Sprintf("\t+Name: %s\n", *opts.Name))
		result = true
	}

	spec := instance.Spec
	if spec.Type != n.Type {
		opts.Type = &spec.Type
		delta.WriteString(fmt.Sprintf("\t+Type: %s\n", *opts.Type))
		result = true
	}

	if spec.MTU != nil && *spec.MTU != n.MTU {
		opts.MTU = spec.MTU
		delta.WriteString(fmt.Sprintf("\t+MTU: %d\n", *opts.MTU))
		result = true
	}

	if spec.Description != nil && *spec.Description != n.Description {
		opts.Description = spec.Description
		delta.WriteString(fmt.Sprintf("\t+Description: %s\n", *opts.Description))
		result = true
	}

	if spec.Type == datanetworks.TypeVxLAN && spec.VxLAN != nil {
		vxlan := spec.VxLAN
		if vxlan.EndpointMode != nil && n.Mode != nil {
			if *vxlan.EndpointMode != *n.Mode {
				opts.Mode = vxlan.EndpointMode
				delta.WriteString(fmt.Sprintf("\t+Mode: %s\n", *opts.Mode))
				result = true
			}
		}

		if vxlan.UDPPortNumber != nil && n.UDPPortNumber != nil {
			if *vxlan.UDPPortNumber != *n.UDPPortNumber {
				opts.PortNumber = vxlan.UDPPortNumber
				delta.WriteString(fmt.Sprintf("\t+PortNumber: %d\n", *opts.PortNumber))
				result = true
			}
		}

		if vxlan.TTL != nil && n.TTL != nil {
			if *vxlan.TTL != *n.TTL {
				opts.TTL = vxlan.TTL
				delta.WriteString(fmt.Sprintf("\t+TTL: %d\n", *opts.TTL))
				result = true
			}
		}

		if vxlan.MulticastGroup != nil && n.MulticastGroup != nil {
			if *vxlan.MulticastGroup != *n.MulticastGroup {
				opts.MulticastGroup = vxlan.MulticastGroup
				delta.WriteString(fmt.Sprintf("\t+MulticastGroup: %s\n", *opts.MulticastGroup))
				result = true
			}
		}
	}
	deltaString := delta.String()
	if deltaString != "" {
		deltaString = "\n" + strings.TrimSuffix(deltaString, "\n")
		logDataNetwork.Info(fmt.Sprintf("delta configuration:%s\n", deltaString))
	}
	instance.Status.Delta = deltaString
	err := r.Client.Status().Update(context.TODO(), instance)
	if err != nil {
		logDataNetwork.Info(fmt.Sprintf("failed to update status:  %s\n", err))
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
	if opts, ok := dataNetworkUpdateRequired(instance, network, r); ok {
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
		status.ConfigurationUpdated = false
		status.StrategyRequired = cloudManager.StrategyNotRequired
		if instance.Status.DeploymentScope == cloudManager.ScopePrincipal {
			r.CloudManager.SetResourceInfo(cloudManager.ResourceDatanetwork, "", instance.Name, status.Reconciled, status.StrategyRequired)
		}
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

// Obtain deploymentScope value from configuration
// Taking this value from annotation in instacne
// (It seems Client.Get does not update Status value from configuration)
// "bootstrap" if "bootstrap" in configuration or deploymentScope not specified
// "principal" if "principal" in configuration
func (r *DataNetworkReconciler) GetScopeConfig(instance *starlingxv1.DataNetwork) (scope string, err error) {
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
func (r *DataNetworkReconciler) UpdateConfigStatus(instance *starlingxv1.DataNetwork) (err error) {
	deploymentScope, err := r.GetScopeConfig(instance)
	if err != nil {
		return err
	}
	logDataNetwork.V(2).Info("deploymentScope in configuration", "deploymentScope", deploymentScope)

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
				r.CloudManager.SetResourceInfo(cloudManager.ResourceDatanetwork, "", instance.Name, instance.Status.Reconciled, cloudManager.StrategyNotRequired)
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

// Reconcile reads that state of the cluster for a DataNetwork object and makes changes based on the state read
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=datanetworks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=datanetworks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=datanetworks/finalizers,verbs=update
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

	// Update scope from configuration
	logDataNetwork.V(2).Info("before UpdateConfigStatus", "instance", instance)
	err = r.UpdateConfigStatus(instance)
	if err != nil {
		logDataNetwork.Error(err, "unable to update scope")
		return reconcile.Result{}, err
	}
	logDataNetwork.V(2).Info("after UpdateConfigStatus", "instance", instance)

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
