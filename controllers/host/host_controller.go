/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2024 Wind River Systems, Inc. */

package host

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/labels"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinstances"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/storagebackends"
	perrors "github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers/common"
	cloudManager "github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var logHost = log.Log.WithName("controller").WithName("host")

const HostControllerName = "host-controller"

const HostFinalizerName = "host.finalizers.windriver.com"

// RequiredState defines an alias that represents in which host state(s) is a
// resource allowed to be provisioned.
type RequiredState string

// Defines the value values for the RequiredState type alias.
const (
	RequiredStateNone     RequiredState = "none"
	RequiredStateAny      RequiredState = "any"
	RequiredStateEnabled  RequiredState = "enabled"
	RequiredStateDisabled RequiredState = "disabled"
)

// DefaultHostProfile contains mandatory default values for a host profile.
// This is intentionally sparsely populated because the system API defaults are
// preferred for any attributes not specified by the user.  Only attributes that
// are absolutely required for the proper functioning of this controller should
// be specified here.
var AdminLocked = hosts.AdminLocked
var DynamicProvisioningMode = starlingxv1.ProvioningModeDynamic
var DefaultHostProfile = starlingxv1.HostProfileSpec{
	ProfileBaseAttributes: starlingxv1.ProfileBaseAttributes{
		AdministrativeState: &AdminLocked,
		ProvisioningMode:    &DynamicProvisioningMode,
	},
}

var CephPrimaryGroup []string

// Only the listed file systems are allow to create and delete
var FileSystemCreationAllowed = []string{"instances", "image-conversion", "ceph"}
var FileSystemDeletionAllowed = []string{"instances", "image-conversion"}

var _ reconcile.Reconciler = &HostReconciler{}

// HostReconciler reconciles a Host object
type HostReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	cloudManager.CloudManager
	common.ReconcilerErrorHandler
	common.ReconcilerEventLogger
	hosts []hosts.Host
}

// hostMatchesCriteria evaluates whether a host matches the criteria specified
// by the operator.  All match attributes must match for a host to match a
// profile.
func hostMatchesCriteria(h hosts.Host, criteria *starlingxv1.MatchInfo) bool {
	result := true
	count := 0

	if criteria == nil {
		return false
	}

	if criteria.BootMAC != nil {
		count++
		result = result && strings.EqualFold(h.BootMAC, *criteria.BootMAC)
	}

	if criteria.BoardManagement != nil {
		bm := criteria.BoardManagement
		if bm.Address != nil {
			count++
			result = result && strings.EqualFold(*bm.Address, *h.BMAddress)
		}

		if bm.Type != nil {
			count++
			result = result && strings.EqualFold(*bm.Type, *h.BMType)
		}
	}

	if criteria.DMI != nil {
		dmi := criteria.DMI
		if dmi.SerialNumber != nil {
			count++
			if h.SerialNumber != nil {
				result = result && strings.EqualFold(*dmi.SerialNumber, *h.SerialNumber)
			}
		}

		if dmi.AssetTag != nil {
			count++
			if h.AssetTag != nil {
				result = result && strings.EqualFold(*dmi.AssetTag, *h.AssetTag)
			}
		}
	}

	return result && count > 0
}

// Defines the keys used to access BM credential information stored in a secret.
const (
	usernameKey = "username"
	passwordKey = "password"
)

func (r *HostReconciler) IsActiveHost(client *gophercloud.ServiceClient, instance *starlingxv1.Host, reqNs string) (bool, error) {
	host_instance, _, err := r.CloudManager.GetHostByPersonality(reqNs, client, cloudManager.ActiveController)
	if err != nil {
		msg := "failed to get active host"
		return false, common.NewUserDataError(msg)
	}

	if host_instance.Name == instance.Name {
		return true, nil
	}

	return false, nil
}

// getBMPasswordCredentials is a utility to retrieve the host's board management
// credentials from the information stored in the specified secret.
func (r *HostReconciler) getBMPasswordCredentials(namespace string, name string) (username, password string, err error) {
	secret := &v1.Secret{}
	secretName := types.NamespacedName{Namespace: namespace, Name: name}

	// Lookup the secret via the system client.
	err = r.Client.Get(context.TODO(), secretName, secret)
	if err != nil {
		if !errors.IsNotFound(err) {
			err = perrors.Wrap(err, "failed to get host BM secret")
		}
		return "", "", err
	}

	// Make sure that required keys are present.
	for _, key := range []string{usernameKey, passwordKey} {
		if _, ok := secret.Data[usernameKey]; !ok {
			msg := fmt.Sprintf("missing %q key within BM credential secret", key)
			return "", "", common.NewUserDataError(msg)
		}
	}

	return string(secret.Data[usernameKey]), string(secret.Data[passwordKey]), nil
}

// buildInitialHostOpts is a utility to assemble the options required to
// provision a host that needs to be statically provisioned.  Further
// provisioning of other host attributes will be handled at a later stage.
func (r *HostReconciler) buildInitialHostOpts(instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec) (hosts.HostOpts, error) {
	dummy := hosts.Host{}
	result, _, err := r.UpdateRequired(instance, profile, &dummy)
	return result, err
}

func provisioningAllowed(objects []hosts.Host) bool {
	for _, host := range objects {
		if host.Hostname == hosts.Controller0 || host.Hostname == "controller-1" {
			if host.IsUnlockedEnabled() {
				return true
			}
		}
	}

	return false
}

// ProvisioningAllowed determines whether the system will allow creating or
// configuring new hosts.  The primary controller must be enabled for these
// actions to be allowed.
func (r *HostReconciler) ProvisioningAllowed() bool {
	return provisioningAllowed(r.hosts)
}

func MonitorsEnabled(objects []hosts.Host, required int) bool {
	count := 0
	for _, host := range objects {
		function := host.Capabilities.StorFunction
		if function != nil && strings.EqualFold(*function, hosts.StorFunctionMonitor) {
			if host.IsUnlockedEnabled() {
				count += 1
			}
		}
	}
	return count >= required
}

// MonitorsEnabled determines whether the required number of monitors are
// enabled or not. Provisioning certain storage resources requires that a
// certain number of monitors be enabled.
func (r *HostReconciler) MonitorsEnabled(required int) bool {
	return MonitorsEnabled(r.hosts, required)
}

func AllControllerNodesEnabled(objects []hosts.Host, required int) bool {
	count := 0

	for _, host := range objects {
		if host.Personality == hosts.PersonalityController {
			if host.IsUnlockedEnabled() {
				count += 1
			}
		}
	}

	return count >= required
}

// AllControllerNodesEnabled determines whether the system is ready for additional
// nodes to be unlocked.  To avoid issues with provisioning storage resources
// we need to wait for both controllers to be unlocked/enabled.
func (r *HostReconciler) AllControllerNodesEnabled(required int) bool {
	return AllControllerNodesEnabled(r.hosts, required)
}

// UpdateRequired determines if any of the configured attributes mismatch with
// those in the running system.  If there are mismatches then true is returned
// in the result and opts is configured with only those values that
// need to change.
func (r *HostReconciler) UpdateRequired(instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, h *hosts.Host) (opts hosts.HostOpts, result bool, err error) {

	if instance.Name != h.Hostname {
		result = true
		opts.Hostname = &instance.Name
	}

	if profile.Personality != nil && *profile.Personality != h.Personality {
		result = true
		opts.Personality = profile.Personality
	}

	if profile.SubFunctions != nil && profile.Kernel == nil {
		profileSubFunctions := make([]string, 0)
		for _, single := range profile.SubFunctions {
			profileSubFunctions = append(profileSubFunctions, string(single))
		}
		subfunctions := strings.Split(h.SubFunctions, ",")
		if utils.ListChanged(profileSubFunctions, subfunctions) {
			result = true
			subfunctions := strings.Join(profileSubFunctions, ",")
			opts.SubFunctions = &subfunctions
		}
	}

	if profile.Console != nil && *profile.Console != h.Console {
		result = true
		opts.Console = profile.Console
	}

	if profile.InstallOutput != nil && *profile.InstallOutput != h.InstallOutput {
		result = true
		opts.InstallOutput = profile.InstallOutput
	}

	if profile.MaxCPUMhzConfigured != nil && *profile.MaxCPUMhzConfigured != h.MaxCPUMhzConfigured {
		result = true
		opts.MaxCPUMhzConfigured = profile.MaxCPUMhzConfigured
	}

	if profile.AppArmor != nil && *profile.AppArmor != h.AppArmor {
		result = true
		opts.AppArmor = profile.AppArmor
	}

	if profile.HwSettle != nil && *profile.HwSettle != h.HwSettle {
		result = true
		opts.HwSettle = profile.HwSettle
	}

	if profile.RootDevice != nil && *profile.RootDevice != h.RootDevice {
		result = true
		opts.RootDevice = profile.RootDevice
	}

	if profile.BootDevice != nil && *profile.BootDevice != h.BootDevice {
		result = true
		opts.BootDevice = profile.BootDevice
	}

	if profile.BootMAC != nil && *profile.BootMAC != h.BootMAC {
		// Special case for initial provisioning only.  Update not supported.
		result = true
		opts.BootMAC = profile.BootMAC
	}

	if profile.Location != nil {
		if h.Location.Name == nil || *h.Location.Name != *profile.Location {
			result = true
			location := hosts.LocationOpts{Name: *profile.Location}
			opts.Location = &location
		}
	}

	if profile.BoardManagement != nil {
		bm := profile.BoardManagement
		if bm.Address != nil && (h.BMAddress == nil || *bm.Address != *h.BMAddress) {
			result = true
			opts.BMAddress = bm.Address
		}

		if h.BMType == nil || *bm.Type != *h.BMType {
			result = true
			opts.BMType = bm.Type
		}

		if bm.Credentials != nil && bm.Credentials.Password != nil {
			// Password based authentication therefore retrieve the information
			// from the provided secret.
			info := bm.Credentials.Password
			username, password, err := r.getBMPasswordCredentials(instance.Namespace, info.Secret)
			if err != nil {
				if errors.IsNotFound(err) {
					msg := fmt.Sprintf("waiting for BM credentials secret: %q", info.Secret)
					name := types.NamespacedName{Namespace: instance.Namespace, Name: info.Secret}
					m := NewKubernetesSecretMonitor(instance, name)
					return hosts.HostOpts{}, result, r.CloudManager.StartMonitor(m, msg)
				}

				return hosts.HostOpts{}, result, err
			}

			if h.BMUsername == nil || *h.BMUsername != username {
				result = true
				opts.BMUsername = &username
				// TODO(alegacy): There is no good way of knowing if only the
				//  password has changed.
				opts.BMPassword = &password
			}
		}

	} else {
		if h.BMType != nil {
			result = true
			none := hosts.BMTypeDisabled
			opts.BMType = &none
		}
	}

	if profile.ClockSynchronization != nil {
		if h.ClockSynchronization == nil || *profile.ClockSynchronization != *h.ClockSynchronization {
			result = true
			opts.ClockSynchronization = profile.ClockSynchronization
		}
	}

	return opts, result, nil
}

