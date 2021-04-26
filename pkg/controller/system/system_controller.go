/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2021 Wind River Systems, Inc. */

package system

import (
	"context"
	"crypto/md5"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
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
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/pkg/apis/starlingx/v1"
	"github.com/wind-river/cloud-platform-deployment-manager/pkg/config"
	"github.com/wind-river/cloud-platform-deployment-manager/pkg/controller/common"
	cloudManager "github.com/wind-river/cloud-platform-deployment-manager/pkg/manager"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/pkg/platform"
	"io/ioutil"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
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
	"time"
)

var log = logf.Log.WithName("controller").WithName("system")

const ControllerName = "system-controller"

// Add creates a new SystemNamespace Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	tMgr := cloudManager.GetInstance(mgr)
	return &ReconcileSystem{
		Manager:      mgr,
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

	// Watch for changes to SystemNamespace
	err = c.Watch(&source.Kind{Type: &starlingxv1.System{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileSystem{}

// ReconcileSystem reconciles a SystemNamespace object
type ReconcileSystem struct {
	manager.Manager
	client.Client
	scheme *runtime.Scheme
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
	err = ioutil.WriteFile(path, data, 0600)
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
func (r *ReconcileSystem) installRootCertificates(instance *starlingxv1.System) error {
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
		err := r.Get(context.TODO(), secretName, &secret)
		if err != nil {
			return err
		}

		var caBytes []byte
		var ok bool
		numRetries := 30
		for iter := 0; iter < numRetries; iter++ {
			caBytes, ok = secret.Data[starlingxv1.SecretCaCertKey]
			if !ok {
				log.Info("Platform certificate CA not ready/available", "name", c.Secret)
				time.Sleep(5 * time.Second)
			} else {
				log.Info("Platform certificate CA ready!")
				break
			}
		}
		if !ok {
			log.Info("Continuing deployment without a CA certificate", "name", c.Secret)
			continue
		}

		filename := fmt.Sprintf("%s-%s-ca-cert.pem", instance.Namespace, c.Secret)
		err = InstallCertificate(filename, caBytes)
		if err != nil {
			log.Error(err, "failed to install root certificate")
			return err
		}

		r.NormalEvent(instance, common.ResourceCreated,
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
			timeservers = strings.Join(*spec.NTPServers, ",")
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
func (r *ReconcileSystem) ReconcileNTP(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	if !config.IsReconcilerEnabled(config.NTP) {
		return nil
	}

	if ntpOpts, ok := ntpUpdateRequired(spec, info.NTP); ok {
		log.Info("updating NTP servers", "opts", ntpOpts)

		result, err := ntp.Update(client, info.NTP.ID, ntpOpts).Extract()
		if err != nil {
			return err
		}

		info.NTP = result

		r.NormalEvent(instance, common.ResourceUpdated, "NTP servers have been updated")
	}

	return nil
}

// ReconcileStorageBackend configures the storage Backend to align with the desired Ceph State
// Only supports creating storage backends
func (r *ReconcileSystem) ReconcileStorageBackends(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	if !config.IsReconcilerEnabled(config.Backends) {
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
		r.NormalEvent(instance, common.ResourceCreated, "%s storage backend created", result.Name)
	}

	if updated {
		result, err := storagebackends.ListBackends(client)
		if err != nil {
			err = perrors.Wrap(err, "failed to refresh storage backends")
			return err
		}
		r.NormalEvent(instance, common.ResourceUpdated, "StorageBackend info has been updated")
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
			nameservers = strings.Join(*spec.DNSServers, ",")
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
func (r *ReconcileSystem) ReconcileDNS(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	if !config.IsReconcilerEnabled(config.DNS) {
		return nil
	}

	if dnsOpts, ok := dnsUpdateRequired(spec, info.DNS); ok {
		log.Info("updating DNS servers", "opts", dnsOpts)

		result, err := dns.Update(client, info.DNS.ID, dnsOpts).Extract()
		if err != nil {
			return err
		}

		info.DNS = result

		r.NormalEvent(instance, common.ResourceUpdated, "DNS servers have been updated")
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
func (r *ReconcileSystem) ReconcileDRBD(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	if !config.IsReconcilerEnabled(config.DRBD) {
		return nil
	}

	if drbdOpts, ok := drbdUpdateRequired(spec, info.DRBD); ok {
		log.Info("updating DRBD configuration", "opts", drbdOpts)

		result, err := drbd.Update(client, info.DRBD.ID, drbdOpts).Extract()
		if err != nil {
			return err
		}

		info.DRBD = result

		r.NormalEvent(instance, common.ResourceUpdated, "DRBD configuration has been updated")
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
func (r *ReconcileSystem) ReconcilePTP(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	if !config.IsReconcilerEnabled(config.PTP) {
		return nil
	}

	if ptpOpts, ok := ptpUpdateRequired(spec.PTP, info.PTP); ok {
		log.Info("updating PTP config", "opts", ptpOpts)

		result, err := ptp.Update(client, info.PTP.ID, ptpOpts).Extract()
		if err != nil {
			return err
		}

		info.PTP = result

		r.NormalEvent(instance, common.ResourceUpdated, "PTP info has been updated")
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
func (r *ReconcileSystem) ReconcileServiceParameters(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	if !config.IsReconcilerEnabled(config.ServiceParameters) {
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
					r.NormalEvent(instance, common.ResourceUpdated, "ServiceParameter %q %q %q has been modified", result.Service, result.Section, result.ParamName)
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
			r.NormalEvent(instance, common.ResourceCreated, "ServiceParameter %q %q %q has been created", result.Service, result.Section, result.ParamName)
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
		r.NormalEvent(instance, common.ResourceUpdated, "ServiceParameter list info has been updated")
		info.ServiceParameters = result
	}

	return nil
}

func controllerNodesAvailable(objects []hosts.Host, required int) bool {
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
func (r *ReconcileSystem) ControllerNodesAvailable(required int) bool {
	return controllerNodesAvailable(r.hosts, required)
}

// FileSystemResizeAllowed defines whether a particular file system can be
// resized.
func (r *ReconcileSystem) FileSystemResizeAllowed(instance *starlingxv1.System, info *v1info.SystemInfo, fs controllerFilesystems.FileSystem) (ready bool, err error) {
	required := 2
	if strings.EqualFold(info.SystemMode, string(cloudManager.SystemModeSimplex)) {
		required = 1
	}

	if !r.ControllerNodesAvailable(required) {
		msg := fmt.Sprintf("waiting for %d controller(s) in available state before resizing filesystems", required)
		m := NewAvailableControllerNodeMonitor(instance, required)
		return false, r.StartMonitor(m, msg)
	}

	if fs.State == controllerFilesystems.ResizeInProgress {
		msg := fmt.Sprintf("filesystem resize operation already in progress on %q", fs.Name)
		m := NewFileSystemResizeMonitor(instance)
		return false, r.StartMonitor(m, msg)
	}

	ready = true

	return ready, err
}

// ReconcileFilesystems configures the system resources to align with the
// desired controller filesystem configuration.
func (r *ReconcileSystem) ReconcileFileSystems(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) (err error) {
	if !config.IsReconcilerEnabled(config.SystemFileSystems) {
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
		log.Info("updating controller filesystem sizes", "opts", updates)

		err := controllerFilesystems.Update(client, info.ID, updates).ExtractErr()
		if err != nil {
			err = perrors.Wrapf(err, "failed to update filesystems sizes")
			return err
		}

		r.NormalEvent(instance, common.ResourceUpdated, "filesystem sizes have been updated")
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
func (r *ReconcileSystem) ReconcileSystemAttributes(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	if config.IsReconcilerEnabled(config.System) {
		if opts, ok := systemUpdateRequired(instance, spec, &info.System); ok {
			log.Info("updating system config", "opts", opts)

			result, err := system.Update(client, info.ID, opts).Extract()
			if err != nil {
				return err
			}

			info.System = *result

			r.NormalEvent(instance, common.ResourceUpdated, "system has been updated")
		}
	}

	return nil
}

// HTTPSRequired determines whether an HTTPS connection is required for the
// purpose of installing system certificates.
func (r *ReconcileSystem) HTTPSRequiredForCertificates() bool {
	value := config.GetReconcilerOption(config.Certificate, config.HTTPSRequired)
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

func (r *ReconcileSystem) PrivateKeyTranmissionAllowed(client *gophercloud.ServiceClient, info *v1info.SystemInfo) error {
	if r.HTTPSRequiredForCertificates() {
		if !info.Capabilities.HTTPSEnabled {
			// Do not send private key information in the clear.
			msg := fmt.Sprintf("it is unsafe to install certificates while HTTPS is disabled")
			return common.NewSystemDependency(msg)
		}

		if strings.HasPrefix(client.Endpoint, cloudManager.HTTPPrefix) {
			// If HTTPS is enabled and we are still using an HTTPPrefix then either
			// the endpoint hasn't been switched over yet, or the user is trying
			// to do this through the internal URL so disallow, reset the client,
			// and try again.
			msg := fmt.Sprintf("it is unsafe to install certificates thru a non HTTPS URL")
			return common.NewHTTPSClientRequired(msg)
		}
	} else {
		log.Info("allowing certificates to be installed over HTTP connection")
	}

	return nil
}

// ReconcileCertificates configures the system certificates to align with the
// desired list of certificates.
func (r *ReconcileSystem) ReconcileCertificates(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {

	var cert *x509.Certificate
	var certificateList []*certificates.Certificate

	if !config.IsReconcilerEnabled(config.Certificate) {
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
		err := r.Get(context.TODO(), secretName, &secret)
		if err != nil {
			if !errors.IsNotFound(err) {
				err = perrors.Wrap(err, "failed to get certificate secret")
				return err
			}

			// If we don't find the corresponding secret, this is most likely
			// a certificate installed outside the scope of deployment-manager
			// and will be ignored here.

			msg := fmt.Sprintf("skipping %q certificate %q from system", c.Type, c.Secret)
			r.WarningEvent(instance, common.ResourceDependency, msg)
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

			log.Info("installing certificate", "signature", signature)

			certificateList, err = certificates.Create(client, opts).Extract()

			if err != nil {
				err = perrors.Wrapf(err, "failed to create certificate: %s", common.FormatStruct(opts))
				return err
			}
			for _, certificate := range certificateList {
				r.NormalEvent(instance, common.ResourceCreated,
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
func (r *ReconcileSystem) ReconcileLicense(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
	if !config.IsReconcilerEnabled(config.License) {
		return nil
	}

	if spec.License == nil {
		return nil
	}

	// The license cannot be deleted once installed.  Compare the file contents
	// and replace it if it does not match; otherwise take no action.
	secret := v1.Secret{}
	secretName := types.NamespacedName{Namespace: instance.Namespace, Name: spec.License.Secret}
	err := r.Get(context.TODO(), secretName, &secret)
	if err != nil {
		if !errors.IsNotFound(err) {
			err = perrors.Wrap(err, "failed to get certificate secret")
			return err
		}

		msg := fmt.Sprintf("waiting for license %q to be created", spec.License.Secret)
		r.WarningEvent(instance, common.ResourceDependency, msg)
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
		log.Info("installing license", "md5sum", hex.EncodeToString(checksum[:]))

		err = licenses.Create(client, opts).ExtractErr()
		if err != nil {
			err = perrors.Wrapf(err, "failed to install license: %s", common.FormatStruct(opts))
			return err
		}

		r.NormalEvent(instance, common.ResourceCreated,
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
func (r *ReconcileSystem) ReconcileSystemInitial(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
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
func (r *ReconcileSystem) ReconcileSystemFinal(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) error {
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
func (r *ReconcileSystem) ReconcileRequired(instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) (err error, required bool) {
	// Build a new system spec based on the current configuration so that
	// we can compare it to the desired configuration.
	if !instance.Status.Reconciled {
		// We have not reconciled at least once so skip this check and just
		// allow reconciliation to proceed.  This will ensure that attributes
		// that are not readily comparable with the DeepEqual (i.e., licenses
		// and certificates) will get handled properly when needed.
		// TODO(alegacy):  This will need to be improved when we need to
		// support day-2 operations.
		return nil, true
	}

	current, err := starlingxv1.NewSystemSpec(*info)
	if err != nil {
		return err, false
	}

	if spec.DeepEqual(current) {
		log.V(2).Info("no changes between spec and current configuration")
		return nil, false
	}

	log.V(2).Info("spec is:", "values", spec)

	log.V(2).Info("current is:", "values", current)

	if instance.Status.Reconciled && r.StopAfterInSync() {
		// Do not process any further changes once we have reached a
		// synchronized state unless there is an annotation on the resource.
		if _, present := instance.Annotations[cloudManager.ReconcileAfterInSync]; !present {
			msg := common.NoChangesAfterReconciled
			r.NormalEvent(instance, common.ResourceUpdated, msg)
			return common.NewChangeAfterInSync(msg), false
		} else {
			log.Info(common.ChangedAllowedAfterReconciled)
		}
	}

	log.V(2).Info("A System Reconcile is required")
	return nil, true
}

// ReconcileSystem is the main top level reconciler for System resources.
func (r *ReconcileSystem) ReconcileSystem(client *gophercloud.ServiceClient, instance *starlingxv1.System, spec *starlingxv1.SystemSpec, info *v1info.SystemInfo) (ready bool, err error) {

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

	r.NormalEvent(instance, common.ResourceUpdated,
		"system has been provisioned")

	return true, nil
}

// statusUpdateRequired determines whether the resource status attribute
// needs to be updated to reflect the current system status.
func (r *ReconcileSystem) statusUpdateRequired(instance *starlingxv1.System, info v1info.SystemInfo, inSync bool) (result bool) {
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
func (r *ReconcileSystem) BuildSystemDefaults(instance *starlingxv1.System, system v1info.SystemInfo) (*starlingxv1.SystemSpec, error) {
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

	err = r.Status().Update(context.Background(), instance)
	if err != nil {
		err = perrors.Wrap(err, "failed to update system defaults")
		return nil, err
	}

	return defaults, nil
}

// GetHostDefaults retrieves the default attributes for a host.  The set of
// default attributes are collected from the host before any user configurations
// are applied.
func (r *ReconcileSystem) GetSystemDefaults(instance *starlingxv1.System) (*starlingxv1.SystemSpec, error) {
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

func (r *ReconcileSystem) GetCertificateSignatures(instance *starlingxv1.System) error {
	var cert *x509.Certificate
	result := make([]starlingxv1.CertificateInfo, 0)

	if instance.Spec.Certificates == nil {
		return nil
	}

	for _, c := range *instance.Spec.Certificates {
		secret := v1.Secret{}

		secretName := types.NamespacedName{Namespace: instance.Namespace, Name: c.Secret}
		err := r.Get(context.TODO(), secretName, &secret)
		if err != nil {
			if !errors.IsNotFound(err) {
				err = perrors.Wrap(err, "failed to get certificate secret")
				return err
			}

			// If we don't find the corresponding secret, this is most likely
			// a certificate installed outside the scope of deployment-manager
			// and will be ignored here.
			msg := fmt.Sprintf("skipping %q certificate %q from system", c.Type, c.Secret)
			r.WarningEvent(instance, common.ResourceDependency, msg)
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
func (r *ReconcileSystem) ReconcileResource(client *gophercloud.ServiceClient, instance *starlingxv1.System) (err error) {

	systemInfo := v1info.SystemInfo{}
	err = systemInfo.PopulateSystemInfo(client)
	if err != nil {
		return err
	}

	defaults, err := r.GetSystemDefaults(instance)
	if err != nil {
		return err
	} else if defaults == nil {
		log.Info("collecting system default values")

		defaults, err = r.BuildSystemDefaults(instance, systemInfo)
		if err != nil {
			return err
		}

		r.NormalEvent(instance, common.ResourceCreated,
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
	spec, err := MergeSystemSpecs(defaults, &instance.Spec)
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
		if !r.GetSystemReady(instance.Namespace) {
			// Set the system type which may be used by other reconcilers to make
			// decisions about when to reconcile certain resources.
			value := strings.ToLower(systemInfo.System.SystemType)
			r.SetSystemType(instance.Namespace, cloudManager.SystemType(value))

			// Unblock all other controllers that are waiting to reconcile
			// resources.
			r.SetSystemReady(instance.Namespace, true)

			r.NormalEvent(instance, common.ResourceUpdated,
				"system is now ready for other reconcilers")

			err = r.NotifySystemDependencies(instance.Namespace)
			if err != nil {
				// Revert to not-ready so that when we reconcile the system
				// resource again we will push the change out to all other
				// reconcilers again.
				r.SetSystemReady(instance.Namespace, false)
				return err
			}
		}
	}

	if r.statusUpdateRequired(instance, systemInfo, inSync) {
		log.Info("updating status for system", "status", instance.Status)

		err2 := r.Status().Update(context.TODO(), instance)
		if err2 != nil {
			err2 = perrors.Wrap(err2, "failed to update system status")
			return err2
		}
	}

	log.V(2).Info("reconcile finished", "error", err)

	return err
}

// StopAfterInSync determines whether the reconciler should continue processing
// change requests after the configuration has been reconciled a first time.
func (r *ReconcileSystem) StopAfterInSync() bool {
	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return config.GetReconcilerOptionBool(config.System, config.StopAfterInSync, true)
}

// Reconcile reads that state of the cluster for a SystemNamespace object and makes
// changes based on the state read and what is in the SystemNamespace.Spec
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=systems,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=starlingx.windriver.com,resources=systems/status,verbs=get;update;patch
func (r *ReconcileSystem) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	savedLog := log
	log = log.WithName(request.NamespacedName.String())
	defer func() { log = savedLog }()

	log.V(2).Info("reconcile called")

	// Fetch the SystemNamespace instance
	instance := &starlingxv1.System{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		log.Error(err, "unable to read object: %v", request)
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Cancel any existing monitors
	r.CancelMonitor(instance)

	err = r.installRootCertificates(instance)
	if err != nil {
		log.Error(err, "failed to install root certificates")
		return r.HandleReconcilerError(request, err)
	}

	platformClient := r.GetPlatformClient(request.Namespace)
	if platformClient == nil {
		// Create the platform client
		platformClient, err = r.BuildPlatformClient(request.Namespace)
		if err != nil {
			return r.HandleReconcilerError(request, err)
		}

		if r.GetSystemReady(instance.Namespace) {
			// The system is already ready from a previous reconciliation so
			// we were simply refreshing the client from a past error state
			// therefore unblock other reconcilers now rather than wait for
			// the sync state to be reconfirmed.
			err = r.NotifySystemDependencies(instance.Namespace)
			if err != nil {
				return r.HandleReconcilerError(request, err)
			}
		}
	}

	err = r.ReconcileResource(platformClient, instance)
	if err != nil {
		return r.HandleReconcilerError(request, err)
	}

	return reconcile.Result{}, nil
}
