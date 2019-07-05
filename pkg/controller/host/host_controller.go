/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package host

import (
	"context"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/labels"
	perrors "github.com/pkg/errors"
	starlingxv1beta1 "github.com/wind-river/titanium-deployment-manager/pkg/apis/starlingx/v1beta1"
	utils "github.com/wind-river/titanium-deployment-manager/pkg/common"
	"github.com/wind-river/titanium-deployment-manager/pkg/config"
	"github.com/wind-river/titanium-deployment-manager/pkg/controller/common"
	titaniumManager "github.com/wind-river/titanium-deployment-manager/pkg/manager"
	v1info "github.com/wind-river/titanium-deployment-manager/pkg/platform"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strconv"
	"strings"
)

var log = logf.Log.WithName("controller").WithName("host")

const ControllerName = "host-controller"

const FinalizerName = "host.finalizers.windriver.com"

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
var DynamicProvisioningMode = starlingxv1beta1.ProvioningModeDynamic
var DefaultHostProfile = starlingxv1beta1.HostProfileSpec{
	ProfileBaseAttributes: starlingxv1beta1.ProfileBaseAttributes{
		AdministrativeState: &AdminLocked,
		ProvisioningMode:    &DynamicProvisioningMode,
	},
}

// Add creates a new Host Controller and adds it to the Manager with default
// RBAC. The Manager will set fields on the Controller and Start it when the
// Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	tMgr := titaniumManager.GetInstance(mgr)
	return &ReconcileHost{
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

	// Watch for changes to Host
	err = c.Watch(&source.Kind{Type: &starlingxv1beta1.Host{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileHost{}

// ReconcileHostByState reconciles a Host object
type ReconcileHost struct {
	client.Client
	scheme *runtime.Scheme
	titaniumManager.TitaniumManager
	common.ReconcilerErrorHandler
	common.ReconcilerEventLogger
	hosts []hosts.Host
}

// hostMatchesCriteria evaluates whether a host matches the criteria specified
// by the operator.  All match attributes must match for a host to match a
// profile.
func hostMatchesCriteria(h hosts.Host, criteria *starlingxv1beta1.MatchInfo) bool {
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

// getBMPasswordCredentials is a utility to retrieve the host's board management
// credentials from the information stored in the specified secret.
func (r *ReconcileHost) getBMPasswordCredentials(namespace string, name string) (username, password string, err error) {
	secret := &v1.Secret{}
	secretName := types.NamespacedName{Namespace: namespace, Name: name}

	// Lookup the secret via the system client.
	err = r.Client.Get(context.TODO(), secretName, secret)
	if err != nil {
		if errors.IsNotFound(err) == false {
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
func (r *ReconcileHost) buildInitialHostOpts(instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec) (hosts.HostOpts, error) {
	dummy := hosts.Host{}
	result, _, err := r.UpdateRequired(instance, profile, &dummy)
	return result, err
}

func provisioningAllowed(objects []hosts.Host) bool {
	for _, host := range objects {
		if host.Hostname == hosts.Controller0 {
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
func (r *ReconcileHost) ProvisioningAllowed() bool {
	return provisioningAllowed(r.hosts)
}

func monitorsEnabled(objects []hosts.Host, required int) bool {
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
func (r *ReconcileHost) MonitorsEnabled(required int) bool {
	return monitorsEnabled(r.hosts, required)
}

func allControllerNodesEnabled(objects []hosts.Host, required int) bool {
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
func (r *ReconcileHost) AllControllerNodesEnabled(required int) bool {
	return allControllerNodesEnabled(r.hosts, required)
}

// UpdateRequired determines if any of the configured attributes mismatch with
// those in the running system.  If there are mismatches then true is returned
// in the result and opts is configured with only those values that
// need to change.
func (r *ReconcileHost) UpdateRequired(instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, h *hosts.Host) (opts hosts.HostOpts, result bool, err error) {

	if instance.Name != h.Hostname {
		result = true
		opts.Hostname = &instance.Name
	}

	if profile.Personality != nil && *profile.Personality != h.Personality {
		result = true
		opts.Personality = profile.Personality
	}

	if profile.SubFunctions != nil {
		subfunctions := strings.Split(h.SubFunctions, ",")
		if utils.ListChanged(profile.SubFunctions, subfunctions) {
			result = true
			subfunctions := strings.Join(profile.SubFunctions, ",")
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

		if bm.Credentials.Password != nil {
			// Password based authentication therefore retrieve the information
			// from the provided secret.
			info := bm.Credentials.Password
			username, password, err := r.getBMPasswordCredentials(instance.Namespace, info.Secret)
			if err != nil {
				if errors.IsNotFound(err) == true {
					msg := fmt.Sprintf("waiting for BM credentials secret: %q", info.Secret)
					name := types.NamespacedName{Namespace: instance.Namespace, Name: info.Secret}
					m := NewKubernetesSecretMonitor(instance, name)
					return hosts.HostOpts{}, result, r.StartMonitor(m, msg)
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

	return opts, result, nil
}

// HTTPSRequired determines whether an HTTPS connection is required for the
// purpose of configuring host BMC attributes.
func (r *ReconcileHost) HTTPSRequired() bool {
	value := config.GetReconcilerOption(config.BMC, config.HTTPSRequired)
	if value != nil {
		if required, ok := value.(bool); ok {
			return required
		} else {
			log.Info("unexpected option type",
				"option", config.HTTPSRequired, "type", reflect.TypeOf(value))
		}
	}

	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return true
}

// ReconcileAttributes is responsible for reconciling the basic attributes for a
// host resource.
func (r *ReconcileHost) ReconcileAttributes(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *hosts.Host) error {
	if opts, ok, err := r.UpdateRequired(instance, profile, host); ok && err == nil {

		if opts.BMPassword != nil && strings.HasPrefix(client.Endpoint, titaniumManager.HTTPPrefix) {
			if r.HTTPSRequired() {
				// Do not send password information in the clear.
				msg := fmt.Sprintf("it is unsafe to configure BM credentials thru a non HTTPS URL")
				return common.NewSystemDependency(msg)
			} else {
				log.Info("allowing BMC configuration over HTTP connection")
			}
		}

		log.Info("updating host attributes", "opts", opts)

		result, err := hosts.Update(client, host.ID, opts).Extract()
		if err != nil || result == nil {
			err = perrors.Wrapf(err, "failed to update host attributes: %s, %s",
				host.ID, common.FormatStruct(opts))
			return err
		}

		*host = *result

		r.NormalEvent(instance, common.ResourceUpdated,
			"attributes have been updated")

	} else if err != nil {
		return err
	}

	return nil
}

// ReconcileAttributes is responsible for reconciling the labels on each host.
func (r *ReconcileHost) ReconcileLabels(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
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
			log.Info("removing label", "label", label)

			err := labels.Delete(client, label.ID).ExtractErr()
			if err != nil {
				err = perrors.Wrapf(err, "failed to remove label %s", label.ID)
				return err
			}

			r.NormalEvent(instance, common.ResourceUpdated,
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
		log.Info("adding labels", "labels", request)

		_, err := labels.Create(client, host.ID, request).Extract()
		if err != nil {
			err = perrors.Wrapf(err, "failed to create labels")
			return err
		}

		keys := make([]string, 0, len(request))
		for k := range request {
			keys = append(keys, k)
		}

		r.NormalEvent(instance, common.ResourceUpdated,
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
func (r *ReconcileHost) ReconcilePowerState(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	var action string

	if profile.PowerOn == nil {
		return nil
	}

	// NOTE: The "task" is not considered here because we only reconcile hosts
	// that are not currently executing a task

	if *profile.PowerOn == true {
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
		r.WarningEvent(instance, common.ResourceDependency, msg)
		return common.NewResourceConfigurationDependency(msg)
	}

	opts := hosts.HostOpts{
		Action: &action,
	}

	log.Info("sending action to host", "opts", opts)

	result, err := hosts.Update(client, host.ID, opts).Extract()
	if err != nil || result == nil {
		err = perrors.Wrapf(err, "failed to set power state for host: %s, %s",
			host.ID, common.FormatStruct(opts))
		return err
	}

	host.Host = *result

	r.NormalEvent(instance, common.ResourceUpdated,
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
func (r *ReconcileHost) ReconcileInitialState(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	desiredState := profile.AdministrativeState
	if desiredState != nil && *desiredState != host.AdministrativeState {
		if *desiredState == hosts.AdminLocked {
			action := hosts.ActionLock
			opts := hosts.HostOpts{
				Action: &action,
			}

			log.Info("locking host", "opts", opts)

			result, err := hosts.Update(client, host.ID, opts).Extract()
			if err != nil || result == nil {
				err = perrors.Wrapf(err, "failed to lock host: %s, %s",
					host.ID, common.FormatStruct(opts))
				return err
			}

			host.Host = *result

			r.NormalEvent(instance, common.ResourceUpdated,
				"host has been locked")

			// Return a retry result here because we know that it won't be possible to
			// make any other changes until this change is complete.
			return common.NewResourceStatusDependency("waiting for host state change")
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
func (r *ReconcileHost) ReconcileFinalState(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	state := profile.AdministrativeState
	if state == nil || *state == host.AdministrativeState {
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
			return r.StartMonitor(m, msg)
		}
	}

	action := hosts.ActionUnlock
	opts := hosts.HostOpts{
		Action: &action,
	}

	log.Info("unlocking host", "opts", opts)

	result, err := hosts.Update(client, host.ID, opts).Extract()
	if err != nil || result == nil {
		err = perrors.Wrapf(err, "failed to unlock host: %s, %s",
			host.ID, common.FormatStruct(opts))
		return err
	}

	host.Host = *result

	r.NormalEvent(instance, common.ResourceUpdated,
		"host has been unlocked")

	// Return a retry result here because we know that it won't be possible to
	// make any other changes until this change is complete.
	return common.NewResourceStatusDependency("waiting for host state change")
}

func (r *ReconcileHost) ReconcileEnabledHost(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	err := r.ReconcileInitialState(client, instance, profile, host)
	if err != nil {
		return err
	}

	// The state may have changed in the last step so double check and wait if
	// necessary.
	if host.IsUnlockedEnabled() == false {
		msg := "enabled host changed state during reconciliation"
		m := NewIdleHostMonitor(instance, host.ID)
		return r.StartMonitor(m, msg)
	}

	switch r.OSDProvisioningState(instance.Namespace, host.Personality) {
	case RequiredStateEnabled, RequiredStateAny:
		err = r.ReconcileOSDs(client, instance, profile, host)
		if err != nil {
			return err
		}
	}

	return nil
}

// ReconcileHostByState is responsible for reconciling each individual sub-domain of a
// host resource.
func (r *ReconcileHost) ReconcileDisabledHost(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {

	err := r.ReconcileAttributes(client, instance, profile, &host.Host)
	if err != nil {
		return err
	}

	err = r.ReconcileLabels(client, instance, profile, host)
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

		err = r.ReconcileMemory(client, instance, profile, host)
		if err != nil {
			return err
		}
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

// CompareOSDs determine if there has been a change to the list of OSDs between
// two profile specs. This method takes into consideration that the storage
// section may be completely empty on either side of the comparison.
func (r *ReconcileHost) CompareOSDs(in *starlingxv1beta1.HostProfileSpec, other *starlingxv1beta1.HostProfileSpec) bool {
	if other == nil {
		return false
	}

	if (in.Storage == nil) && (other.Storage == nil) {
		return true

	} else if in.Storage != nil {
		if in.Storage.DeepEqual(other.Storage) {
			// The full storage profile matches there the OSDs match.
			return true
		}

		// Other just check the OSD list and ignore the other attributes.
		if in.Storage.OSDs.DeepEqual(&other.Storage.OSDs) == false {
			return false
		}

	} else if len(other.Storage.OSDs) > 0 {
		return false
	}

	return true
}

// CompareAttributes determines if two profiles are identical for the
// purpose of reconciling a current host configuration to its desired host
// profile.
func (r *ReconcileHost) CompareAttributes(in *starlingxv1beta1.HostProfileSpec, other *starlingxv1beta1.HostProfileSpec, namespace, personality string) bool {
	// This could be replaced with in.DeepEqual(other) but it is coded this way
	// (and tested this way) to ensure that if both the "enabled" and "disabled"
	// comparisons are true then no reconciliation is missed.  The intent is
	// that CompareEnabledAttributes && CompareDisabledAttributes will always
	// be equivalent to DeepEqual.
	return r.CompareEnabledAttributes(in, other, namespace, personality) &&
		r.CompareDisabledAttributes(in, other, namespace, personality)
}

// CompareEnabledAttributes determines if two profiles are identical for the
// purpose of reconciling any attributes that can only be applied when the host
// is enabled.  The only attributes that we can reconcile while enabled are the
// storage OSD resources therefore return false if there are any differences
// in the storage OSD list.
func (r *ReconcileHost) CompareEnabledAttributes(in *starlingxv1beta1.HostProfileSpec, other *starlingxv1beta1.HostProfileSpec, namespace, personality string) bool {
	if other == nil {
		return false
	}

	if in.AdministrativeState != nil {
		if (in.AdministrativeState == nil) != (other.AdministrativeState == nil) {
			return false
		} else if in.AdministrativeState != nil {
			if *in.AdministrativeState != *other.AdministrativeState {
				return false
			}
		}
	}

	if config.IsReconcilerEnabled(config.OSD) {
		switch r.OSDProvisioningState(namespace, personality) {
		case RequiredStateEnabled, RequiredStateAny:
			if r.CompareOSDs(in, other) == false {
				return false
			}
		}
	}

	return true
}

// CompareEnabledAttributes determines if two profiles are identical for the
// purpose of reconciling any attributes that can only be applied when the host
// is enabled.
func (r *ReconcileHost) CompareDisabledAttributes(in *starlingxv1beta1.HostProfileSpec, other *starlingxv1beta1.HostProfileSpec, namespace, personality string) bool {
	if other == nil {
		return false
	}

	if in.ProfileBaseAttributes.DeepEqual(&other.ProfileBaseAttributes) == false {
		return false
	}

	if (in.BoardManagement == nil) != (other.BoardManagement == nil) {
		return false
	} else if in.BoardManagement != nil {
		if in.BoardManagement.DeepEqual(other.BoardManagement) == false {
			return false
		}
	}

	if config.IsReconcilerEnabled(config.Memory) {
		if in.Memory.DeepEqual(&other.Memory) == false {
			return false
		}
	}

	if config.IsReconcilerEnabled(config.Processor) {
		if in.Processors.DeepEqual(&other.Processors) == false {
			return false
		}
	}

	if config.IsReconcilerEnabled(config.Networking) {
		if config.IsReconcilerEnabled(config.Interface) {
			if (in.Interfaces == nil) != (other.Interfaces == nil) {
				return false
			} else if in.Interfaces != nil {
				if in.Interfaces.DeepEqual(other.Interfaces) == false {
					return false
				}
			} else {
				return false
			}
		}

		if config.IsReconcilerEnabled(config.Address) {
			if in.Addresses.DeepEqual(&other.Addresses) == false {
				return false
			}
		}

		if config.IsReconcilerEnabled(config.Route) {
			if in.Routes.DeepEqual(&other.Routes) == false {
				return false
			}
		}
	}

	if config.IsReconcilerEnabled(config.OSD) {
		switch r.OSDProvisioningState(namespace, personality) {
		case RequiredStateDisabled, RequiredStateAny:
			if r.CompareOSDs(in, other) == false {
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
func (r *ReconcileHost) ReconcileHostByState(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, current *starlingxv1beta1.HostProfileSpec, profile *starlingxv1beta1.HostProfileSpec, host *v1info.HostInfo) error {
	if host.IsUnlockedEnabled() {
		if r.CompareEnabledAttributes(profile, current, instance.Namespace, host.Personality) == false {
			err := r.ReconcileEnabledHost(client, instance, profile, host)
			if err != nil {
				return err
			}
		} else {
			log.Info("no enabled attribute changes required")
		}

		if r.CompareDisabledAttributes(profile, current, instance.Namespace, host.Personality) == false {
			msg := "waiting for locked state before applying out-of-service attributes"
			m := NewLockedDisabledHostMonitor(instance, host.ID)
			return r.StartMonitor(m, msg)
		}

	} else if host.IsLockedDisabled() {
		if r.CompareDisabledAttributes(profile, current, instance.Namespace, host.Personality) == false {
			err := r.ReconcileDisabledHost(client, instance, profile, host)
			if err != nil {
				return err
			}
		} else {
			log.Info("no disabled attribute changes required")
		}

		if r.CompareEnabledAttributes(profile, current, instance.Namespace, host.Personality) == false {
			msg := "waiting for the unlocked state before applying in-service attributes"
			m := NewUnlockedEnabledHostMonitor(instance, host.ID)
			return r.StartMonitor(m, msg)
		}

	} else {
		msg := fmt.Sprintf("waiting for a stable state")
		m := NewIdleHostMonitor(instance, host.ID)
		return r.StartMonitor(m, msg)
	}

	return nil
}

// statusUpdateRequired is a utility function which determines whether an update
// is required to the host status attribute.  Updating this unnecessarily
// will result in an infinite reconciliation loop.
func (r *ReconcileHost) statusUpdateRequired(instance *starlingxv1beta1.Host, host *hosts.Host, inSync bool) (result bool) {
	status := &instance.Status

	if status.ID == nil || *status.ID != host.ID {
		status.ID = &host.ID
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

	if status.InSync == true && status.Reconciled == false {
		// Record the fact that we have reached inSync at least once.
		status.Reconciled = true
		result = true
	}

	return result
}

// findExistingHost searches the current list of hosts and attempts to find one
// that fits the provided match criteria.
func findExistingHost(objects []hosts.Host, hostname string, match *starlingxv1beta1.MatchInfo, bootMAC *string) *hosts.Host {
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
func (r *ReconcileHost) ReconcileNewHost(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *hosts.Host) (*hosts.Host, error) {
	host = findExistingHost(r.hosts, instance.Name, instance.Spec.Match, profile.BootMAC)
	if host != nil {
		log.Info("found matching host", "id", host.ID)
	}

	if host == nil {
		// A new host needs to be provisioned or we need to wait for one to
		// appear in the system.
		if *profile.ProvisioningMode != starlingxv1beta1.ProvioningModeStatic {
			// We only create missing hosts for statically provisioned hosts.
			// For dynamic, hosts we wait for them to appear in the system
			msg := "waiting for dynamic host to appear in inventory"
			m := NewDynamicHostMonitor(instance, instance.Name, instance.Spec.Match, profile.BootMAC)
			return nil, r.StartMonitor(m, msg)

		} else if r.ProvisioningAllowed() {
			// Populate a new host into system inventory.
			opts, err := r.buildInitialHostOpts(instance, profile)
			if err != nil {
				return nil, err // Already logged
			}

			log.Info("creating host", "opts", opts)

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

			r.NormalEvent(instance, common.ResourceCreated,
				"static host has been created")

		} else {
			msg := "waiting for system to allow creating static hosts"
			m := NewProvisioningAllowedMonitor(instance)
			return nil, r.StartMonitor(m, msg)
		}

	} else if host.Hostname == "" {
		// The host was found but it has not been provisioned with a hostname
		// and personality so set up its initial attributes.
		if r.ProvisioningAllowed() {
			log.Info("setting initial attributes")
			err := r.ReconcileAttributes(client, instance, profile, host)
			if err != nil {
				return host, err
			}

		} else {
			msg := "waiting for system to allow host provisioning"
			m := NewProvisioningAllowedMonitor(instance)
			return host, r.StartMonitor(m, msg)
		}
	}

	return host, nil
}

// StopAfterInSync determines whether the reconciler should continue processing
// change requests after the
// purpose of configuring host BMC attributes.
func (r *ReconcileHost) StopAfterInSync() bool {
	value := config.GetReconcilerOption(config.Host, config.StopAfterInSync)
	if value != nil {
		if required, ok := value.(bool); ok {
			return required
		} else {
			log.Info("unexpected option type",
				"option", config.StopAfterInSync, "type", reflect.TypeOf(value))
		}
	}

	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return true
}

// ReconcileExistingHost is responsible for dealing with the provisioning of an
// existing host.
func (r *ReconcileHost) ReconcileExistingHost(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec, host *hosts.Host) error {
	var defaults *starlingxv1beta1.HostProfileSpec
	var current *starlingxv1beta1.HostProfileSpec

	if !host.Idle() {
		msg := fmt.Sprintf("waiting for a stable state")
		m := NewIdleHostMonitor(instance, host.ID)
		return r.StartMonitor(m, msg)
	}

	// Gather all host attributes so that they can be reused by various
	// functions without needing to be re-queried each time.
	hostInfo := v1info.HostInfo{}
	err := hostInfo.PopulateHostInfo(client, host.ID)
	if err != nil {
		return err
	}

	// Fetch default attributes so that they can be used to back sparse host
	// profile configurations.
	defaults, err = r.GetHostDefaults(instance)
	if err != nil {
		return err
	} else if defaults == nil {
		if host.Idle() == false || host.AvailabilityStatus == hosts.AvailOffline {
			// Ideally we would only ever collect the defaults when the host is
			// in the locked/disabled/online state.  This is the best approach
			// when provisioning a system from scratch, but for cases where
			// an operator may want to start with a partially configured system
			// then using any stable state is sufficient.
			msg := "waiting for a stable state before collecting defaults"
			m := NewInventoryCollectedMonitor(instance, host.ID)
			return r.StartMonitor(m, msg)
		}

		if len(hostInfo.Disks) == 0 {
			// There is no way to tell if the inventory process has finished so
			// we rely on checking the number of disks since that's one of the
			// last things to be collected.  If that list is 0 then we need to
			// wait some more.
			msg := "waiting for inventory collection to complete before collecting defaults"
			m := NewInventoryCollectedMonitor(instance, host.ID)
			return r.StartMonitor(m, msg)
		}

		log.Info("collecting default values")

		defaults, err = r.BuildHostDefaults(instance, hostInfo)
		if err != nil {
			return err
		}

		r.NormalEvent(instance, common.ResourceCreated,
			"defaults collected and stored")

		current = defaults.DeepCopy()

	} else {
		// Otherwise, the defaults already existed so build a new profile with
		// the current host configuration so that we can compare it to the
		// desired state.
		log.V(2).Info("building current profile from current config")

		current, err = starlingxv1beta1.NewHostProfileSpec(hostInfo)
		if err != nil {
			return err
		}
	}

	// NOTE(alegacy): The defaults collected may include BMC information and
	// since the API does not return any password info it is not possible to
	// build a true representation of the system state.  Trying to reconcile
	// the system generated BMC info will always result in an error because
	// it does not contain password info.
	defaults.BoardManagement = nil

	// Create a new composite profile that is backed by the host's default
	// configuration.  This will ensure that if a user deletes an optional
	// attribute that we will know how to restore the original value.
	profile, err = MergeProfiles(defaults, profile)
	if err != nil {
		return err
	}

	// TODO(alegacy): Need to move ProvisioningMode out of the profile or
	//  find a way to populate it into profiles generated from the running
	//  configuration.
	profile.ProvisioningMode = nil

	inSync := r.CompareAttributes(profile, current, instance.Namespace, host.Personality)
	if inSync {
		log.V(2).Info("no changes between composite profile and current configuration")
		return nil
	}

	log.V(2).Info("defaults are:", "values", defaults)

	log.V(2).Info("final profile is:", "values", profile)

	log.V(2).Info("current config is:", "values", current)

	if instance.Status.Reconciled && r.StopAfterInSync() {
		// Do not process any further changes once we have reached a
		// synchronized state unless there is an annotation on the host.
		if _, present := instance.Annotations[titaniumManager.ReconcileAfterInSync]; !present {
			msg := "configuration changes ignored after initial synchronization has completed"
			r.NormalEvent(instance, common.ResourceUpdated, msg)
			return common.NewChangeAfterInSync(msg)
		} else {
			log.Info("Manual override; allowing configuration changes after initial synchronization.")
		}
	}

	err = r.ReconcileHostByState(client, instance, current, profile, &hostInfo)
	if err != nil {
		return err
	}

	log.V(2).Info("final configuration is:", "profile", current)

	return nil
}

// ReconcileExistingHost is responsible for dealing with the provisioning of an
// existing host.
func (r *ReconcileHost) ReconcileDeletedHost(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host, host *hosts.Host) (err error) {
	if host.Capabilities.Personality != nil {
		if strings.EqualFold(*host.Capabilities.Personality, hosts.ActiveController) {
			// Always leave the active controller installed.
			log.Info("skipping delete for active controller")
			return nil
		}
	}

	if host.Idle() && host.IsLockedDisabled() == false {
		action := hosts.ActionLock
		opts := hosts.HostOpts{Action: &action}

		log.Info("locking host", "opts", opts)

		result, err := hosts.Update(client, host.ID, opts).Extract()
		if err != nil {
			err = perrors.Wrap(err, "failed to lock host")
			return err
		}
		*host = *result

		r.NormalEvent(instance, common.ResourceUpdated, "host has been locked")
	}

	if host.IsLockedDisabled() == false {
		// Host is still not locked so wait for the action to complete.
		msg := "waiting for host to lock before deleting it"
		m := NewLockedDisabledHostMonitor(instance, host.ID)
		return r.StartMonitor(m, msg)
	}

	log.Info("deleting host")

	err = hosts.Delete(client, host.ID).ExtractErr()
	if err != nil {
		err = perrors.Wrapf(err, "failed to delete host: %s", host.ID)
		return err
	}

	r.NormalEvent(instance, common.ResourceDeleted, "host has been deleted")

	return nil
}

// ReconcileResource interacts with the system API in order to reconcile the
// state of a data network with the state stored in the k8s database.
func (r *ReconcileHost) ReconcileResource(client *gophercloud.ServiceClient, instance *starlingxv1beta1.Host) (err error) {
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
				err = perrors.Wrapf(err, "failed to get: %s", *id)
				return err
			}

			// The resource may have been deleted by the system or operator
			// therefore continue and attempt to recreate it.
			log.Info("resource no longer exists", "id", *id)
		}
	}

	if instance.DeletionTimestamp.IsZero() == false {
		if utils.ContainsString(instance.ObjectMeta.Finalizers, FinalizerName) {
			// A finalizer is still present so we need to try to delete the
			// host from the system.
			if host != nil {
				err = r.ReconcileDeletedHost(client, instance, host)
				if err != nil {
					return err
				}

			} else {
				log.Info("host being deleted is no longer present on system")
			}

			// Remove the finalizer so we don't try to do this delete action again.
			instance.ObjectMeta.Finalizers = utils.RemoveString(instance.ObjectMeta.Finalizers, FinalizerName)
			if err := r.Update(context.Background(), instance); err != nil {
				return err
			}
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

	// Build a composite profile based on the profile chain and host overrides
	profile, err := r.BuildCompositeProfile(instance)
	if err != nil {
		return err
	}

	log.V(2).Info("composite profile is:", "name", instance.Spec.Profile, "profile", profile)

	err = r.ValidateProfile(instance, profile)
	if err != nil {
		return err
	}

	if host == nil {
		// This host either needs to be provisioned for the first time or we
		// need to audit the list of hosts so that we can find one that already
		// exists.
		host, err = r.ReconcileNewHost(client, instance, profile, host)
		if err != nil {
			return err
		}
	}

	// Check that the current configuration of a host matches the desired state.
	err = r.ReconcileExistingHost(client, instance, profile, host)

	inSync = err == nil
	oldInSync := instance.Status.InSync

	if r.statusUpdateRequired(instance, host, inSync) {
		log.V(2).Info("updating host status", "status", instance.Status)

		err2 := r.Status().Update(context.TODO(), instance)
		if err2 != nil {
			err2 = perrors.Wrapf(err2, "failed to update status: %s",
				common.FormatStruct(instance.Status))
			return err2
		}

		if oldInSync != inSync {
			r.NormalEvent(instance, common.ResourceUpdated, "synchronization has changed to: %t", inSync)
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
		return r.StartMonitor(m, msg)
	}

	return err
}

// Reconcile reads that state of the cluster for a Host object and makes changes
// based on the state read and what is in the Host.Spec
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=hosts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=hosts/status,verbs=get;update;patch
func (r *ReconcileHost) Reconcile(request reconcile.Request) (result reconcile.Result, err error) {
	savedLog := log
	log = log.WithName(request.NamespacedName.String())
	defer func() { log = savedLog }()

	log.V(2).Info("reconcile called")

	// Fetch the Host instance
	instance := &starlingxv1beta1.Host{}
	err = r.Get(context.TODO(), request.NamespacedName, instance)
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

	// Cancel any existing monitors
	r.CancelMonitor(instance)

	if instance.DeletionTimestamp.IsZero() {
		// Ensure that the object has a finalizer setup as a pre-delete hook so
		// that we can delete any hosts that we have previously added.
		if !utils.ContainsString(instance.ObjectMeta.Finalizers, FinalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, FinalizerName)
			if err := r.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}

			// Might as well return immediately as that update is going to cause
			// another reconcile event for this host and we don't want to hit
			// the system API more than necessary.
			return reconcile.Result{}, nil
		}
	}

	if config.IsReconcilerEnabled(config.Host) == false {
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
