/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2023 Wind River Systems, Inc. */

package system

import (
	"context"
	"crypto/md5"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/certificates"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/controllerFilesystems"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/dns"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/drbd"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/licenses"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ntp"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptp"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/serviceparameters"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/storagebackends"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/system"
	"github.com/imdario/mergo"
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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var logSystem = log.Log.WithName("controller").WithName("system")

const SystemControllerName = "system-controller"

var _ reconcile.Reconciler = &SystemReconciler{}

// SystemReconciler reconciles a System object
type SystemReconciler struct {
	manager.Manager
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	cloudManager.CloudManager
	common.ReconcilerErrorHandler
	common.ReconcilerEventLogger
	hosts []hosts.Host
}

const CertificateDirectory = "/etc/ssl/certs"

func InstallCertificate(filename string, data []byte) error {
	err := os.MkdirAll("/etc/ssl/certs", 0600)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s/%s", CertificateDirectory, filename)
	err = os.WriteFile(path, data, 0600)
	if err != nil {
		return err
	}

	return nil
}

// installRootCertificates examines the list of certificates to be configured
// against the system and for any platform certificates found it will install
// the corresponding CA certificate into the system certificate path.  Since
// all controllers need to spawn clients to communicate with the system API
// this approach was used rather than to pass the certificates directly to the
// gophercloud API since that would require several additional steps by each
// controller to load the certificates from the system secret.
func (r *SystemReconciler) installRootCertificates(instance *starlingxv1.System) error {
	if instance.Spec.Certificates == nil {
		return nil
	}

	for _, c := range *instance.Spec.Certificates {
		if c.Type != starlingxv1.PlatformCertificate {
			// We only interact with the platform API therefore we do not need
			// to install any other CA certificate locally.
			continue
		}

		secret := v1.Secret{}
		secretName := types.NamespacedName{Namespace: instance.Namespace, Name: c.Secret}
		err := r.Client.Get(context.TODO(), secretName, &secret)
		if err != nil {
			return err
		}

		var caBytes []byte
		var ok bool
		numRetries := 30
		for iter := 0; iter < numRetries; iter++ {
			caBytes, ok = secret.Data[starlingxv1.SecretCaCertKey]
			if !ok {
				logSystem.Info("Platform certificate CA not ready/available", "name", c.Secret)
				time.Sleep(5 * time.Second)
			} else {
				logSystem.Info("Platform certificate CA ready!")
				break
			}
		}
		if !ok {
			logSystem.Info("Continuing deployment without a CA certificate", "name", c.Secret)
			continue
		}

		filename := fmt.Sprintf("%s-%s-ca-cert.pem", instance.Namespace, c.Secret)
		err = InstallCertificate(filename, caBytes)
		if err != nil {
			logSystem.Error(err, "failed to install root certificate")
			return err
		}

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceCreated,
			"root certificate saved as file: %s", filename)
	}

	return nil
}

// NoContent represents an empty command separated list to certain endpoints
// of the system API (i.e., DNS and NTP)
const NoContent = "NC"

// ntpUpdateRequired determines whether an update is required to the NTP
// system attributes and returns the attributes to be changed if an update
// is necessary.
func ntpUpdateRequired(spec *starlingxv1.SystemSpec, info *ntp.NTP) (ntpOpts ntp.NTPOpts, result bool) {
	if spec.NTPServers != nil {
		var timeservers string

		if len(*spec.NTPServers) != 0 {
			n := starlingxv1.NTPServerListToStrings(*spec.NTPServers)
			timeservers = strings.Join(n, ",")
		} else {
			timeservers = NoContent
		}

		if (timeservers != NoContent && info.NTPServers != timeservers) || timeservers == NoContent && info.NTPServers != "" {
			ntpOpts.NTPServers = &timeservers
			result = true
		}
	}

	return ntpOpts, result
}

// ReconcileNTP configures the system resources to align with the desired NTP state.
func (r *SystemReconciler) ReconcileNTP(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	if !utils.IsReconcilerEnabled(utils.NTP) {
		return nil
	}

	if ntpOpts, ok := ntpUpdateRequired(spec, info.NTP); ok {
		logSystem.Info("updating NTP servers", "opts", ntpOpts)

		result, err := ntp.Update(client, info.NTP.ID, ntpOpts).Extract()
		if err != nil {
			return err
		}

		info.NTP = result

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "NTP servers have been updated")
	}

	return nil
}

// ReconcileStorageBackend configures the storage Backend to align with the desired Ceph State
// Only supports creating storage backends
func (r *SystemReconciler) ReconcileStorageBackends(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	if !utils.IsReconcilerEnabled(utils.Backends) {
		return nil
	}
	if spec.Storage == nil {
		return nil
	}

	if spec.Storage.Backends == nil {
		return nil
	}

	updated := false
	for _, spec_sb := range *spec.Storage.Backends {
		found := false
		for _, info_sb := range info.StorageBackends {
			// The Type parameter in the spec maps to the Backend
			// parameter in the request
			if info_sb.Backend == spec_sb.Type &&
				info_sb.Name == spec_sb.Name {
				found = true
				break
			}
		}

		if found {
			continue
		}

		// In order to apply the backend config, we must set Confirmed
		// to true
		opts := storagebackends.StorageBackendOpts{
			Confirmed: true,
			Backend:   &spec_sb.Type,
			Name:      &spec_sb.Name,
			Network:   spec_sb.Network,
		}

		// Replication is an optional parameter.
		// In the spec, the parameter is named ReplicationFactor,
		// and it maps to the replication key in the Capabilities
		// dictionary
		if spec_sb.ReplicationFactor != nil {
			capabilities := make(map[string]interface{})
			capabilities["replication"] = strconv.Itoa(*spec_sb.ReplicationFactor)
			opts.Capabilities = &capabilities
		}

		result, err := storagebackends.Create(client, opts).Extract()
		if err != nil {
			return err
		}
		updated = true
		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceCreated, "%s storage backend created", result.Name)
	}

	if updated {
		result, err := storagebackends.ListBackends(client)
		if err != nil {
			err = perrors.Wrap(err, "failed to refresh storage backends")
			return err
		}
		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "StorageBackend info has been updated")
		info.StorageBackends = result
	}

	return nil
}