// HTTPSRequired determines whether an HTTPS connection is required for the
// purpose of configuring host BMC attributes.
func (r *HostReconciler) HTTPSRequired() bool {
	value := utils.GetReconcilerOption(utils.BMC, utils.HTTPSRequired)
	if value != nil {
		if required, ok := value.(bool); ok {
			return required
		} else {
			logHost.Info("unexpected option type",
				"option", utils.HTTPSRequired, "type", reflect.TypeOf(value))
		}
	}

	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return true
}

// ReconcileAttributes is responsible for reconciling the basic attributes for a
// host resource.
func (r *HostReconciler) ReconcileAttributes(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *hosts.Host) error {
	if opts, ok, err := r.UpdateRequired(instance, profile, host); ok && err == nil {

		if opts.BMPassword != nil && strings.HasPrefix(client.Endpoint, cloudManager.HTTPPrefix) {
			if r.HTTPSRequired() {
				// Do not send password information in the clear.
				msg := "it is unsafe to configure BM credentials thru a non HTTPS URL"
				return common.NewSystemDependency(msg)
			} else {
				logHost.Info("allowing BMC configuration over HTTP connection")
			}
		}

		logHost.Info("updating host attributes", "opts", opts)

		result, err := hosts.Update(client, host.ID, opts).Extract()
		if err != nil || result == nil {
			err = perrors.Wrapf(err, "failed to update host attributes: %s, %s",
				host.ID, common.FormatStruct(opts))
			return err
		}

		*host = *result

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
			"attributes have been updated")

	} else if err != nil {
		return err
	}

	return nil
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

// ReconcilePTPInstances is responsible for reconciling the PTP instances
// associated with each host.
func (r *HostReconciler) ReconcilePTPInstances(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	// Remove any stale PTP instances
	for _, existing := range host.PTPInstances {
		found := false
		for _, configured := range profile.PtpInstances {
			if string(configured) == existing.Name {
				found = true
				break
			}
		}

		if !found {
			logHost.Info("removing PTP instance", "PTP instance", existing)

			opt := ptpinstances.PTPInstToHostOpts{
				PTPInstanceID: &existing.ID,
			}
			_, err := ptpinstances.RemovePTPInstanceFromHost(client, host.ID, opt).Extract()
			if err != nil {
				err = perrors.Wrapf(err, "failed to remove PTP instance from host: %s", host.ID)
				return err
			}
			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
				"ptp instance %s removed from host", existing.Name)
			updated = true
		}
	}

	for _, configured := range profile.PtpInstances {
		found := false
		for _, result := range host.PTPInstances {
			if string(configured) == result.Name {
				found = true
				break
			}
		}

		if !found {
			result, err := findPTPInstanceByName(client, string(configured))
			if err != nil {
				err = perrors.Wrapf(err, "failed to find PTP instance for host: %s", host.ID)
				return err
			} else if result == nil {
				return common.NewResourceStatusDependency("PTP instance is not created, waiting for the creation")
			}

			opt2 := ptpinstances.PTPInstToHostOpts{
				PTPInstanceID: &result.ID,
			}
			_, err = ptpinstances.AddPTPInstanceToHost(client, host.ID, opt2).Extract()
			if err != nil {
				err = perrors.Wrapf(err, "failed to add PTP instance to host: %s", host.ID)
				return err
			}
			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
				"ptp instance %s added to host", configured)
			updated = true
		}
	}

	if updated {
		results, err := ptpinstances.ListHostPTPInstances(client, host.ID)
		if err != nil {
			err = perrors.Wrapf(err, "failed to refresh PTP instances from host: %s", host.ID)
			return err
		}

		host.PTPInstances = results
	}

	return nil
}

// ReconcileAttributes is responsible for reconciling the labels on each host.
func (r *HostReconciler) ReconcileLabels(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	updated := false

	// Remove any stale or modified labels
	for _, label := range host.Labels {
		found := false
		for key, value := range profile.Labels {
			if label.Key == key && value == label.Value {
				found = true
				break
			}
		}

		if !found {
			logHost.Info("removing label", "label", label)

			err := labels.Delete(client, label.ID).ExtractErr()
			if err != nil {
				err = perrors.Wrapf(err, "failed to remove label %s", label.ID)
				return err
			}

			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
				"label %q removed", label.Key)

			updated = true
		}
	}

	// Add missing labels
	request := make(map[string]string)
	for k, v := range profile.Labels {
		if value, ok := host.FindLabel(k); !ok || value != v {
			request[k] = v
		}
	}

	if len(request) > 0 {
		logHost.Info("adding labels", "labels", request)

		_, err := labels.Create(client, host.ID, request).Extract()
		if err != nil {
			err = perrors.Wrapf(err, "failed to create labels")
			return err
		}

		keys := make([]string, 0, len(request))
		for k := range request {
			keys = append(keys, k)
		}

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
			"labels %q added", strings.Join(keys, ","))

		updated = true
	}

	if updated {
		result, err := labels.ListLabels(client, host.ID)
		if err != nil {
			err = perrors.Wrap(err, "failed to refresh host labels")
			return err
		}

		host.Labels = result
	}

	return nil
}

// ReconcileInitialState is intended to be run before any other changes are
// reconciled on the host.  Its purpose is to set the administrative state to
// Locked if that is the intended state.  Attribute changes may require this and
// if the operator knows this then they may have set the state to Locked in
// order to change certain attributes.
func (r *HostReconciler) ReconcilePowerState(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	var action string

	if profile.PowerOn == nil {
		return nil
	}

	// NOTE: The "task" is not considered here because we only reconcile hosts
	// that are not currently executing a task

	if *profile.PowerOn {
		if host.AvailabilityStatus == hosts.AvailPowerOff {
			// Only send the power on action if the host is powered off
			action = hosts.ActionPowerOn
		}

	} else if host.AvailabilityStatus != hosts.AvailPowerOff {
		// Only send the power off action if the host is not already powered off
		action = hosts.ActionPowerOff
	}

	if action == "" {
		return nil
	}

	if profile.BoardManagement == nil {
		msg := "board management controller required for power on/off actions"
		r.ReconcilerEventLogger.WarningEvent(instance, common.ResourceDependency, msg)
		return common.NewResourceConfigurationDependency(msg)
	}

	opts := hosts.HostOpts{
		Action: &action,
	}

	logHost.Info("sending action to host", "opts", opts)

	result, err := hosts.Update(client, host.ID, opts).Extract()
	if err != nil || result == nil {
		err = perrors.Wrapf(err, "failed to set power state for host: %s, %s",
			host.ID, common.FormatStruct(opts))
		return err
	}

	host.Host = *result

	r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
		"power-on state has been changed to: %s",
		strconv.FormatBool(*profile.PowerOn))

	// Return a retry result here because we know that it won't be possible to
	// make any other changes until this change is complete.
	return common.NewResourceStatusDependency("waiting for power-on state change")
}

// ReconcileInitialState is intended to be run before any other changes are
// reconciled on the host.  Its purpose is to set the administrative state to
// Locked if that is the intended state.  Attribute changes may require this and
// if the operator knows this then they may have set the state to Locked in
// order to change certain attributes.
func (r *HostReconciler) ReconcileInitialState(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	desiredState := profile.AdministrativeState

	if desiredState != nil && *desiredState != host.AdministrativeState &&
		instance.Status.DeploymentScope == cloudManager.ScopeBootstrap &&
		!r.CloudManager.GetStrategySent() {
		if *desiredState == hosts.AdminLocked {
			action := hosts.ActionLock
			opts := hosts.HostOpts{
				Action: &action,
			}

			logHost.Info("locking host", "opts", opts)

			result, err := hosts.Update(client, host.ID, opts).Extract()
			if err != nil || result == nil {
				err = perrors.Wrapf(err, "failed to lock host: %s, %s",
					host.ID, common.FormatStruct(opts))
				return err
			}

			host.Host = *result

			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
				"host has been locked")

			// Return a retry result here because we know that it won't be possible to
			// make any other changes until this change is complete.
			return common.NewResourceStatusDependency("waiting for host state change in intial state")
		}
	}

	return nil
}

// MinimumEnabledControllerNodesForNonController defines the minimum acceptable
// number of controller nodes that must be enable prior to unlocking a non-
// controller node.
const MinimumEnabledControllerNodesForNonController = 2

// ReconcileFinalState is intended to be run as the last step.  Once all
// configuration changes have been applied it is safe to change the state of the
// host if the desired state is different than the current state.
func (r *HostReconciler) ReconcileFinalState(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	state := profile.AdministrativeState
	if state == nil || *state == host.AdministrativeState ||
		instance.Status.DeploymentScope == cloudManager.ScopePrincipal ||
		instance.Status.StrategyRequired != cloudManager.StrategyNotRequired {
		// No action required.
		return nil
	}

	if *profile.AdministrativeState != hosts.AdminUnlocked {
		// No action required.
		return nil
	}

	personality := profile.Personality
	if *personality == hosts.PersonalityWorker || *personality == hosts.PersonalityStorage {
		if !r.AllControllerNodesEnabled(2) {
			msg := "waiting for all controller nodes to be ready"
			m := NewEnabledControllerNodeMonitor(instance, MinimumEnabledControllerNodesForNonController)
			return r.CloudManager.StartMonitor(m, msg)
		}
	}

	action := hosts.ActionUnlock
	opts := hosts.HostOpts{
		Action: &action,
	}

	logHost.Info("unlocking host", "opts", opts)

	result, err := hosts.Update(client, host.ID, opts).Extract()
	if err != nil || result == nil {
		err = perrors.Wrapf(err, "failed to unlock host: %s, %s",
			host.ID, common.FormatStruct(opts))
		return err
	}

	host.Host = *result

	r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
		"host has been unlocked")

	logHost.Info("finalizing factory install")
	err = r.SetFactoryConfigFinalized(instance.Namespace, true)
	if err != nil {
		return err
	}

	// Return a retry result here because we know that it won't be possible to
	// make any other changes until this change is complete.
	return common.NewResourceStatusDependency("waiting for host state change in final state")
}