// dnsUpdateRequired determines whether an update is required to the DNS
// system attributes and returns the attributes to be changed if an update
// is necessary.
func dnsUpdateRequired(spec *starlingxv1.SystemSpec, info *dns.DNS) (dnsOpts dns.DNSOpts, result bool) {
	if spec.DNSServers != nil {
		var nameservers string

		if len(*spec.DNSServers) != 0 {
			d := starlingxv1.DNSServerListToStrings(*spec.DNSServers)
			nameservers = strings.Join(d, ",")
		} else {
			nameservers = NoContent
		}

		if (nameservers != NoContent && info.Nameservers != nameservers) || nameservers == NoContent && info.Nameservers != "" {
			dnsOpts.Nameservers = &nameservers
			result = true
		}
	}

	return dnsOpts, result
}

// ReconcileDNS configures the system resources to align with the desired DNS
// configuration.
func (r *SystemReconciler) ReconcileDNS(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	if !utils.IsReconcilerEnabled(utils.DNS) {
		return nil
	}

	if dnsOpts, ok := dnsUpdateRequired(spec, info.DNS); ok {
		logSystem.Info("updating DNS servers", "opts", dnsOpts)

		result, err := dns.Update(client, info.DNS.ID, dnsOpts).Extract()
		if err != nil {
			return err
		}

		info.DNS = result

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "DNS servers have been updated")
	}

	return nil
}

// drbdUpdateRequired determines whether an update is required to the DRBD
// system attributes and returns the attributes to be changed if an update
// is necessary.
func drbdUpdateRequired(spec *starlingxv1.SystemSpec, info *drbd.DRBD) (drbdOpts drbd.DRBDOpts, result bool) {
	if spec.Storage != nil && spec.Storage.DRBD != nil {
		if spec.Storage.DRBD.LinkUtilization != info.LinkUtilization {
			drbdOpts.LinkUtilization = spec.Storage.DRBD.LinkUtilization
			result = true
		}
	}

	return drbdOpts, result
}

// ReconcileDRBD configures the system resources to align with the desired DRBD
// configuration.
func (r *SystemReconciler) ReconcileDRBD(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	if !utils.IsReconcilerEnabled(utils.DRBD) {
		return nil
	}

	if drbdOpts, ok := drbdUpdateRequired(spec, info.DRBD); ok {
		logSystem.Info("updating DRBD configuration", "opts", drbdOpts)

		result, err := drbd.Update(client, info.DRBD.ID, drbdOpts).Extract()
		if err != nil {
			return err
		}

		info.DRBD = result

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "DRBD configuration has been updated")
	}

	return nil
}

// ptpUpdateRequired determines whether an update is required to the PTP
// system attributes and returns the attributes to be changed if an update
// is necessary.
func ptpUpdateRequired(spec *starlingxv1.PTPInfo, p *ptp.PTP) (ptpOpts ptp.PTPOpts, result bool) {
	if spec != nil {
		if spec.Mode != nil && *spec.Mode != p.Mode {
			ptpOpts.Mode = spec.Mode
			result = true
		}

		if spec.Mechanism != nil && *spec.Mechanism != p.Mechanism {
			ptpOpts.Mechanism = spec.Mechanism
			result = true
		}

		if spec.Transport != nil && *spec.Transport != p.Transport {
			ptpOpts.Transport = spec.Transport
			result = true
		}
	}

	return ptpOpts, result
}

// ReconcilePTP configures the system resources to align with the desired PTP state.
func (r *SystemReconciler) ReconcilePTP(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	if !utils.IsReconcilerEnabled(utils.PTP) {
		return nil
	}

	if ptpOpts, ok := ptpUpdateRequired(spec.PTP, info.PTP); ok {
		logSystem.Info("updating PTP config", "opts", ptpOpts)

		result, err := ptp.Update(client, info.PTP.ID, ptpOpts).Extract()
		if err != nil {
			return err
		}

		info.PTP = result

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "PTP info has been updated")
	}

	return nil
}

// serviceparametersUpdateRequired determines whether an update is required to the ServiceParameter
// and returns the field to be changed if an update is necessary.
// Only the value for a serviceparameter can be changed at this time.
// To modify service or section, a delete / create is used. (which is not supported by the API)
func serviceparametersUpdateRequired(spec *starlingxv1.ServiceParameterInfo, p *serviceparameters.ServiceParameter) (serviceparametersOpts serviceparameters.ServiceParameterPatchOpts, result bool) {
	// this method assumes it is only called when service, section and paramname are all equal
	if spec != nil {
		if spec.ParamValue != p.ParamValue {
			serviceparametersOpts.ParamValue = &spec.ParamValue
			result = true
		}
		// We only need to compare Resource if both are not nil
		// since it is not supported to remove the resource, or add one to an existing  service param
		if spec.Resource != nil && p.Resource != nil && *spec.Resource != *p.Resource {
			serviceparametersOpts.Resource = spec.Resource
			result = true
		}
		// We only need to compare Personality if both are not nil
		// since it is not supported to remove the resource, or add one to an existing  service param
		if spec.Personality != nil && p.Personality != nil && *spec.Personality != *p.Personality {
			serviceparametersOpts.Personality = spec.Personality
			result = true
		}
	}
	return serviceparametersOpts, result
}

// ReconcileServiceParameters configures the system resources to align with the desired ServiceParameter state.
func (r *SystemReconciler) ReconcileServiceParameters(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	if !utils.IsReconcilerEnabled(utils.ServiceParameters) {
		return nil
	}
	if spec.ServiceParameters == nil {
		return nil
	}
	updated := false
	for _, spec_sp := range *spec.ServiceParameters {
		found := false
		for _, info_sp := range info.ServiceParameters {
			// A match occurs when service, section and paramname are equal
			if info_sp.Service == spec_sp.Service &&
				info_sp.Section == spec_sp.Section &&
				info_sp.ParamName == spec_sp.ParamName {
				found = true
				if spOpts, ok := serviceparametersUpdateRequired(&spec_sp, &info_sp); ok {
					result, err := serviceparameters.Update(client, info_sp.ID, spOpts).Extract()
					if err != nil {
						return err
					}
					// success
					updated = true
					r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "ServiceParameter %q %q %q has been modified", result.Service, result.Section, result.ParamName)
				}
				break
			}
		}
		if !found {
			params := make(map[string]string)
			params[spec_sp.ParamName] = spec_sp.ParamValue
			opts := serviceparameters.ServiceParameterOpts{
				Service:     &spec_sp.Service,
				Section:     &spec_sp.Section,
				Parameters:  &params,
				Resource:    spec_sp.Resource,
				Personality: spec_sp.Personality,
			}

			result, err := serviceparameters.Create(client, opts).Extract()
			if err != nil {
				return err
			}
			// success
			updated = true
			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceCreated, "ServiceParameter %q %q %q has been created", result.Service, result.Section, result.ParamName)
		}

		// Note: There is no support in the reconcile for DELETE of service parameters, since there are
		// default service parameters setup by the system, that would complicate a reconcile.
	}
	// update the system object with the list of service params
	if updated {
		result, err := serviceparameters.ListServiceParameters(client)
		if err != nil {
			err = perrors.Wrap(err, "failed to refresh service parameters list")
			return err
		}
		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "ServiceParameter list info has been updated")
		info.ServiceParameters = result
	}

	return nil
}

func ControllerNodesAvailable(objects []hosts.Host, required int) bool {
	count := 0
	for _, host := range objects {
		if host.Personality == hosts.PersonalityController {
			if host.IsUnlockedEnabled() {
				if host.AvailabilityStatus == hosts.AvailAvailable {
					count += 1
				}
			}
		}
	}

	return count >= required
}

// ControllerNodesAvailable counts the number of nodes that are unlocked,
// enabled, and available.
func (r *SystemReconciler) ControllerNodesAvailable(required int) bool {
	return ControllerNodesAvailable(r.hosts, required)
}

// FileSystemResizeAllowed defines whether a particular file system can be
// resized.
func (r *SystemReconciler) FileSystemResizeAllowed(instance *starlingxv1.System, info *v1info.SystemInfo, fs controllerFilesystems.FileSystem) (ready bool, err error) {
	required := 2
	if strings.EqualFold(info.SystemMode, string(cloudManager.SystemModeSimplex)) {
		required = 1
	}

	if !r.ControllerNodesAvailable(required) {
		if instance.Status.DeploymentScope == cloudManager.ScopePrincipal {
			instance.Status.StrategyRequired = cloudManager.StrategyUnlockRequired
			r.CloudManager.SetResourceInfo(cloudManager.ResourceSystem, "", instance.Name, instance.Status.Reconciled, instance.Status.StrategyRequired)
			err := r.Client.Status().Update(context.TODO(), instance)
			if err != nil {
				err = perrors.Wrapf(err, "failed to update status: %s",
					common.FormatStruct(instance.Status))
				return false, err
			}
		}
		msg := fmt.Sprintf("waiting for %d controller(s) in available state before resizing filesystems", required)
		m := NewAvailableControllerNodeMonitor(instance, required)
		return false, r.CloudManager.StartMonitor(m, msg)
	}

	if fs.State == controllerFilesystems.ResizeInProgress {
		msg := fmt.Sprintf("filesystem resize operation already in progress on %q", fs.Name)
		m := NewFileSystemResizeMonitor(instance)
		return false, r.CloudManager.StartMonitor(m, msg)
	}

	ready = true

	return ready, err
}