func (r *HostReconciler) ReconcileEnabledHost(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {
	err := r.ReconcileInitialState(client, instance, profile, host)
	if err != nil {
		return err
	}

	// The state may have changed in the last step so double check and wait if
	// necessary.
	if !host.IsUnlockedEnabled() {
		msg := "enabled host changed state during reconciliation"
		m := NewStableHostMonitor(instance, host.ID)
		return r.CloudManager.StartMonitor(m, msg)
	}

	switch r.OSDProvisioningState(instance.Namespace, host.Personality) {
	case RequiredStateEnabled, RequiredStateAny:
		err = r.ReconcileOSDs(client, instance, profile, host)
		if err != nil {
			return err
		}
	}

	// Update/Add routes
	err = r.ReconcileRoutes(client, instance, profile, host)
	if err != nil {
		return err
	}

	err = r.ReconcileFileSystemSizes(client, instance, profile, host)
	if err != nil {
		return err
	}

	return nil
}

// ReconcileDisabledHost is responsible for reconciling each individual sub-domain of a
// host resource on a disabled host.
func (r *HostReconciler) ReconcileDisabledHost(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *v1info.HostInfo) error {

	err := r.ReconcileAttributes(client, instance, profile, &host.Host)
	if err != nil {
		return err
	}

	err = r.ReconcilePTPInstances(client, instance, profile, host)
	if err != nil {
		return err
	}

	if profile.HasWorkerSubFunction() {
		// The system API only supports setting these attributes on nodes
		// that support the compute subfunction.

		err = r.ReconcileProcessors(client, instance, profile, host)
		if err != nil {
			return err
		}

	}

	err = r.ReconcileMemory(client, instance, profile, host)
	if err != nil {
		return err
	}

	if profile.HasWorkerSubFunction() {
		err = r.ReconcileKernel(client, instance, profile, host)
		if err != nil {
			return err
		}
	}

	err = r.ReconcileLabels(client, instance, profile, host)
	if err != nil {
		return err
	}

	err = r.ReconcileNetworking(client, instance, profile, host)
	if err != nil {
		return err
	}

	err = r.ReconcileStorage(client, instance, profile, host)
	if err != nil {
		return err
	}

	err = r.ReconcilePowerState(client, instance, profile, host)
	if err != nil {
		return err
	}

	err = r.ReconcileFinalState(client, instance, profile, host)
	if err != nil {
		return err
	}

	return nil
}

// CompareFileSystemTypes determine if there is difference regarding optional
// file system types between two profile specs.
func (r *HostReconciler) CompareFileSystemTypes(in *starlingxv1.HostProfileSpec, other *starlingxv1.HostProfileSpec) bool {
	if other == nil {
		return false
	}

	if (in.Storage == nil) && (other.Storage == nil) {
		return true
	} else if in.Storage != nil {
		if in.Storage.DeepEqual(other.Storage) {
			// The full storage profile matches therefore the file systems match.
			return true
		}

		configured := []string{}
		current := []string{}

		if in.Storage.FileSystems != nil {
			for _, fsInfo := range *in.Storage.FileSystems {
				configured = append(configured, fsInfo.Name)
			}
		}

		if other.Storage != nil {
			if other.Storage.FileSystems != nil {
				for _, fs := range *other.Storage.FileSystems {
					current = append(current, fs.Name)
				}
			}
		}

		// Find difference of file system types to add or remove
		added, removed, _ := utils.ListDelta(current, configured)
		_, _, fs_to_add := utils.ListDelta(added, FileSystemCreationAllowed)
		_, _, fs_to_remove := utils.ListDelta(removed, FileSystemDeletionAllowed)

		if len(fs_to_remove) > 0 || len(fs_to_add) > 0 {
			return false
		}
	}
	return true
}

// CompareOSDs determine if there has been a change to the list of OSDs between
// two profile specs. This method takes into consideration that the storage
// section may be completely empty on either side of the comparison.
func (r *HostReconciler) CompareOSDs(in *starlingxv1.HostProfileSpec, other *starlingxv1.HostProfileSpec) bool {
	if other == nil {
		return false
	}

	if (in.Storage == nil) && (other.Storage == nil) {
		return true

	} else if in.Storage != nil {
		if in.Storage.DeepEqual(other.Storage) {
			// The full storage profile matches therefore the OSDs match.
			return true
		}

		if in.Storage.OSDs != nil {
			// Otherwise just check the OSD list and ignore the other attributes.
			if !in.Storage.OSDs.DeepEqual(other.Storage.OSDs) {
				return false
			}
		} else if other.Storage.OSDs != nil && len(*other.Storage.OSDs) > 0 {
			return false
		}

	} else if other.Storage.OSDs != nil && len(*other.Storage.OSDs) > 0 {
		return false
	}

	return true
}

// CompareAttributes determines if two profiles are identical for the
// purpose of reconciling a current host configuration to its desired host
// profile.
func (r *HostReconciler) CompareAttributes(in *starlingxv1.HostProfileSpec, other *starlingxv1.HostProfileSpec, instance *starlingxv1.Host, personality string, system_info *cloudManager.SystemInfo) bool {
	// This could be replaced with in.DeepEqual(other) but it is coded this way
	// (and tested this way) to ensure that if both the "enabled" and "disabled"
	// comparisons are true then no reconciliation is missed.  The intent is
	// that CompareEnabledAttributes && CompareDisabledAttributes will always
	// be equivalent to DeepEqual.
	return r.CompareEnabledAttributes(in, other, instance, personality, system_info) &&
		r.CompareDisabledAttributes(in, other, instance.Namespace, personality, false)
}

// CompareEnabledAttributes determines if two profiles are identical for the
// purpose of reconciling any attributes that can only be applied when the host
// is enabled.  The only attributes that we can reconcile while enabled are the
// storage OSD resources therefore return false if there are any differences
// in the storage OSD list.
func (r *HostReconciler) CompareEnabledAttributes(in *starlingxv1.HostProfileSpec, other *starlingxv1.HostProfileSpec, instance *starlingxv1.Host, personality string, system_info *cloudManager.SystemInfo) bool {
	if other == nil {
		return false
	}

	if instance.Status.DeploymentScope != cloudManager.ScopePrincipal && in.AdministrativeState != nil {
		if (in.AdministrativeState == nil) != (other.AdministrativeState == nil) {
			return false
		} else if in.AdministrativeState != nil {
			if *in.AdministrativeState != *other.AdministrativeState {
				return false
			}
		}
	}

	if utils.IsReconcilerEnabled(utils.OSD) {
		switch r.OSDProvisioningState(instance.Namespace, personality) {
		case RequiredStateEnabled, RequiredStateAny:
			if !r.CompareOSDs(in, other) {
				return false
			}
		}
	}

	if utils.IsReconcilerEnabled(utils.FileSystemSizes) {
		if in.Storage != nil && in.Storage.FileSystems != nil {
			if other.Storage == nil || other.Storage.FileSystems == nil {
				return false
			}

			// Special case: 'ceph' filesystem must be ignored for All-in-one Duplex.
			if system_info.SystemType == cloudManager.SystemTypeAllInOne && system_info.SystemMode != cloudManager.SystemModeSimplex {
				for _, fsInfo := range *other.Storage.FileSystems {
					// Skip ceph filesystem
					if fsInfo.Name == "ceph" {
						continue
					}

					// Compare 'in' and 'other' fileystems
					found := false
					for _, fs := range *in.Storage.FileSystems {
						if fs.Name != fsInfo.Name {
							continue
						}

						found = true
						if fsInfo.Size != fs.Size {
							return false
						}
					}

					// If not found, return false
					if !found {
						return false
					}
				}
			} else {
				// Do a deep compare when this is not an All-in-one Duplex
				if !in.Storage.FileSystems.DeepEqual(other.Storage.FileSystems) {
					return false
				}
			}
		}
	}

	if utils.IsReconcilerEnabled(utils.Route) {
		if !in.Routes.DeepEqual(&other.Routes) {
			return false
		}
	}

	return true
}

// CompareDisabledAttributes determines if two profiles are identical for the
// purpose of reconciling any attributes that can only be applied when the host
// is disabled.
func (r *HostReconciler) CompareDisabledAttributes(in *starlingxv1.HostProfileSpec, other *starlingxv1.HostProfileSpec, namespace, personality string, reconfig bool) bool {
	if other == nil {
		return false
	}

	// In cases of reconfiguration(day2 operation copy AdministrativeState temporary
	// to ignore difference. Put original value back after DeepEqual
	adminState := other.ProfileBaseAttributes.AdministrativeState
	if reconfig {
		other.ProfileBaseAttributes.AdministrativeState = in.ProfileBaseAttributes.AdministrativeState
	}
	if !in.ProfileBaseAttributes.DeepEqual(&other.ProfileBaseAttributes) {
		if reconfig {
			other.ProfileBaseAttributes.AdministrativeState = adminState
		}
		return false
	}
	if reconfig {
		other.ProfileBaseAttributes.AdministrativeState = adminState
	}

	if (in.BoardManagement == nil) != (other.BoardManagement == nil) {
		return false
	} else if in.BoardManagement != nil {
		if !in.BoardManagement.DeepEqual(other.BoardManagement) {
			return false
		}
	}

	if utils.IsReconcilerEnabled(utils.Memory) {
		if !in.Memory.DeepEqual(&other.Memory) {
			return false
		}
	}

	if utils.IsReconcilerEnabled(utils.Processor) {
		if !in.Processors.DeepEqual(&other.Processors) {
			return false
		}
	}

	if utils.IsReconcilerEnabled(utils.Networking) {
		if utils.IsReconcilerEnabled(utils.Interface) {
			if (in.Interfaces == nil) != (other.Interfaces == nil) {
				return false
			} else if in.Interfaces != nil {
				if !in.Interfaces.DeepEqual(other.Interfaces) {
					return false
				}
			} else {
				return false
			}
		}

		if utils.IsReconcilerEnabled(utils.Address) {
			if !in.Addresses.DeepEqual(&other.Addresses) {
				return false
			}
		}

		if !reconfig && utils.IsReconcilerEnabled(utils.Route) {
			if !in.Routes.DeepEqual(&other.Routes) {
				return false
			}
		}
	}

	if utils.IsReconcilerEnabled(utils.FileSystemTypes) {
		if !r.CompareFileSystemTypes(in, other) {
			return false
		}
	}

	if utils.IsReconcilerEnabled(utils.OSD) {
		switch r.OSDProvisioningState(namespace, personality) {
		case RequiredStateDisabled, RequiredStateAny:
			if !r.CompareOSDs(in, other) {
				return false
			}
		}
	}

	return true
}

// ReconcileHostByState is responsible for differentiating between an enabled
// host and a disabled host.  Most attributes only support being updated when
// the host is in a certain state therefore those differences are discriminated
// here.
func (r *HostReconciler) ReconcileHostByState(
	client *gophercloud.ServiceClient,
	instance *starlingxv1.Host,
	current *starlingxv1.HostProfileSpec,
	profile *starlingxv1.HostProfileSpec,
	host *v1info.HostInfo,
	system_info *cloudManager.SystemInfo,
) error {

	principal := false
	// Other than day2 changes to host resource VIM strategy on host could be updated
	// by the platform network reconciler to perform management network reconfiguration.
	// During management network reconfig, strategy_required is set to 'true' so that
	// host reconciler can fix broken interface-network association and unlock the host.
	strategy_required := false
	if instance.Status.DeploymentScope == cloudManager.ScopePrincipal {
		principal = true
	}

	if instance.Status.StrategyRequired != cloudManager.StrategyNotRequired {
		strategy_required = true
	}

	if host.IsUnlockedEnabled() {
		if !r.CompareEnabledAttributes(profile, current, instance, host.Personality, system_info) {
			err := r.ReconcileEnabledHost(client, instance, profile, host)
			if err != nil {
				return err
			}
		} else {
			logHost.Info("no enabled attribute changes required")
		}

		if !r.CompareDisabledAttributes(profile, current, instance.Namespace, host.Personality, principal) {
			if principal || strategy_required {
				if interfaceName, commonInterfaceInfo, hasChange := hasAdminNetworkChange(profile.Interfaces, current.Interfaces); hasChange {
					logHost.Info("Has admin network multi-netting changes", "interface", interfaceName)
					iface, found := host.FindInterfaceByName(interfaceName)
					if !found {
						msg := fmt.Sprintf("unable to find interface: %s", interfaceName)
						return starlingxv1.NewMissingSystemResource(msg)
					}

					// We need to reconcile the interfaces to assign the admin network
					_, err := r.ReconcileInterfaceNetworks(client, instance, *commonInterfaceInfo, *iface, host)
					if err != nil {
						return err
					}

					// The platformnetwork_controller will create the address pool and
					// the network; the controller just needs to assign the network to
					// the interface. If any other changes require lock/unlock, then it
					// will do so afterward.
					return nil
				} else {
					instance.Status.StrategyRequired = cloudManager.StrategyLockRequired
					logHost.V(2).Info("set lock required")
				}
				r.CloudManager.SetResourceInfo(cloudManager.ResourceHost, host.Personality, instance.Name, instance.Status.Reconciled, instance.Status.StrategyRequired)
				err := r.Client.Status().Update(context.TODO(), instance)
				if err != nil {
					err = perrors.Wrapf(err, "failed to update status: %s",
						common.FormatStruct(instance.Status))
					return err
				}
			}

			msg := "waiting for locked state before applying out-of-service attributes"
			m := NewLockedDisabledHostMonitor(instance, host.ID)
			return r.CloudManager.StartMonitor(m, msg)
		}

		// Clean up strategy required after Disabled and enabled attributes are all in-sync
		if strategy_required &&
			!(r.CloudManager.GetStrategyExpectedByOtherReconcilers() || r.CloudManager.GetStrategySent()) {
			logHost.V(2).Info("set strategy not required as attributes are all configured")
			instance.Status.StrategyRequired = cloudManager.StrategyNotRequired
			r.CloudManager.SetResourceInfo(cloudManager.ResourceHost, host.Personality, instance.Name, instance.Status.Reconciled, instance.Status.StrategyRequired)
			err := r.Client.Status().Update(context.TODO(), instance)
			if err != nil {
				err = perrors.Wrapf(err, "failed to update status: %s",
					common.FormatStruct(instance.Status))
				return err
			}
		}

	} else if host.IsLockedDisabled() {
		if !r.CompareDisabledAttributes(profile, current, instance.Namespace, host.Personality, principal) {
			err := r.ReconcileDisabledHost(client, instance, profile, host)
			if err != nil {
				return err
			}
		} else {
			logHost.Info("no disabled attribute changes required")
		}

		// As disabled attributes are configured, ready to unlock the system.
		if strategy_required {
			instance.Status.StrategyRequired = cloudManager.StrategyUnlockRequired
			logHost.V(2).Info("set unlock required. Lock required attributes are configured")
			r.CloudManager.SetResourceInfo(cloudManager.ResourceHost, host.Personality, instance.Name, instance.Status.Reconciled, instance.Status.StrategyRequired)
			err := r.Client.Status().Update(context.TODO(), instance)
			if err != nil {
				err = perrors.Wrapf(err, "failed to update status: %s",
					common.FormatStruct(instance.Status))
				return err
			}
		}
		msg := "waiting for the unlocked state before applying in-service attributes"
		m := NewUnlockedEnabledHostMonitor(instance, host.ID)
		return r.CloudManager.StartMonitor(m, msg)
	} else {
		msg := "waiting for a stable state in unknown state"
		m := NewStableHostMonitor(instance, host.ID)
		return r.CloudManager.StartMonitor(m, msg)
	}

	return nil
}

// hasAdminNetworkChange checks whether the addition of "admin" to PlatformNetworks
// within an existing (multi-netting) EthernetInfo, VLANInfo and BondInfo is present
// in the given profile and current InterfaceInfo instances.
func hasAdminNetworkChange(profileInterfaces, currentInterfaces *starlingxv1.InterfaceInfo) (string, *starlingxv1.CommonInterfaceInfo, bool) {
	if profileInterfaces == nil || currentInterfaces == nil {
		return "", nil, false
	}

	if ethInfo, hasChange := hasEthernetAdminNetworkChange(profileInterfaces, currentInterfaces); hasChange {
		return ethInfo.Name, &ethInfo.CommonInterfaceInfo, true
	} else if VLANInfo, hasChange := hasVLANAdminNetworkChange(profileInterfaces, currentInterfaces); hasChange {
		return VLANInfo.Name, &VLANInfo.CommonInterfaceInfo, true
	} else if BondInfo, hasChange := hasBondAdminNetworkChange(profileInterfaces, currentInterfaces); hasChange {
		return BondInfo.Name, &BondInfo.CommonInterfaceInfo, true
	} else {
		return "", nil, false
	}
}

func hasEthernetAdminNetworkChange(profileInterfaces, currentInterfaces *starlingxv1.InterfaceInfo) (*starlingxv1.EthernetInfo, bool) {
	for _, profileEth := range profileInterfaces.Ethernet {
		logHost.V(2).Info("Profile Ethernet Name", "name", profileEth.Name)
		currentEth := findEthernetInfoByName(currentInterfaces.Ethernet, profileEth.Name)
		logHost.V(2).Info("Current ethernetinfo", "ethernet", currentEth)
		if currentEth == nil {
			continue
		}

		if !reflect.DeepEqual(profileEth.PlatformNetworks, currentEth.PlatformNetworks) {
			return &profileEth, containsAdminPlatformNetwork(*profileEth.PlatformNetworks)
		}
	}

	return nil, false
}

func hasVLANAdminNetworkChange(profileInterfaces, currentInterfaces *starlingxv1.InterfaceInfo) (*starlingxv1.VLANInfo, bool) {
	for _, profileVLAN := range profileInterfaces.VLAN {
		logHost.V(2).Info("Profile VLAN Name", "name", profileVLAN.Name)
		currentVLAN := findVLANInfoByName(currentInterfaces.VLAN, profileVLAN.Name)
		logHost.V(2).Info("Current VLANInfo", "VLAN", currentVLAN)
		if currentVLAN == nil {
			continue
		}

		if !reflect.DeepEqual(profileVLAN.PlatformNetworks, currentVLAN.PlatformNetworks) {
			return &profileVLAN, containsAdminPlatformNetwork(*profileVLAN.PlatformNetworks)
		}
	}

	return nil, false
}

func hasBondAdminNetworkChange(profileInterfaces, currentInterfaces *starlingxv1.InterfaceInfo) (*starlingxv1.BondInfo, bool) {
	for _, profileBond := range profileInterfaces.Bond {
		logHost.V(2).Info("Profile Bond Name", "name", profileBond.Name)
		currentBond := findBondInfoByName(currentInterfaces.Bond, profileBond.Name)
		logHost.V(2).Info("Current bondinfo", "bond", currentBond)
		if currentBond == nil {
			continue
		}

		if !reflect.DeepEqual(profileBond.PlatformNetworks, currentBond.PlatformNetworks) {
			return &profileBond, containsAdminPlatformNetwork(*profileBond.PlatformNetworks)
		}
	}

	return nil, false
}

// findEthernetInfoByName is a utility function to locate an EthernetInfo in a list
// by its name.
func findEthernetInfoByName(ethList []starlingxv1.EthernetInfo, name string) *starlingxv1.EthernetInfo {
	for i := range ethList {
		if ethList[i].Name == name {
			return &ethList[i]
		}
	}
	return nil
}

// findVLANInfoByName is a utility function to locate an VLANInfo in a list by its name.
func findVLANInfoByName(VLANList []starlingxv1.VLANInfo, name string) *starlingxv1.VLANInfo {
	for i := range VLANList {
		if VLANList[i].Name == name {
			return &VLANList[i]
		}
	}
	return nil
}

// findBondInfoByName is a utility function to locate an BondInfo in a list by its name.
func findBondInfoByName(bondList []starlingxv1.BondInfo, name string) *starlingxv1.BondInfo {
	for i := range bondList {
		if bondList[i].Name == name {
			return &bondList[i]
		}
	}
	return nil
}

// containsAdminPlatformNetwork is a utility function to determine if "admin" has
// been added to the PlatformNetworks list.
func containsAdminPlatformNetwork(platformNetworks starlingxv1.PlatformNetworkItemList) bool {
	for _, network := range platformNetworks {
		if strings.ToLower(string(network)) == cloudManager.AdminNetworkType {
			return true
		}
	}
	return false
}

// statusUpdateRequired is a utility function which determines whether an update
// is required to the host status attribute.  Updating this unnecessarily
// will result in an infinite reconciliation loop.
func (r *HostReconciler) statusUpdateRequired(instance *starlingxv1.Host, host *hosts.Host, inSync bool) (result bool) {
	status := &instance.Status

	if status.ID == nil || *status.ID != host.ID {
		status.ID = &host.ID
		// If the ID is being set or changed then make sure the defaults are
		// reset back to nil so that the host is re-inventoried before being
		// configured.
		status.Defaults = nil
		result = true
	}

	if status.AdministrativeState == nil || *status.AdministrativeState != host.AdministrativeState {
		status.AdministrativeState = &host.AdministrativeState
		result = true
	}

	if status.OperationalStatus == nil || *status.OperationalStatus != host.OperationalStatus {
		status.OperationalStatus = &host.OperationalStatus
		result = true
	}

	if status.AvailabilityStatus == nil || *status.AvailabilityStatus != host.AvailabilityStatus {
		status.AvailabilityStatus = &host.AvailabilityStatus
		result = true
	}

	if status.InSync != inSync {
		status.InSync = inSync
		result = true
	}

	logHost.V(2).Info("Current Status", "status", status)
	strategyUpdated := false

	if status.InSync && status.Reconciled && host.IsUnlockedAvailable() &&
		status.StrategyRequired == cloudManager.StrategyUnlockRequired {
		// Already unlocked and inSync, set strategy not required
		logHost.V(2).Info("Unlocked, set strategy not required")
		status.StrategyRequired = cloudManager.StrategyNotRequired
		strategyUpdated = true
		result = true
	}

	if status.InSync && !status.Reconciled {
		// Record the fact that we have reached inSync at least once.
		logHost.V(2).Info("Insync=true so that change reconciled=true")
		if host.IsLockedDisabled() && (status.ConfigurationUpdated || status.HostProfileConfigurationUpdated) &&
			status.StrategyRequired == cloudManager.StrategyLockRequired {
			// Unlock if current is locked
			logHost.V(2).Info("host inSync, set unlock required")
			status.StrategyRequired = cloudManager.StrategyUnlockRequired
			strategyUpdated = true
		} else if status.StrategyRequired != cloudManager.StrategyNotRequired &&
			!(r.CloudManager.GetStrategyExpectedByOtherReconcilers() || r.CloudManager.GetStrategySent()) {
			logHost.V(2).Info("set not required: reconcile finished")
			status.StrategyRequired = cloudManager.StrategyNotRequired
			strategyUpdated = true
		}
		status.Reconciled = true
		status.ConfigurationUpdated = false
		status.HostProfileConfigurationUpdated = false
		logHost.V(2).Info("set profile config updated false: reconcile finished")
		// Update resource info for Day-2 operation
		if strategyUpdated {
			r.CloudManager.SetResourceInfo(cloudManager.ResourceHost, host.Personality, instance.Name, status.Reconciled, status.StrategyRequired)
		}
		result = true
	}

	if status.Defaults == nil {
		logHost.Info("defaults is nil. Update status")
		result = true
	}

	return result
}

// findExistingHost searches the current list of hosts and attempts to find one
// that fits the provided match criteria.
func FindExistingHost(objects []hosts.Host, hostname string, match *starlingxv1.MatchInfo, bootMAC *string) *hosts.Host {
	for _, host := range objects {
		if host.Hostname != "" && host.Hostname == hostname {
			// Forgo the match criteria if the hostname is a match.
			return &host
		}

		if hostMatchesCriteria(host, match) {
			// The host satisfies the match criteria, but as an additional
			// sanity check of the data we need to make sure that the
			// hostname matches as well.  This is to help avoid typos that
			// cause the system to be misconfigured which might be difficult
			// to recover from.
			if host.Hostname == "" || host.Hostname == hostname {
				return &host
			}
		}

		if bootMAC != nil && host.BootMAC == *bootMAC {
			// For static provisioning, the boot MAC is specified rather than a
			// match criteria therefore check to see if it is already present
			// which may be possible if the end user proactively powered on the
			// host.
			return &host
		}
	}

	return nil
}

// ReconcileNewHost is responsible for dealing with the initial provisioning of
// a host. This handles both static and dynamic provisioning of hosts.  If a
// new host is created then the 'host' return parameter will be updated with a
// pointer to the new host object.
func (r *HostReconciler) ReconcileNewHost(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec) (host *hosts.Host, err error) {
	host = FindExistingHost(r.hosts, instance.Name, instance.Spec.Match, profile.BootMAC)
	if host != nil {
		logHost.Info("found matching host", "id", host.ID)
	}

	if host == nil {
		// A new host needs to be provisioned or we need to wait for one to
		// appear in the system.
		if *profile.ProvisioningMode != starlingxv1.ProvioningModeStatic {
			// We only create missing hosts for statically provisioned hosts.
			// For dynamic, hosts we wait for them to appear in the system
			msg := "waiting for dynamic host to appear in inventory"
			m := NewDynamicHostMonitor(instance, instance.Name, instance.Spec.Match, profile.BootMAC)
			return nil, r.CloudManager.StartMonitor(m, msg)

		} else if r.ProvisioningAllowed() {
			// Populate a new host into system inventory.
			if instance.Status.Reconciled && r.StopAfterInSync() {
				// Do not process any further changes once we have reached a
				// synchronized state unless there is an annotation on the host.
				if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
					msg := common.NoProvisioningAfterReconciled
					r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, msg)
					return nil, common.NewChangeAfterInSync(msg)
				} else {
					logHost.Info(common.ProvisioningAllowedAfterReconciled)
				}
			}

			opts, err := r.buildInitialHostOpts(instance, profile)
			if err != nil {
				return nil, err // Already logged
			}

			logHost.Info("creating host", "opts", opts)

			host, err = hosts.Create(client, opts).Extract()
			if err != nil || host == nil {
				err = perrors.Wrapf(err, "failed to create: %s",
					common.FormatStruct(opts))
				return nil, err
			}

			if profile.BoardManagement != nil && (profile.PowerOn != nil && *profile.PowerOn) {
				// Attempt to power-on the host; otherwise the user will need
				// to do this manually.
				action := hosts.ActionReinstall
				opts = hosts.HostOpts{Action: &action}
				host, err = hosts.Update(client, host.ID, opts).Extract()
				if err != nil {
					err = perrors.Wrapf(err, "failed to power-on host")
					return nil, err
				}
			}

			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceCreated,
				"static host has been created")

		} else {
			msg := "waiting for system to allow creating static hosts"
			m := NewProvisioningAllowedMonitor(instance)
			return nil, r.CloudManager.StartMonitor(m, msg)
		}

	} else if host.Hostname == "" {
		// The host was found but it has not been provisioned with a hostname
		// and personality so set up its initial attributes.
		if r.ProvisioningAllowed() {
			logHost.Info("setting initial attributes")
			err := r.ReconcileAttributes(client, instance, profile, host)
			if err != nil {
				return host, err
			}

		} else {
			msg := "waiting for system to allow host provisioning"
			m := NewProvisioningAllowedMonitor(instance)
			return host, r.CloudManager.StartMonitor(m, msg)
		}
	}

	return host, nil
}