// ReconcileFilesystems configures the system resources to align with the
// desired controller filesystem configuration.
func (r *SystemReconciler) ReconcileFileSystems(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) (err error) {
	if !utils.IsReconcilerEnabled(utils.SystemFileSystems) {
		return nil
	}

	if spec.Storage == nil || spec.Storage.FileSystems == nil {
		return nil
	}

	// Get a fresh snapshot of the current hosts.  These are used to search for
	// a matching host record if one is not already found as well as to
	// determine when it is safe/allowed to configure new hosts or unlock
	// existing hosts.
	// TODO(alegacy): move this to earlier in the reconcile loop.  For now,
	// since this is the only user then it can stay here.
	r.hosts, err = hosts.ListHosts(client)
	if err != nil {
		err = perrors.Wrap(err, "failed to list hosts")
		return err
	}

	updates := make([]controllerFilesystems.FileSystemOpts, 0)
	for _, fsInfo := range *spec.Storage.FileSystems {
		found := false
		for _, fs := range info.FileSystems {
			if fs.Name != fsInfo.Name {
				continue
			}

			found = true
			if fsInfo.Size > fs.Size {
				if ready, err := r.FileSystemResizeAllowed(instance, info, fs); !ready {
					return err
				}

				// Update the system resource with the new size.
				opts := controllerFilesystems.FileSystemOpts{
					Name: fsInfo.Name,
					Size: fsInfo.Size,
				}

				updates = append(updates, opts)
			}
		}

		if !found {
			msg := fmt.Sprintf("unknown controller filesystem %q", fsInfo.Name)
			return starlingxv1.NewMissingSystemResource(msg)
		}
	}

	if len(updates) > 0 {
		logSystem.Info("updating controller filesystem sizes", "opts", updates)

		err := controllerFilesystems.Update(client, info.ID, updates).ExtractErr()
		if err != nil {
			err = perrors.Wrapf(err, "failed to update filesystems sizes")
			return err
		}

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "filesystem sizes have been updated")
	}

	return nil
}

func systemUpdateRequired(instance *starlingxv1.System, spec *starlingxv1.SystemSpec, s *system.System) (opts system.SystemOpts, result bool) {
	if instance.Name != s.Name {
		result = true
		opts.Name = &instance.Name
	}

	if spec.Description != nil && *spec.Description != s.Description {
		result = true
		opts.Description = spec.Description
	}

	if spec.Contact != nil && *spec.Contact != s.Contact {
		result = true
		opts.Contact = spec.Contact
	}

	if spec.Location != nil && *spec.Location != s.Location {
		result = true
		opts.Location = spec.Location
	}

	if spec.Latitude != nil && *spec.Latitude != s.Latitude {
		result = true
		opts.Latitude = spec.Latitude
	}

	if spec.Longitude != nil && *spec.Longitude != s.Longitude {
		result = true
		opts.Longitude = spec.Longitude
	}

	if instance.HTTPSEnabled() != s.Capabilities.HTTPSEnabled {
		result = true
		value := strconv.FormatBool(instance.HTTPSEnabled())
		opts.HTTPSEnabled = &value
	}

	if spec.VSwitchType != nil && *spec.VSwitchType != s.Capabilities.VSwitchType {
		result = true
		opts.VSwitchType = spec.VSwitchType
	}

	return opts, result
}

// ReconcileSystemAttributes configures the system resources to align with the desired state.
func (r *SystemReconciler) ReconcileSystemAttributes(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	if utils.IsReconcilerEnabled(utils.System) {
		if opts, ok := systemUpdateRequired(instance, spec, &info.System); ok {
			logSystem.Info("updating system config", "opts", opts)

			result, err := system.Update(client, info.ID, opts).Extract()
			if err != nil {
				return err
			}

			info.System = *result

			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, "system has been updated")
		}
	}

	return nil
}

// HTTPSRequired determines whether an HTTPS connection is required for the
// purpose of installing system certificates.
func (r *SystemReconciler) HTTPSRequiredForCertificates() bool {
	value := utils.GetReconcilerOption(utils.Certificate, utils.HTTPSRequired)
	if value != nil {
		if required, ok := value.(bool); ok {
			return required
		} else {
			logSystem.Info("unexpected option type",
				"option", utils.HTTPSRequired, "type", reflect.TypeOf(value))
		}
	}

	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return true
}

func (r *SystemReconciler) PrivateKeyTranmissionAllowed(client *gophercloud.ServiceClient, info *v1info.SystemInfo) error {
	if r.HTTPSRequiredForCertificates() {
		if !info.Capabilities.HTTPSEnabled {
			// Do not send private key information in the clear.
			msg := "it is unsafe to install certificates while HTTPS is disabled"
			return common.NewSystemDependency(msg)
		}

		if strings.HasPrefix(client.Endpoint, cloudManager.HTTPPrefix) {
			// If HTTPS is enabled and we are still using an HTTPPrefix then either
			// the endpoint hasn't been switched over yet, or the user is trying
			// to do this through the internal URL so disallow, reset the client,
			// and try again.
			msg := "it is unsafe to install certificates thru a non HTTPS URL"
			return common.NewHTTPSClientRequired(msg)
		}
	} else {
		logSystem.Info("allowing certificates to be installed over HTTP connection")
	}

	return nil
}

// ReconcileCertificates configures the system certificates to align with the
// desired list of certificates.
func (r *SystemReconciler) ReconcileCertificates(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {

	var cert *x509.Certificate
	var certificateList []*certificates.Certificate

	if !utils.IsReconcilerEnabled(utils.Certificate) {
		return nil
	}

	if spec.Certificates == nil {
		return nil
	}

	// Certificates cannot be deleted once they are installed so look at the
	// list of certificates coming from the user and add any that are missing
	// from the system.
	updated := false
	for _, c := range *spec.Certificates {
		secret := v1.Secret{}

		secretName := types.NamespacedName{Namespace: instance.Namespace, Name: c.Secret}
		err := r.Client.Get(context.TODO(), secretName, &secret)
		if err != nil {
			if !errors.IsNotFound(err) {
				err = perrors.Wrap(err, "failed to get certificate secret")
				return err
			}

			// If we don't find the corresponding secret, this is most likely
			// a certificate installed outside the scope of deployment-manager
			// and will be ignored here.

			msg := fmt.Sprintf("skipping %q certificate %q from system", c.Type, c.Secret)
			r.ReconcilerEventLogger.WarningEvent(instance, common.ResourceDependency, msg)
			continue
		}

		pemBlock, ok := secret.Data[starlingxv1.SecretCertKey]
		if !ok {
			msg := fmt.Sprintf("missing %q key in certificate secret %s",
				starlingxv1.SecretCertKey, c.Secret)
			return common.NewUserDataError(msg)
		}

		block, _ := pem.Decode(pemBlock)
		if block == nil {
			msg := fmt.Sprintf("unexpected certificate contents in secret %s", c.Secret)
			return common.NewUserDataError(msg)
		}

		cert, err = x509.ParseCertificate(block.Bytes)
		if cert == nil || err != nil {
			msg := fmt.Sprintf("corrupt certificate contents in secret %s", c.Secret)
			return common.NewUserDataError(msg)
		}

		if c.PrivateKeyExpected() {
			if err := r.PrivateKeyTranmissionAllowed(client, info); err != nil {
				// The system is not in a state to safely transmit private key
				// information.
				return err
			}

			keyBytes, ok := secret.Data[starlingxv1.SecretPrivKeyKey]
			if !ok {
				msg := fmt.Sprintf("missing %q key in certificate secret %s",
					starlingxv1.SecretPrivKeyKey, c.Secret)
				return common.NewUserDataError(msg)
			}

			pemBlock = append(pemBlock, keyBytes...)
		}

		// The system API reports the serial number prepended with the mode as
		// a "signature" rather than the actual signature so replicate that here
		// for the purpose of comparisons.
		signature := fmt.Sprintf("%s_%d", c.Type, cert.SerialNumber)

		found := false
		for _, certificate := range info.Certificates {
			if certificate.Signature == signature {
				found = true
				break
			}
		}

		if !found {
			opts := certificates.CertificateOpts{
				Type: c.Type,
				File: pemBlock,
			}

			logSystem.Info("installing certificate", "signature", signature)

			certificateList, err = certificates.Create(client, opts).Extract()

			if err != nil {
				err = perrors.Wrapf(err, "failed to create certificate: %s", common.FormatStruct(opts))
				return err
			}
			for _, certificate := range certificateList {
				r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceCreated,
					"certificate %q has been installed", certificate.Signature)
			}
			updated = true
		}
	}

	if updated {

		result, err := certificates.ListCertificates(client)
		if err != nil {
			err = perrors.Wrap(err, "failed to refresh certificate list")
			return err
		}

		info.Certificates = result
	}

	return nil
}

// ReconcileLicense configures the system license to align with the desired
// license file.
func (r *SystemReconciler) ReconcileLicense(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	if !utils.IsReconcilerEnabled(utils.License) {
		return nil
	}

	if spec.License == nil {
		return nil
	}

	// The license cannot be deleted once installed.  Compare the file contents
	// and replace it if it does not match; otherwise take no action.
	secret := v1.Secret{}
	secretName := types.NamespacedName{Namespace: instance.Namespace, Name: spec.License.Secret}
	err := r.Client.Get(context.TODO(), secretName, &secret)
	if err != nil {
		if !errors.IsNotFound(err) {
			err = perrors.Wrap(err, "failed to get certificate secret")
			return err
		}

		msg := fmt.Sprintf("waiting for license %q to be created", spec.License.Secret)
		r.ReconcilerEventLogger.WarningEvent(instance, common.ResourceDependency, msg)
		return common.NewMissingKubernetesResource(msg)
	}

	contents, ok := secret.Data[starlingxv1.SecretLicenseContentKey]
	if !ok {
		msg := fmt.Sprintf("missing %q key in certificate secret %s",
			starlingxv1.SecretLicenseContentKey, spec.License.Secret)
		return common.NewUserDataError(msg)
	}

	if info.License == nil || info.License.Content != string(contents) {
		opts := licenses.LicenseOpts{
			Contents: contents,
		}

		checksum := md5.Sum(contents)
		logSystem.Info("installing license", "md5sum", hex.EncodeToString(checksum[:]))

		err = licenses.Create(client, opts).ExtractErr()
		if err != nil {
			err = perrors.Wrapf(err, "failed to install license: %s", common.FormatStruct(opts))
			return err
		}

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceCreated,
			"license has been installed")

		result, err := licenses.Get(client).Extract()
		if err != nil {
			err = perrors.Wrap(err, "failed to refresh license info")
			return err
		}

		info.License = result
	}

	return nil
}

// ReconcileSystemInitial is responsible for reconciling the system attributes
// that do not depend on any other resource types (i.e., hosts).  Its purpose
// is to get the system into a state in which other resources can be
// configured.
func (r *SystemReconciler) ReconcileSystemInitial(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	err := r.ReconcileSystemAttributes(client, instance, spec, info)
	if err != nil {
		return err
	}

	// Update the certificate/https as soon as possible so that all subsequent
	// communications with the system API are secure if that was the intent
	// of the user.
	err = r.ReconcileCertificates(client, instance, spec, info)
	if err != nil {
		return err
	}

	err = r.ReconcileLicense(client, instance, spec, info)
	if err != nil {
		return err
	}

	err = r.ReconcileDRBD(client, instance, spec, info)
	if err != nil {
		return err
	}

	err = r.ReconcileDNS(client, instance, spec, info)
	if err != nil {
		return err
	}

	err = r.ReconcileNTP(client, instance, spec, info)
	if err != nil {
		return err
	}

	err = r.ReconcilePTP(client, instance, spec, info)
	if err != nil {
		return err
	}

	err = r.ReconcileServiceParameters(client, instance, spec, info)
	if err != nil {
		return err
	}

	err = r.ReconcileStorageBackends(client, instance, spec, info)
	if err != nil {
		return err
	}

	return nil
}