// StopAfterInSync determines whether the reconciler should continue processing
// change requests after the configuration has been reconciled a first time.
func (r *HostReconciler) StopAfterInSync() bool {
	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return utils.GetReconcilerOptionBool(utils.Host, utils.StopAfterInSync, true)
}

// ReconcileExistingHost is responsible for dealing with the provisioning of an
// existing host.
func (r *HostReconciler) ReconcileExistingHost(client *gophercloud.ServiceClient, instance *starlingxv1.Host, profile *starlingxv1.HostProfileSpec, host *hosts.Host, reqNs string) error {
	logHost.Info("Starting to configure on existing host", "host", host.ID)

	var defaults *starlingxv1.HostProfileSpec
	var current *starlingxv1.HostProfileSpec
	var platform_network_subreconciler_errs []error

	if !host.Stable() {
		msg := "waiting for a stable state for existing host"
		m := NewStableHostMonitor(instance, host.ID)
		return r.CloudManager.StartMonitor(m, msg)
	}

	// Gather all host attributes so that they can be reused by various
	// functions without needing to be re-queried each time.
	logHost.V(2).Info("gathering host information", "host", host.ID)
	hostInfo := v1info.HostInfo{}
	err := hostInfo.PopulateHostInfo(client, host.ID)
	if err != nil {
		return err
	}

	// Fetch default attributes so that they can be used to back sparse host
	// profile configurations.
	logHost.V(2).Info("fetching default host attributes", "host", host.ID)
	defaults, err = r.GetHostDefaults(instance)
	if err != nil {
		return err
	}

	// Check factory install config map
	logHost.V(2).Info("checking factory install config map", "namespace", reqNs)
	factory, err := r.CloudManager.GetFactoryInstall(reqNs)
	if err != nil {
		return perrors.Wrap(err, "failed to get factory install config map")
	}

	if factory && host.IsUnlockedEnabled() {
		// finalize factory install if host already unlocked
		logHost.Info("finalizing factory install", "host", host.ID)
		err = r.SetFactoryConfigFinalized(instance.Namespace, true)
		if err != nil {
			return perrors.Wrap(err, "failed to finalize factory config")
		}
		factory = false
	}

	// Determine if we need to update the default configuration
	logHost.V(2).Info("checking if update to defaults is required", "host", host.ID)
	updatedRequired, err := common.UpdateDefaultsRequired(
		r.CloudManager,
		reqNs,
		instance.Name,
		factory,
	)
	if err != nil {
		return perrors.Wrap(err, "failed to check if update to defaults is required")
	}

	if defaults == nil || updatedRequired {
		logHost.Info("defaults are nil or update required", "host", host.ID)
		if !host.Stable() || host.AvailabilityStatus == hosts.AvailOffline {
			// Ideally we would only ever collect the defaults when the host is
			// in the locked/disabled/online state.  This is the best approach
			// when provisioning a system from scratch, but for cases where
			// an operator may want to start with a partially configured system
			// then using any stable state is sufficient.
			msg := "waiting for a stable state before collecting defaults"
			m := NewInventoryCollectedMonitor(instance, host.ID)
			return r.CloudManager.StartMonitor(m, msg)
		}

		if !hostInfo.IsInventoryCollected() {
			msg := "waiting for inventory collection to complete before collecting defaults"
			m := NewInventoryCollectedMonitor(instance, host.ID)
			return r.CloudManager.StartMonitor(m, msg)
		}

		logHost.Info("collecting default values for host", "host", host.ID)
		defaults, err = r.BuildHostDefaults(instance, hostInfo)
		if err != nil {
			return err
		}

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceCreated,
			"defaults collected and stored")

		if factory {
			logHost.V(2).Info("setting factory resource data updated", "namespace", reqNs, "host", host.ID)
			err := r.CloudManager.SetFactoryResourceDataUpdated(
				reqNs,
				instance.Name,
				"default",
				true,
			)
			if err != nil {
				return perrors.Wrap(err, "failed to set factory resource data updated")
			}
		}

		current = defaults.DeepCopy()
	} else {
		// Otherwise, the defaults already existed so build a new profile with
		// the current host configuration so that we can compare it to the
		// desired state.
		logHost.V(2).Info("building current profile from current config", "host", host.ID)
		current, err = starlingxv1.NewHostProfileSpec(hostInfo)
		if err != nil {
			return perrors.Wrap(err, "failed to build current profile")
		}
	}

	// NOTE(alegacy): The defaults collected may include BMC information and
	// since the API does not return any password info it is not possible to
	// build a true representation of the system state.  Trying to reconcile
	// the system generated BMC info will always result in an error because
	// it does not contain password info.
	bmType := "none"
	bmInfo := starlingxv1.BMInfo{
		Type:        &bmType,
		Address:     nil,
		Credentials: nil,
	}
	defaults.BoardManagement = &bmInfo

	// Create a new composite profile that is backed by the host's default
	// configuration.  This will ensure that if a user deletes an optional
	// attribute that we will know how to restore the original value.
	logHost.Info("merging profiles", "host", host.ID)
	profile, err = MergeProfiles(defaults, profile)
	if err != nil {
		return perrors.Wrap(err, "failed to merge profiles")
	}

	// Fix attributes in profiles to a uniformed format
	// As the Merge Profiles will overwrite some format in the default profile
	// parsed in the constructor, move this process after it.
	logHost.V(2).Info("fixing profile attributes", "host", host.ID)
	FixProfileAttributes(defaults, profile, current, &hostInfo)
	FillEmptyUuidbyName(defaults, current)

	// TODO(alegacy): Need to move ProvisioningMode out of the profile or
	//  find a way to populate it into profiles generated from the running
	//  configuration.
	profile.ProvisioningMode = nil

	// N3000 interface name change apply
	if host.IsUnlockedEnabled() {
		logHost.Info("syncing interface name", "host", host.ID)
		SyncIFNameByUuid(profile, current)
	}

	is_active_host, err := r.IsActiveHost(client, instance, reqNs)
	if err != nil {
		return err
	}

	if is_active_host {
		logHost.Info("host is active, reconciling platform networks", "host", host.ID)
		system_info, err := r.CloudManager.GetSystemInfo(reqNs, client)
		if err != nil {
			return common.NewUserDataError("failed to get system info")
		}

		// Only reconcile platform networks in active controller reconciliation
		// loops to prevent concurrency issues.
		// Also within active controller prevent potential concurrent reconciliation
		// if IsPlatformNetworkReconciling.
		if !r.CloudManager.IsPlatformNetworkReconciling() {
			r.CloudManager.SetPlatformNetworkReconciling(true)
			// Do not allow notifying active host until ReconcilePlatformNetworks
			// is completed, this is to prevent unnecessary status update conflicts
			// in addrpools and platform network resources.
			r.CloudManager.SetNotifyingActiveHost(true)

			logHost.Info("starting platform network reconciliation", "host", host.ID)
			platform_network_subreconciler_errs = r.ReconcilePlatformNetworks(client, instance, profile, &hostInfo, system_info)

			r.CloudManager.SetPlatformNetworkReconciling(false)
			r.CloudManager.SetNotifyingActiveHost(false)

			for _, err := range platform_network_subreconciler_errs {
				cause := perrors.Cause(err)
				if _, ok := cause.(cloudManager.WaitForMonitor); ok {
					// If there is a WaitForMonitor error the reconciliation
					// request should not be requeued.
					// Reconciliation will be triggered after change in state
					// of the host by the monitor itself.
					return err
				}
			}

			if len(platform_network_subreconciler_errs) != 0 {
				err_msg := "there were errors during platform network reconciliation"
				return common.NewPlatformNetworkReconciliationError(err_msg)
			}
		} else {
			err_msg := "waiting for platform network reconciled."
			return common.NewPlatformNetworkReconciliationError(err_msg)
		}
	}

	// Fetch system information for further comparison
	logHost.V(2).Info("fetching system info for attribute comparison", "namespace", reqNs)
	system_info, err := r.CloudManager.GetSystemInfo(reqNs, client)
	if err != nil {
		msg := "failed to get system info"
		return common.NewUserDataError(msg)
	}

	logHost.Info("comparing profile attributes", "host", host.ID)
	instance.Status.InSync = r.CompareAttributes(profile, current, instance, host.Personality, system_info)
	common.SetInstanceDelta(instance, profile, current, common.HostProperties, r.Client.Status(), logHost)

	// strategy not finished
	if instance.Status.InSync && (instance.Status.StrategyRequired == cloudManager.StrategyNotRequired) {
		logHost.V(2).Info("no changes between composite profile and current config", "host", host.ID)
		return nil
	}

	logHost.Info("defaults are:", "values", defaults)
	logHost.Info("final profile is:", "values", profile)
	logHost.Info("current config is:", "values", current)

	if instance.Status.Reconciled &&
		r.StopAfterInSync() &&
		instance.Status.StrategyRequired != cloudManager.StrategyLockRequired {
		if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
			if !host.IsUnlockedAvailable() {
				msg := "waiting for the host reach available state"
				m := NewUnlockedAvailableHostMonitor(instance, host.ID)
				return r.CloudManager.StartMonitor(m, msg)
			} else {
				// Do not process any further changes once we have reached a
				// synchronized state unless there is an annotation on the host.
				msg := common.NoChangesAfterReconciled
				r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, msg)
				return common.NewChangeAfterInSync(msg)
			}
		} else {
			logHost.Info(common.ChangedAllowedAfterReconciled)
		}
	}

	logHost.Info("reconciling host by state", "host", host.ID)
	err = r.ReconcileHostByState(client, instance, current, profile, &hostInfo, system_info)
	if err != nil {
		return err
	}

	logHost.V(2).Info("final configuration is set", "profile", current)

	return nil
}