// ReconcileSystemFinal is responsible for completing the configuration of the
// system entity by running all steps that can be completed in parallel with
// other resource types.  That is, once we know that the controllers are already
// enabled so that we can provision the file systems.
func (r *SystemReconciler) ReconcileSystemFinal(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	err := r.ReconcileFileSystems(client, instance, spec, info)
	if err != nil {
		return err
	}

	return nil
}

// ReconcileRequired determines whether reconciliation is allowed/required on
// the System resource.  Reconciliation is required if there is a difference
// between the configured Spec and the current system state.  Reconciliation
// is only allowed if the resource has not already been successfully reconciled
// at least once; or the user has overridden this check by adding an annotation
// on the resource.
func (r *SystemReconciler) ReconcileRequired(instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) (err error, required bool) {
	// Build a new system spec based on the current configuration so that
	// we can compare it to the desired configuration.
	if !instance.Status.Reconciled {
		// We have not reconciled at least once so skip this check and just
		// allow reconciliation to proceed.  This will ensure that attributes
		// that are not readily comparable with the DeepEqual (i.e., licenses
		// and certificates) will get handled properly when needed.
		return nil, true
	}

	current, err := starlingxv1.NewSystemSpec(*info)
	if err != nil {
		return err, false
	}

	if spec.DeepEqual(current) {
		logSystem.V(2).Info("no changes between spec and current configuration")
		instance.Status.Delta = ""
		return nil, false
	}

	logSystem.Info("spec is:", "values", spec)

	logSystem.Info("current is:", "values", current)

	deltaString, err := common.GetDeltaString(current, spec, common.SystemProperties)
	if err != nil {
		logSystem.Info(fmt.Sprintf("failed to get Delta status:  %s\n", err))
	}

	if deltaString != "" {
		logSystem.Info(fmt.Sprintf("delta configuration:%s\n", deltaString))
		instance.Status.Delta = deltaString
		err = r.Client.Status().Update(context.TODO(), instance)
		if err != nil {
			logSystem.Info(fmt.Sprintf("failed to update status:  %s\n", err))
		}
	}

	if instance.Status.Reconciled && r.StopAfterInSync() {
		// Do not process any further changes once we have reached a
		// synchronized state unless there is an annotation on the resource.
		if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
			msg := common.NoChangesAfterReconciled
			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated, msg)
			return common.NewChangeAfterInSync(msg), false
		} else {
			logSystem.Info(common.ChangedAllowedAfterReconciled)
		}
	}

	logSystem.V(2).Info("A System Reconcile is required")
	return nil, true
}

// ReconcileSystem is the main top level reconciler for System resources.
func (r *SystemReconciler) ReconcileSystem(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) (ready bool, err error) {

	if err, required := r.ReconcileRequired(instance, spec, info); err != nil {
		return instance.Status.Reconciled, err
	} else if !required {
		return instance.Status.Reconciled, nil
	}

	err = r.ReconcileSystemInitial(client, instance, spec, info)
	if err != nil {
		return instance.Status.Reconciled, err
	}

	err = r.ReconcileSystemFinal(client, instance, spec, info)
	if err != nil {
		return true, err
	}

	r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
		"system has been provisioned")

	return true, nil
}

// statusUpdateRequired determines whether the resource status attribute
// needs to be updated to reflect the current system status.
func (r *SystemReconciler) statusUpdateRequired(instance *starlingxv1.System, info v1info.SystemInfo, inSync bool) (result bool) {
	status := &instance.Status

	if status.ID != info.ID {
		result = true
		status.ID = info.ID
	}

	if status.InSync != inSync {
		result = true
		status.InSync = inSync
	}

	if status.InSync && !status.Reconciled {
		// Record the fact that we have reached inSync at least once.
		status.Reconciled = true
		status.ConfigurationUpdated = false
		status.StrategyRequired = cloudManager.StrategyNotRequired
		if instance.Status.DeploymentScope == cloudManager.ScopePrincipal {
			r.CloudManager.SetResourceInfo(cloudManager.ResourceSystem, "", instance.Name, status.Reconciled, status.StrategyRequired)
		}
		result = true
	}

	if !strings.EqualFold(status.SystemType, info.SystemType) {
		result = true
		status.SystemType = strings.ToLower(info.SystemType)
	}

	if !strings.EqualFold(status.SystemMode, info.SystemMode) {
		result = true
		status.SystemMode = strings.ToLower(info.SystemMode)
	}

	if !strings.EqualFold(status.SoftwareVersion, info.SoftwareVersion) {
		result = true
		status.SoftwareVersion = strings.ToLower(info.SoftwareVersion)
	}

	return result
}

// BuildSystemDefaults takes the current set of system attributes and builds a
// fake system object that can be used as a reference for the current settings
// applied to the system.  The default settings are saved on the system status.
func (r *SystemReconciler) BuildSystemDefaults(instance *starlingxv1.System, system v1info.SystemInfo) (*starlingxv1.SystemSpec, error) {
	defaults, err := starlingxv1.NewSystemSpec(system)
	if defaults == nil || err != nil {
		return nil, err
	}

	buffer, err := json.Marshal(defaults)
	if err != nil {
		err = perrors.Wrap(err, "failed to marshal system defaults")
		return nil, err
	}

	data := string(buffer)
	instance.Status.Defaults = &data

	err = r.Client.Status().Update(context.Background(), instance)
	if err != nil {
		err = perrors.Wrap(err, "failed to update system defaults")
		return nil, err
	}

	return defaults, nil
}