// ReconcileDeletedHost is responsible for dealing with the provisioning of an
// existing host.
func (r *HostReconciler) ReconcileDeletedHost(client *gophercloud.ServiceClient, instance *starlingxv1.Host, host *hosts.Host) (err error) {
	if host.Capabilities.Personality != nil {
		if strings.EqualFold(*host.Capabilities.Personality, hosts.ActiveController) {
			// Always leave the active controller installed.
			logHost.Info("skipping delete for active controller")
			return nil
		}
	}

	if !host.Stable() {
		msg := "waiting for a stable state before deleting host"
		m := NewStableHostMonitor(instance, host.ID)
		return r.CloudManager.StartMonitor(m, msg)
	}

	if !host.IsLockedDisabled() {
		action := hosts.ActionLock
		opts := hosts.HostOpts{Action: &action}

		logHost.Info("locking host", "opts", opts)

		result, err := hosts.Update(client, host.ID, opts).Extract()
		if err != nil {
			err = perrors.Wrap(err, "failed to lock host")
			return err
		}
		*host = *result

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "host has been locked")
	}

	if !host.IsLockedDisabled() {
		// Host is still not locked so wait for the action to complete.
		msg := "waiting for host to lock before deleting it"
		m := NewLockedDisabledHostMonitor(instance, host.ID)
		return r.CloudManager.StartMonitor(m, msg)
	}

	logHost.Info("deleting host")

	err = hosts.Delete(client, host.ID).ExtractErr()
	if err != nil {
		err = perrors.Wrapf(err, "failed to delete host: %s", host.ID)
		return err
	}

	r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceDeleted, "host has been deleted")

	return nil
}