// GetHostDefaults retrieves the default attributes for a host.  The set of
// default attributes are collected from the host before any user configurations
// are applied.
func (r *SystemReconciler) GetSystemDefaults(instance *starlingxv1.System) (*starlingxv1.SystemSpec, error) {
	if instance.Status.Defaults == nil {
		return nil, nil
	}

	defaults := starlingxv1.SystemSpec{}
	err := json.Unmarshal([]byte(*instance.Status.Defaults), &defaults)
	if err != nil {
		err = perrors.Wrap(err, "failed to unmarshal system defaults")
		return nil, err
	}

	return &defaults, nil
}

// MergeSystemSpecs invokes the mergo.Merge API with our desired modifiers.
func MergeSystemSpecs(a, b *starlingxv1.SystemSpec) (*starlingxv1.SystemSpec, error) {
	t := common.DefaultMergeTransformer
	err := mergo.Merge(a, b, mergo.WithOverride, mergo.WithTransformers(t))
	if err != nil {
		err = perrors.Wrap(err, "mergo.Merge failed to merge profiles")
		return nil, err
	}

	return a, nil
}

// After MergeSystemSpecs fill out any missing optional value
func FillOptionalMergedSystemSpec(spec *starlingxv1.SystemSpec) (*starlingxv1.SystemSpec, error) {
	if spec.Storage != nil && spec.Storage.Backends != nil {
		backends := *spec.Storage.Backends
		for i := range backends {
			sb := backends[i]
			// Fill missing network parameter for ceph backend
			if sb.Type == "ceph" && sb.Network == nil {
				default_value := "mgmt"
				sb.Network = &default_value
			}
			backends[i] = sb
		}

		spec.Storage.Backends = &backends
	}

	return spec, nil
}

func (r *SystemReconciler) GetCertificateSignatures(instance *starlingxv1.System) error {
	var cert *x509.Certificate
	result := make([]starlingxv1.CertificateInfo, 0)

	if instance.Spec.Certificates == nil {
		return nil
	}

	for _, c := range *instance.Spec.Certificates {
		secret := v1.Secret{}

		secretName := types.NamespacedName{Namespace: instance.Namespace, Name: c.Secret}
		err := r.Client.Get(context.TODO(), secretName, &secret)
		if err != nil {
			if !errors.IsNotFound(err) {
				err = perrors.Wrap(err, "failed to get certificate secret")
				return err
			}

			// If we don't find the corresponding secret, this is most likely
			// a certificate installed outside the scope of deployment-manager
			// and will be ignored here.
			msg := fmt.Sprintf("skipping %q certificate %q from system", c.Type, c.Secret)
			r.ReconcilerEventLogger.WarningEvent(instance, common.ResourceDependency, msg)
		}

		pemBlock, ok := secret.Data[starlingxv1.SecretCertKey]
		if !ok {
			msg := fmt.Sprintf("missing %q key in certificate secret %s",
				starlingxv1.SecretCertKey, c.Secret)
			return common.NewUserDataError(msg)
		}

		block, _ := pem.Decode(pemBlock)
		if block == nil {
			msg := fmt.Sprintf("unexpected certificate contents in secret %s", c.Secret)
			return common.NewUserDataError(msg)
		}

		cert, err = x509.ParseCertificate(block.Bytes)
		if cert == nil || err != nil {
			msg := fmt.Sprintf("corrupt certificate contents in secret %s", c.Secret)
			return common.NewUserDataError(msg)
		}

		// Determine the "signature" based on the certificate type and the
		// serial number reported by the system API
		signature := fmt.Sprintf("%s_%d", c.Type, cert.SerialNumber)

		certificate := starlingxv1.CertificateInfo{
			Type:      c.Type,
			Secret:    c.Secret,
			Signature: signature,
		}
		result = append(result, certificate)
	}
	*instance.Spec.Certificates = result
	return nil
}

// ReconcileResource interacts with the system API in order to reconcile the
// state of a data network with the state stored in the k8s database.
func (r *SystemReconciler) ReconcileResource(client *gophercloud.ServiceClient, instance *starlingxv1.System) (err error) {

	systemInfo := v1info.SystemInfo{}
	err = systemInfo.PopulateSystemInfo(client)
	if err != nil {
		return err
	}

	defaults, err := r.GetSystemDefaults(instance)
	if err != nil {
		return err
	} else if defaults == nil {
		logSystem.Info("collecting system default values")

		defaults, err = r.BuildSystemDefaults(instance, systemInfo)
		if err != nil {
			return err
		}

		r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceCreated,
			"system defaults collected and stored")
	}

	// Same problem applies to the License file attribute
	defaults.License = nil

	err = r.GetCertificateSignatures(instance)
	if err != nil {
		return err
	}

	// Merge the system defaults with the desired attributes so that any
	// optional attributes not filled in by the user default to how the system
	// looked when it was first installed.
	temp_spec, err := MergeSystemSpecs(defaults, &instance.Spec)
	if err != nil {
		return err
	}

	spec, err := FillOptionalMergedSystemSpec(temp_spec)
	if err != nil {
		return err
	}

	ready, err := r.ReconcileSystem(client, instance, spec, &systemInfo)
	inSync := err == nil

	if ready {
		// Regardless of whether an error occurred, if the reconciling got
		// far enough along to get the system in a state in which the other
		// reconcilers can make progress than we need to mark the system as
		// being ready.
		if !r.CloudManager.GetSystemReady(instance.Namespace) {
			// Set the system type which may be used by other reconcilers to make
			// decisions about when to reconcile certain resources.
			value := strings.ToLower(systemInfo.System.SystemType)
			r.CloudManager.SetSystemType(instance.Namespace, cloudManager.SystemType(value))

			// Unblock all other controllers that are waiting to reconcile
			// resources.
			r.CloudManager.SetSystemReady(instance.Namespace, true)

			r.ReconcilerEventLogger.NormalEvent(instance, common.ResourceUpdated,
				"system is now ready for other reconcilers")

			err = r.CloudManager.NotifySystemDependencies(instance.Namespace)
			if err != nil {
				// Revert to not-ready so that when we reconcile the system
				// resource again we will push the change out to all other
				// reconcilers again.
				r.CloudManager.SetSystemReady(instance.Namespace, false)
				return err
			}
		}
	}

	if r.statusUpdateRequired(instance, systemInfo, inSync) {
		logSystem.Info("updating status for system", "status", instance.Status)

		err2 := r.Client.Status().Update(context.TODO(), instance)
		if err2 != nil {
			err2 = perrors.Wrap(err2, "failed to update system status")
			return err2
		}
	}

	logSystem.V(2).Info("reconcile finished", "error", err)

	return err
}