// removing the HostFinalizer from the host resource
func (r *HostReconciler) removeHostFinalizer(instance *starlingxv1.Host) {
	instance.ObjectMeta.Finalizers = utils.RemoveString(instance.ObjectMeta.Finalizers, HostFinalizerName)
	if err := r.Client.Update(context.Background(), instance); err != nil {
		logHost.Error(err, "failed to remove the finalizer in the host because of the error:%v")
	}
}

// ReconcileResource interacts with the system API in order to reconcile the
// state of a data network with the state stored in the k8s database.
func (r *HostReconciler) ReconcileResource(
	client *gophercloud.ServiceClient,
	instance *starlingxv1.Host,
	profile *starlingxv1.HostProfileSpec,
	reqNs string) (err error) {
	var host *hosts.Host
	var inSync bool

	id := instance.Status.ID
	if id != nil && *id != "" {
		// This host was previously provisioned so check that it still exists
		// as the same uuid value; otherwise it may have been deleted and
		// re-added so we will need to deal with that scenario.
		host, err = hosts.Get(client, *id).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); !ok {
				if !instance.DeletionTimestamp.IsZero() {
					if utils.ContainsString(instance.ObjectMeta.Finalizers, HostFinalizerName) {
						// Remove the finalizer
						r.removeHostFinalizer(instance)
					}
				}
				err = perrors.Wrapf(err, "failed to get: %s", *id)
				return err
			}

			// The resource may have been deleted by the system or operator
			// therefore continue and attempt to recreate it.
			logHost.Info("resource no longer exists", "id", *id)

			// Set host to nil, in case hosts.Get() returned a partially populated structure
			host = nil
		}
	}

	if !instance.DeletionTimestamp.IsZero() {
		if utils.ContainsString(instance.ObjectMeta.Finalizers, HostFinalizerName) {

			// Remove the finalizer so we don't try to do this delete action again.
			// Defer the removal of the finalizer until the end of the function
			defer r.removeHostFinalizer(instance)

			// A finalizer is still present so we need to try to delete the
			// host from the system.
			if host != nil {
				err = r.ReconcileDeletedHost(client, instance, host)
				if err != nil {
					return err
				}
			} else {
				logHost.Info("host being deleted is no longer present on system")
			}
		}

		// Remove deleted host from CephPrimaryGroup
		host_uid := string(instance.UID)
		if utils.ContainsString(CephPrimaryGroup, host_uid) {
			CephPrimaryGroup = utils.RemoveString(CephPrimaryGroup, host_uid)
			logHost.Info("host is no longer present as a ceph primary group")
		}

		return nil
	}

	// Get a fresh snapshot of the current hosts.  These are used to search for
	// a matching host record if one is not already found as well as to
	// determine when it is safe/allowed to configure new hosts or unlock
	// existing hosts.
	r.hosts, err = hosts.ListHosts(client)
	if err != nil {
		err = perrors.Wrap(err, "failed to list hosts")
		return err
	}

	if host == nil {
		// This host either needs to be provisioned for the first time or we
		// need to audit the list of hosts so that we can find one that already
		// exists.
		host, err = r.ReconcileNewHost(client, instance, profile)
		if err != nil {
			return err
		}
	}

	// Check that the current configuration of a host matches the desired state.
	// This also captures errors from platform network subreconciler separately
	// thus enabling conditional handling of certain errors coming from
	// platform network subreconciler in future.
	err = r.ReconcileExistingHost(client, instance, profile, host, reqNs)

	inSync = err == nil
	oldInSync := instance.Status.InSync

	if r.statusUpdateRequired(instance, host, inSync) {
		logHost.V(2).Info("updating host status", "status", instance.Status)

		err2 := r.Client.Status().Update(context.TODO(), instance)
		if err2 != nil {
			err2 = perrors.Wrapf(err2, "failed to update status: %s",
				common.FormatStruct(instance.Status))
			return err2
		}

		if oldInSync != inSync {
			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "synchronization has changed to: %t", inSync)
		}
	}

	if err == nil {
		// We are done reconciling and will not be invoked again and so will
		// not be able to track the host state if it changes administrative,
		// operational or available states for the purpose of recording the
		// change in our database.  Therefore we are going to start a periodic
		// monitor to track the state of the host.
		msg := "monitoring host for state changes"
		m := NewStateChangeMonitor(instance, host.ID)
		return r.CloudManager.StartMonitor(m, msg)
	}

	return err
}