// StopAfterInSync determines whether the reconciler should continue processing
// change requests after the configuration has been reconciled a first time.
func (r *SystemReconciler) StopAfterInSync() bool {
	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return utils.GetReconcilerOptionBool(utils.System, utils.StopAfterInSync, true)
}

// Obtain deploymentScope value from configuration
// Taking this value from annotation in instacne
// (It seems Client.Get does not update Status value from configuration)
// "bootstrap" if "bootstrap" in configuration or deploymentScope not specified
// "principal" if "principal" in configuration
func (r *SystemReconciler) GetScopeConfig(instance *starlingxv1.System) (scope string, err error) {
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
func (r *SystemReconciler) UpdateConfigStatus(instance *starlingxv1.System) (err error) {
	deploymentScope, err := r.GetScopeConfig(instance)
	if err != nil {
		return err
	}
	logSystem.V(2).Info("deploymentScope in configuration", "deploymentScope", deploymentScope)

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
				r.CloudManager.SetResourceInfo(cloudManager.ResourceSystem, "", instance.Name, instance.Status.Reconciled, cloudManager.StrategyNotRequired)
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

// Reconcile reads that state of the cluster for a SystemNamespace object and makes
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=systems,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=systems/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=systems/finalizers,verbs=update
func (r *SystemReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	savedLog := logSystem
	logSystem = logSystem.WithName(request.NamespacedName.String())
	defer func() { logSystem = savedLog }()

	logSystem.V(2).Info("reconcile called")

	// Fetch the SystemNamespace instance
	instance := &starlingxv1.System{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		logSystem.Error(err, "unable to read object: %v", request)
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Update scope from configuration
	logSystem.V(2).Info("before UpdateConfigStatus", "instance", instance)
	err = r.UpdateConfigStatus(instance)
	if err != nil {
		logSystem.Error(err, "unable to update scope")
		return reconcile.Result{}, err
	}
	logSystem.V(2).Info("after UpdateConfigStatus", "instance", instance)

	// Cancel any existing monitors
	r.CloudManager.CancelMonitor(instance)

	err = r.installRootCertificates(instance)
	if err != nil {
		logSystem.Error(err, "failed to install root certificates")
		return r.ReconcilerErrorHandler.HandleReconcilerError(request, err)
	}

	platformClient := r.CloudManager.GetPlatformClient(request.Namespace)
	if platformClient == nil {
		// Create the platform client
		platformClient, err = r.CloudManager.BuildPlatformClient(request.Namespace, cloudManager.SystemEndpointName, cloudManager.SystemEndpointType)
		if err != nil {
			return r.ReconcilerErrorHandler.HandleReconcilerError(request, err)
		}

		if r.CloudManager.GetSystemReady(instance.Namespace) {
			// The system is already ready from a previous reconciliation so
			// we were simply refreshing the client from a past error state
			// therefore unblock other reconcilers now rather than wait for
			// the sync state to be reconfirmed.
			err = r.CloudManager.NotifySystemDependencies(instance.Namespace)
			if err != nil {
				return r.ReconcilerErrorHandler.HandleReconcilerError(request, err)
			}
		}
	}

	// If strategy is applied, start strategy monitor
	if instance.Status.StrategyApplied {
		logSystem.Info("Strategy applied, start strategy monitor")
		r.CloudManager.StrageySent()
		r.CloudManager.StartStrategyMonitor()
	} else {
		logSystem.V(2).Info("Strategy not applied")
	}

	err = r.ReconcileResource(platformClient, instance)
	if err != nil {
		return r.ReconcilerErrorHandler.HandleReconcilerError(request, err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SystemReconciler) SetupWithManager(mgr ctrl.Manager) error {
	tMgr := cloudManager.GetInstance(mgr)
	r.Manager = mgr
	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()
	r.CloudManager = tMgr
	r.ReconcilerErrorHandler = &common.ErrorHandler{
		CloudManager: tMgr,
		Logger:       logSystem}
	r.ReconcilerEventLogger = &common.EventLogger{
		EventRecorder: mgr.GetEventRecorderFor(SystemControllerName),
		Logger:        logSystem}
	return ctrl.NewControllerManagedBy(mgr).
		For(&starlingxv1.System{}).
		Complete(r)
}