// Function to obtain ceph replication factor
func CephReplicationFactor(client *gophercloud.ServiceClient) (rep int, err error) {
	n := 0
	result, err := storagebackends.ListBackends(client)
	if err == nil {
		rep := result[0].Capabilities.Replication
		n, err = strconv.Atoi(rep)
		if err != nil {
			return n, err
		}
	} else {
		return n, err
	}

	logHost.V(2).Info("Ceph replication factor", "num", n)
	return n, nil
}

// Function to judge the specified host is in the ceph
// primary group or not. Add host uid to primary group
// list up to replication factor.
func IsCephPrimaryGroup(host_uid string, rep int) (pg bool, err error) {
	if len(host_uid) > 0 {
		for _, c := range CephPrimaryGroup {
			if c == host_uid {
				logHost.V(2).Info("Host already in CephPrimaryGroup", "id", host_uid)
				return true, nil
			}
		}
		if len(CephPrimaryGroup) < rep {
			CephPrimaryGroup = append(CephPrimaryGroup, host_uid)
			logHost.V(2).Info("Host added in CephPrimaryGroup", "id", host_uid)
			return true, nil
		}
	}
	return false, nil
}

// Check if the ceph primary group hosts are unlocked available.
func (r *HostReconciler) GetCephPrimaryGroupReady(client *gophercloud.ServiceClient) (ready bool, err error) {
	rep, err := CephReplicationFactor(client)
	if err != nil {
		return false, err
	}
	cephReady := false
	num := 0
	for _, host := range r.hosts {
		if host.Personality == hosts.PersonalityStorage && host.IsUnlockedAvailable() {
			num += 1
		}
	}
	if num >= rep {
		cephReady = true
	}
	return cephReady, nil
}

// Check the specified host is to be delay to be added.
// This will return true if the specified host is storage
// node and in the ceph non-primary group.
func (r *HostReconciler) IsCephDelayTargetGroup(client *gophercloud.ServiceClient, instance *starlingxv1.Host) (target bool, err error) {
	profile, err := r.BuildCompositeProfile(instance)
	if err != nil {
		return false, err
	}
	personality := profile.Personality
	if *personality != hosts.PersonalityStorage {
		return false, nil
	}
	if instance.Status.Reconciled {
		// Ignore reconciled strage node
		return false, nil
	}
	rep, err := CephReplicationFactor(client)
	if err != nil {
		return false, err
	}
	pg, err := IsCephPrimaryGroup(string(instance.UID), rep)
	if err != nil {
		return false, err
	} else {
		return !pg, nil
	}
}

// Update ReconcileAfterInSync in instance
// ReconcileAfterInSync value will be:
// "true"  if deploymentScope is "principal" because it is day 2 operation (update configuration)
// "false" if deploymentScope is "bootstrap"
// and
// Set ObservedGeneration as the value of Generation
// If generation is updated (= apply new configuration);
// - Set ConfigurationUpdated true
// - Set Reconciled false (since it is going to reconcile with new configuration)
// Then reflect these values to cluster object
// It is expected that instance.Status.Deployment scope is already updated by
// UpdateDeploymentScope at this point.
func (r *HostReconciler) UpdateConfigStatus(
	profile *starlingxv1.HostProfileSpec,
	instance *starlingxv1.Host,
	ns string,
) (err error) {

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := r.Client.Get(context.TODO(), types.NamespacedName{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		}, instance)
		if err != nil {
			return err
		}

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
		logHost.V(2).Info("update config after", "instance", instance)
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
			logHost.V(2).Info("set not required: set initial value")
		}

		// Check if the HostProfile configuration is updated
		hostProfile, err := r.GetHostProfile(instance.Namespace, instance.Spec.Profile)
		if err != nil {
			return err
		}
		if hostProfile != nil {
			if instance.Status.ObservedHostProfileGeneration != hostProfile.ObjectMeta.Generation {
				if (instance.Status.ObservedHostProfileGeneration == 0 && instance.Status.Reconciled) ||
					instance.Status.DeploymentScope != cloudManager.ScopePrincipal {
					// Case: DM upgrade in reconciled node or update in bootstrap
					instance.Status.HostProfileConfigurationUpdated = false
					logHost.V(2).Info("set profile config updated false: bootstrap or DM upgrade")
				} else {
					// Case: Fresh install or Day-2 operation
					instance.Status.HostProfileConfigurationUpdated = true
					instance.Status.Reconciled = false
					instance.Status.StrategyRequired = cloudManager.StrategyNotRequired
					logHost.V(2).Info("set profile config updated true: initial or day2 operation starts")
					logHost.V(2).Info("set not required: initial or day2 operation starts. profile config updated")
				}
				instance.Status.ObservedHostProfileGeneration = hostProfile.ObjectMeta.Generation
				logHost.V(2).Info("update observed profile config generation", "generation", hostProfile.ObjectMeta.Generation)
			}
		}

		if instance.Status.ObservedGeneration != instance.ObjectMeta.Generation {
			if instance.Status.ObservedGeneration == 0 &&
				instance.Status.Reconciled {
				// Case: DM upgrade in reconciled node
				instance.Status.ConfigurationUpdated = false
				logHost.V(2).Info("set host config updated false: bootstrap or DM upgrade")
			} else {
				// Case: Fresh install or Day-2 operation
				instance.Status.ConfigurationUpdated = true
				instance.Status.StrategyRequired = cloudManager.StrategyNotRequired
				logHost.V(2).Info("set host config updated true: initial or day2 operation starts")
				logHost.V(2).Info("set not required: initial or day2 operation starts. host config updated")
				if instance.Status.DeploymentScope == cloudManager.ScopePrincipal {
					instance.Status.Reconciled = false
					// Update strategy required status for strategy monitor
					r.CloudManager.UpdateConfigVersion()
					if hostProfile.Spec.Personality == nil &&
						profile.Personality != nil {
						hostProfile.Spec.Personality = profile.Personality
					}
					r.CloudManager.SetResourceInfo(cloudManager.ResourceHost, *hostProfile.Spec.Personality, instance.Name, instance.Status.Reconciled, cloudManager.StrategyNotRequired)
				}
			}
			instance.Status.ObservedGeneration = instance.ObjectMeta.Generation
		}

		logHost.V(2).Info("update status after", "instance", instance)
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
func (r *HostReconciler) UpdateStatusForFactoryInstall(
	ns string,
	instance *starlingxv1.Host,
) error {
	factory, err := r.CloudManager.GetFactoryInstall(ns)
	if err != nil {
		return err
	}
	if !factory {
		return nil
	}

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
	}
	r.ReconcilerEventLogger.NormalEvent(
		instance,
		common.ResourceUpdated,
		"Set Reconciled false for factory install",
	)
	return nil
}

// Reconcile reads that state of the cluster for a Host object and makes changes
// based on the state read and what is in the Host.Spec
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=hosts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=hosts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=hosts/finalizers,verbs=update
func (r *HostReconciler) Reconcile(ctx context.Context, request ctrl.Request) (result ctrl.Result, err error) {
	_ = log.FromContext(ctx)
	// FIXME: check log object
	// _ = r.Log.WithValues("host", request.NamespacedName)

	savedLog := logHost
	logHost = logHost.WithName(request.NamespacedName.String())
	defer func() { logHost = savedLog }()

	logHost.V(2).Info("reconcile called")

	// Fetch the Host instance
	instance := &starlingxv1.Host{}
	err = r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically
			// garbage collected. For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		logHost.Error(err, "unable to read object: %v", request)
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Cancel any existing monitors
	r.CloudManager.CancelMonitor(instance)

	if r.checkRestoreInProgress(instance) {
		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "Restoring '%s' host resource status without doing actual reconciliation", instance.Name)
		if err := r.RestoreHostStatus(instance); err != nil {
			return reconcile.Result{}, err
		}
		if err := r.ClearRestoreInProgress(instance); err != nil {
			return reconcile.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	err, platformnetwork_update_required := r.PlatformNetworkUpdateRequired(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	err, scope_updated := r.UpdateDeploymentScope(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.UpdateStatusForFactoryInstall(request.Namespace, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// TODO(wasnio): remove this once migration from helm chart to fluxcd is done
	// The status reaches its desired status post reconciled
	if instance.Status.ObservedGeneration == instance.ObjectMeta.Generation &&
		instance.Status.Reconciled &&
		instance.Status.DeploymentScope == "bootstrap" &&
		instance.Status.AvailabilityStatus != nil && *instance.Status.AvailabilityStatus == "available" &&
		instance.Status.StrategyRequired == cloudManager.StrategyNotRequired &&
		!platformnetwork_update_required {

		if !scope_updated {
			logHost.V(2).Info("reconcile finished, desired state reached after reconciled.")
			return reconcile.Result{}, nil
		}
	}

	if instance.DeletionTimestamp.IsZero() {
		// Ensure that the object has a finalizer setup as a pre-delete hook so
		// that we can delete any hosts that we have previously added.
		if !utils.ContainsString(instance.ObjectMeta.Finalizers, HostFinalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, HostFinalizerName)
			if err := r.Client.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}

			// Might as well return immediately as that update is going to cause
			// another reconcile event for this host and we don't want to hit
			// the system API more than necessary.
			return reconcile.Result{}, nil
		}
	}

	if !utils.IsReconcilerEnabled(utils.Host) {
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

	// Build a composite profile based on the profile chain and host overrides
	profile, err := r.BuildAndValidateCompositeProfile(instance)
	if err != nil {
		return r.ReconcilerErrorHandler.HandleReconcilerError(request, err)
	}

	// Update ReconciledAfterInSync and ObservedGeneration
	logHost.V(2).Info("before UpdateConfigStatus", "instance", instance)
	err = r.UpdateConfigStatus(profile, instance, request.Namespace)
	if err != nil {
		logHost.Error(err, "unable to update ReconciledAfterInSync or ObservedGeneration")
		return reconcile.Result{}, err
	}
	logHost.V(2).Info("after UpdateConfigStatus", "instance", instance)

	target, err := r.IsCephDelayTargetGroup(platformClient, instance)
	if err != nil {
		return reconcile.Result{}, err
	}
	if target {
		// If the node is storage but not in the ceph primary group,
		// it needs to wait until the ceph primary group are unlocked
		// and available.
		ready, err := r.GetCephPrimaryGroupReady(platformClient)
		if err != nil {
			return reconcile.Result{}, err
		}
		if !ready {
			logHost.Info("waiting for ceph primary group nodes unlocked-available")
			r.ReconcilerEventLogger.WarningEvent(instance, common.ResourceDependency,
				"waiting for ceph primary group nodes unlocked-available")
			return common.RetryCephPrimaryGroupNotReady, nil
		} else {
			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
				"ceph primary group is ready. Add host.")
		}
	} else {
		logHost.V(2).Info("not storage node or in ceph primary group. continue")
	}

	err = r.ReconcileResource(platformClient, instance, profile, request.Namespace)
	if err != nil {
		return r.ReconcilerErrorHandler.HandleReconcilerError(request, err)
	}

	return ctrl.Result{}, nil
}

// PlatformNetworkUpdateRequired checks and returns true if any of the
// platform networks / address pools are out of sync.
func (r *HostReconciler) PlatformNetworkUpdateRequired(instance *starlingxv1.Host) (error, bool) {
	errs := make([]error, 0)
	platform_network_instances, fetch_errs := r.ListPlatformNetworks(instance.Namespace)
	errs = append(errs, fetch_errs...)

	for _, platform_network_instance := range platform_network_instances {
		if !(platform_network_instance.Status.InSync && platform_network_instance.Status.Reconciled) {
			return nil, true
		}

		addrpool_instances, fetch_errs := r.GetAddressPoolsFromPlatformNetwork(platform_network_instance.Spec.AssociatedAddressPools,
			instance.Namespace)
		errs = append(errs, fetch_errs...)

		for _, addrpool_instance := range addrpool_instances {
			if !(addrpool_instance.Status.InSync && addrpool_instance.Status.Reconciled) {
				return nil, true
			}
		}
	}

	if len(errs) != 0 {
		err_msg := "There were errors fetching platform networks / addresspools"
		return common.NewPlatformNetworkReconciliationError(err_msg), false
	}

	return nil, false
}

// Verify whether we have annotation restore-in-progress
func (r *HostReconciler) checkRestoreInProgress(instance *starlingxv1.Host) bool {
	restoreInProgress, ok := instance.Annotations[cloudManager.RestoreInProgress]
	if ok && restoreInProgress != "" {
		return true
	}
	return false
}

// Update status
func (r *HostReconciler) RestoreHostStatus(instance *starlingxv1.Host) error {
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
					"Failed to update host status while restoring '%s' resource. Error: %s",
					instance.Name,
					err)
				return common.NewResourceStatusDependency(log_err_msg)
			} else {
				StatusUpdate := fmt.Sprintf("Status updated for host resource '%s' during restore with following values: Reconciled=%t InSync=%t DeploymentScope=%s",
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
func (r *HostReconciler) ClearRestoreInProgress(instance *starlingxv1.Host) error {
	delete(instance.Annotations, cloudManager.RestoreInProgress)
	if !utils.ContainsString(instance.ObjectMeta.Finalizers, HostFinalizerName) {
		instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, HostFinalizerName)
	}
	err := r.Client.Update(context.TODO(), instance)
	if err != nil {
		return common.NewResourceStatusDependency(fmt.Sprintf("Failed to update '%s' host resource after removing '%s' annotation during restoration.",
			instance.Name, cloudManager.RestoreInProgress))
	}
	return nil
}

// ListPlatformNetworks returns all of PlatformNetwork instances or errors while
// retrieving them if any.
func (r *HostReconciler) ListPlatformNetworks(namespace string) ([]*starlingxv1.PlatformNetwork, []error) {
	platform_network_instances := make([]*starlingxv1.PlatformNetwork, 0)
	errs := make([]error, 0)
	opts := client.ListOptions{}
	opts.Namespace = namespace
	platform_networks := &starlingxv1.PlatformNetworkList{}
	err := r.List(context.TODO(), platform_networks, &opts)
	if err != nil {
		err = perrors.Wrap(err, "failed to list platform networks")
		errs = append(errs, err)
	}

	for _, platform_network := range platform_networks.Items {
		platform_network_instance := &starlingxv1.PlatformNetwork{}
		platform_network_namespace := types.NamespacedName{Namespace: namespace, Name: platform_network.ObjectMeta.Name}
		err := r.Client.Get(context.TODO(), platform_network_namespace, platform_network_instance)
		if err != nil {
			logHost.Error(err, "Failed to get platform network resource from namespace")
			errs = append(errs, err)
			continue
		}

		platform_network_instances = append(platform_network_instances, platform_network_instance)
	}

	return platform_network_instances, errs
}

func (r *HostReconciler) GetAddressPoolsFromPlatformNetwork(associated_addrpools []string, namespace string) ([]*starlingxv1.AddressPool, []error) {
	addrpool_instances := make([]*starlingxv1.AddressPool, 0)
	errs := make([]error, 0)
	for _, addrpool_name := range associated_addrpools {
		addrpool_instance := &starlingxv1.AddressPool{}
		addrpool_namespace := types.NamespacedName{
			Namespace: namespace,
			Name:      addrpool_name}
		err := r.Client.Get(context.TODO(), addrpool_namespace, addrpool_instance)
		if err != nil {
			logHost.Error(err, "Failed to get addrpool resource from namespace")
			errs = append(errs, err)
			continue
		}
		addrpool_instances = append(addrpool_instances, addrpool_instance)
	}

	return addrpool_instances, errs
}

// LockHostRequestByOtherController takes the host object, host ID / personality and
// starts lock host monitor and returns LockedDisabledHost monitor error.
// This would initiate lock request of a host through VIM so that disabled attributes
// can be reconciled later.
func (r *HostReconciler) LockHostRequestByOtherController(host_instance *starlingxv1.Host, host_id string, host_personality string, set_res_info bool) error {

	if host_instance.Status.StrategyRequired != cloudManager.StrategyLockRequired {
		if !r.CloudManager.GetStrategyExpectedByOtherReconcilers() {
			r.CloudManager.SetStrategyExpectedByOtherReconcilers(true)
			logHost.Info("StrategyExpectedByOtherReoncilers has been set to true.")
		}

		logHost.Info(fmt.Sprintf("Updating strategyRequired to lock_required for %s.", host_instance.Name))
		host_instance.Status.StrategyRequired = cloudManager.StrategyLockRequired
		if set_res_info {
			r.CloudManager.SetResourceInfo(cloudManager.ResourceHost,
				host_personality,
				host_instance.Name,
				host_instance.Status.Reconciled,
				host_instance.Status.StrategyRequired)
		}

		err := r.Client.Status().Update(context.TODO(), host_instance)
		if err != nil {
			logHost.Error(err, "failed to update host strategy")
			return err
		}
	}

	return nil
}

func (r *HostReconciler) StartLockedDisabledHostMonitor(host_instance *starlingxv1.Host, host_id, msg string) error {
	m := NewLockedDisabledHostMonitor(host_instance, host_id)
	return r.CloudManager.StartMonitor(m, msg)
}

// UpdateDeploymentScope function is used to update the deployment scope for Host.
func (r *HostReconciler) UpdateDeploymentScope(instance *starlingxv1.Host) (error, bool) {
	updated, err := common.UpdateDeploymentScope(r.Client, instance)
	if err != nil {
		logHost.Error(err, "failed to update deploymentScope")
		return err, false
	}
	return nil, updated
}

// SetupWithManager sets up the controller with the Manager.
func (r *HostReconciler) SetupWithManager(mgr ctrl.Manager) error {
	tMgr := cloudManager.GetInstance(mgr)
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.CloudManager = tMgr
	r.ReconcilerErrorHandler = &common.ErrorHandler{
		CloudManager: tMgr,
		Logger:       logHost}
	r.ReconcilerEventLogger = &common.EventLogger{
		EventRecorder: mgr.GetEventRecorderFor(HostControllerName),
		Logger:        logHost}
	return ctrl.NewControllerManagedBy(mgr).
		For(&starlingxv1.Host{}).
		Complete(r)
}
