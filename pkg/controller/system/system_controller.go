/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package system

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/certificates"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/dns"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/drbd"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/filesystems"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ntp"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptp"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/snmpCommunity"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/snmpTrapDest"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/system"
	"github.com/imdario/mergo"
	perrors "github.com/pkg/errors"
	starlingxv1beta1 "github.com/wind-river/titanium-deployment-manager/pkg/apis/starlingx/v1beta1"
	"github.com/wind-river/titanium-deployment-manager/pkg/controller/common"
	titaniumManager "github.com/wind-river/titanium-deployment-manager/pkg/manager"
	v1info "github.com/wind-river/titanium-deployment-manager/pkg/platform"
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
)

var log = logf.Log.WithName("controller").WithName("system")

const ControllerName = "system-controller"

// Defines the conversions between the schema certificate types and the
// expected API modes.
var CertificateTypeModes = map[string]string{
	starlingxv1beta1.PlatformCertificate:    "ssl",
	starlingxv1beta1.PlatformCACertificate:  "ssl_ca",
	starlingxv1beta1.OpenstackCertificate:   "openstack",
	starlingxv1beta1.OpenstackCACertificate: "openstack_ca",
	starlingxv1beta1.TPMCertificate:         "tpm_mode",
	starlingxv1beta1.DockerCertificate:      "docker_registry",
}

// Add creates a new SystemNamespace Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	tMgr := titaniumManager.GetInstance(mgr)
	return &ReconcileSystem{
		Manager:         mgr,
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

	// Watch for changes to SystemNamespace
	err = c.Watch(&source.Kind{Type: &starlingxv1beta1.System{}}, &handler.EnqueueRequestForObject{})
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
	titaniumManager.TitaniumManager
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
func (r *ReconcileSystem) installRootCertificates(instance *starlingxv1beta1.System) error {
	if instance.Spec.Certificates == nil {
		return nil
	}

	for _, c := range *instance.Spec.Certificates {
		if c.Type != starlingxv1beta1.PlatformCertificate {
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

		caEncodedBytes, ok := secret.Data[starlingxv1beta1.SecretCaCertKey]
		if !ok {
			// This can be valid as long as the target certificate is signed
			// by a known CA certificate; otherwise once the target certificate
			// is installed we will not be able to verify the TLS session.
			log.Info("platform certificate without a CA certificate; ignoring", "name", c.Secret)
			continue
		}

		caBytes := make([]byte, base64.StdEncoding.DecodedLen(len(caEncodedBytes)))
		_, err = base64.StdEncoding.Decode(caBytes, caEncodedBytes)

		filename := fmt.Sprintf("%s-%s-ca-cert.pem", instance.Namespace, c.Secret)
		err = InstallCertificate(filename, caEncodedBytes)
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
func ntpUpdateRequired(spec *starlingxv1beta1.SystemSpec, info *ntp.NTP) (ntpOpts ntp.NTPOpts, result bool) {
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

	if spec.PTP != nil {
		// Regardless of whether NTP was defined in the spec or not make sure
		// that the NTP state is compatible with the PTP state.
		if spec.PTP.Enabled && info.Enabled {
			value := strconv.FormatBool(false)
			ntpOpts.Enabled = &value
			result = true
		} else if spec.PTP.Enabled == false && info.Enabled == false {
			value := strconv.FormatBool(true)
			ntpOpts.Enabled = &value
			result = true
		}
	}

	return ntpOpts, result
}

// ReconcileNTP configures the system resources to align with the desired NTP state.
func (r *ReconcileSystem) ReconcileNTP(client *gophercloud.ServiceClient, instance *starlingxv1beta1.System, spec *starlingxv1beta1.SystemSpec, info *v1info.SystemInfo) error {
	if r.IsReconcilerEnabled(titaniumManager.NTP) == false {
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

// dnsUpdateRequired determines whether an update is required to the DNS
// system attributes and returns the attributes to be changed if an update
// is necessary.
func dnsUpdateRequired(spec *starlingxv1beta1.SystemSpec, info *dns.DNS) (dnsOpts dns.DNSOpts, result bool) {
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
func (r *ReconcileSystem) ReconcileDNS(client *gophercloud.ServiceClient, instance *starlingxv1beta1.System, spec *starlingxv1beta1.SystemSpec, info *v1info.SystemInfo) error {
	if r.IsReconcilerEnabled(titaniumManager.DNS) == false {
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
func drbdUpdateRequired(spec *starlingxv1beta1.SystemSpec, info *drbd.DRBD) (drbdOpts drbd.DRBDOpts, result bool) {
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
func (r *ReconcileSystem) ReconcileDRBD(client *gophercloud.ServiceClient, instance *starlingxv1beta1.System, spec *starlingxv1beta1.SystemSpec, info *v1info.SystemInfo) error {
	if r.IsReconcilerEnabled(titaniumManager.DRBD) == false {
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
func ptpUpdateRequired(spec *starlingxv1beta1.PTPInfo, p *ptp.PTP) (ptpOpts ptp.PTPOpts, result bool) {
	if spec != nil {
		if spec.Enabled != p.Enabled {
			ptpOpts.Enabled = &spec.Enabled
			result = true
		}

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
func (r *ReconcileSystem) ReconcilePTP(client *gophercloud.ServiceClient, instance *starlingxv1beta1.System, spec *starlingxv1beta1.SystemSpec, info *v1info.SystemInfo) error {
	if r.IsReconcilerEnabled(titaniumManager.PTP) == false {
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

// ReconcileSNMPCommunities configures the system resources to align with the
// desired SNMP Community string list.
func (r *ReconcileSystem) ReconcileSNMPCommunities(client *gophercloud.ServiceClient, instance *starlingxv1beta1.System, spec *starlingxv1beta1.SystemSpec, info *v1info.SystemInfo) error {
	configured := make(map[string]bool)

	if spec.SNMP == nil || spec.SNMP.Communities == nil {
		return nil
	}

	for _, name := range *spec.SNMP.Communities {
		configured[name] = true
		found := false
		for _, community := range info.SNMPCommunities {
			if community.Community == name {
				found = true
				break
			}
		}

		if found == false {
			opts := snmpCommunity.SNMPCommunityOpts{
				Community: &name,
			}

			log.Info("creating a new SNMP community string", "opts", opts)

			_, err := snmpCommunity.Create(client, opts).Extract()
			if err != nil {

			}

			r.NormalEvent(instance, common.ResourceUpdated, "SNMP community %q has been created", name)
		}
	}

	for _, c := range info.SNMPCommunities {
		if _, present := configured[c.Community]; !present {
			log.Info("deleting SNMP community string", "name", c.Community)

			err := snmpCommunity.Delete(client, c.ID).ExtractErr()
			if err != nil {
				return err
			}

			r.NormalEvent(instance, common.ResourceDeleted, "SNMP community %q has been deleted", c.Community)
		}
	}

	return nil
}

// ReconcileSNMPTrapDestinations configures the system resources to align with
// the desired SNMP Community string list.
func (r *ReconcileSystem) ReconcileSNMPTrapDestinations(client *gophercloud.ServiceClient, instance *starlingxv1beta1.System, spec *starlingxv1beta1.SystemSpec, info *v1info.SystemInfo) error {
	configured := make(map[string]bool)

	if spec.SNMP == nil || spec.SNMP.TrapDestinations == nil {
		return nil
	}

	for _, destInfo := range *spec.SNMP.TrapDestinations {
		configured[destInfo.IPAddress] = true
		found := false
		for _, dest := range info.SNMPTrapDestinations {
			if dest.IPAddress == destInfo.IPAddress {
				found = true
				break
			}
		}

		if found == false {
			opts := snmpTrapDest.SNMPTrapDestOpts{
				Community: &destInfo.Community,
				IPAddress: &destInfo.IPAddress,
			}

			log.Info("creating a new SNMP Trap Destination", "opts", opts)

			_, err := snmpTrapDest.Create(client, opts).Extract()
			if err != nil {

			}

			r.NormalEvent(instance, common.ResourceUpdated, "SNMP Trap Destination %q has been created", destInfo.IPAddress)
		}
	}

	for _, d := range info.SNMPTrapDestinations {
		if _, present := configured[d.IPAddress]; !present {
			log.Info("deleting SNMP Trap Destination", "name", d.IPAddress, "community", d.Community)

			err := snmpTrapDest.Delete(client, d.ID).ExtractErr()
			if err != nil {
				return err
			}

			r.NormalEvent(instance, common.ResourceDeleted, "SNMP Trap Destination %q has been deleted", d.IPAddress)
		}
	}

	return nil
}

// ReconcileSNMP configures the system resources to align with the desired SNMP
// configuration.
func (r *ReconcileSystem) ReconcileSNMP(client *gophercloud.ServiceClient, instance *starlingxv1beta1.System, spec *starlingxv1beta1.SystemSpec, info *v1info.SystemInfo) error {
	if r.IsReconcilerEnabled(titaniumManager.SNMP) == false {
		return nil
	}

	err := r.ReconcileSNMPCommunities(client, instance, spec, info)
	if err != nil {
		return nil
	}

	err = r.ReconcileSNMPTrapDestinations(client, instance, spec, info)
	if err != nil {
		return nil
	}

	return nil
}

// ControllerNodesAvailable counts the number of nodes that are unlocked,
// enabled, and available.
func (r *ReconcileSystem) ControllerNodesAvailable() int {
	count := 0
	for _, host := range r.hosts {
		if host.Personality == hosts.PersonalityController {
			if host.IsUnlockedEnabled() {
				if host.AvailabilityStatus == hosts.AvailAvailable {
					count += 1
				}
			}
		}
	}

	return count
}

// FileSystemResizeAllowed defines whether a particular file system can be
// resized.
func (r *ReconcileSystem) FileSystemResizeAllowed(client *gophercloud.ServiceClient, info *v1info.SystemInfo, fsInfo starlingxv1beta1.FileSystemInfo, fs filesystems.FileSystem) (ready bool, err error) {
	required := 2
	if strings.EqualFold(info.SystemMode, string(titaniumManager.SystemModeSimplex)) {
		required = 1
	}

	if r.ControllerNodesAvailable() < required {
		msg := fmt.Sprintf("waiting for %d controller(s) in available state before resizing filesystems", required)
		return false, common.NewResourceStatusDependency(msg)
	}

	if fs.State == filesystems.ResizeInProgress {
		msg := fmt.Sprintf("filesystem resize operation already in progress on %q", fs.Name)
		return false, common.NewResourceStatusDependency(msg)
	}

	ready = true

	return ready, err
}

// ReconcileFilesystems configures the system resources to align with the
// desired controller filesystem configuration.
func (r *ReconcileSystem) ReconcileFileSystems(client *gophercloud.ServiceClient, instance *starlingxv1beta1.System, spec *starlingxv1beta1.SystemSpec, info *v1info.SystemInfo) (err error) {
	if r.IsReconcilerEnabled(titaniumManager.FileSystems) == false {
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

	updates := make([]filesystems.FileSystemOpts, 0)
	for _, fsInfo := range *spec.Storage.FileSystems {
		found := false
		for _, fs := range info.FileSystems {
			if fs.Name != fsInfo.Name {
				continue
			}

			found = true
			if fsInfo.Size > fs.Size {
				found = true

				if ready, err := r.FileSystemResizeAllowed(client, info, fsInfo, fs); !ready {
					return err
				}

				// Update the system resource with the new size.
				opts := filesystems.FileSystemOpts{
					Name: fsInfo.Name,
					Size: fsInfo.Size,
				}

				updates = append(updates, opts)
			}
		}

		if found == false {
			msg := fmt.Sprintf("unknown controller filesystem %q", fsInfo.Name)
			return starlingxv1beta1.NewMissingSystemResource(msg)
		}
	}

	if len(updates) > 0 {
		log.Info("updating controller filesystem sizes", "opts", updates)

		err := filesystems.Update(client, info.ID, updates).ExtractErr()
		if err != nil {
			err = perrors.Wrapf(err, "failed to update filesystems sizes")
			return err
		}

		r.NormalEvent(instance, common.ResourceUpdated, "filesystem sizes have been updated")
	}

	return nil
}

func systemUpdateRequired(instance *starlingxv1beta1.System, spec *starlingxv1beta1.SystemSpec, s *system.System) (opts system.SystemOpts, result bool) {
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
func (r *ReconcileSystem) ReconcileSystemAttributes(client *gophercloud.ServiceClient, instance *starlingxv1beta1.System, spec *starlingxv1beta1.SystemSpec, info *v1info.SystemInfo) error {
	if r.IsReconcilerEnabled(titaniumManager.System) == true {
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

	// Set the system type which may be used by other reconcilers to make
	// decisions about when to reconcile certain resources.
	value := strings.ToLower(info.System.SystemType)
	r.SetSystemType(instance.Namespace, titaniumManager.SystemType(value))

	return nil
}

// HTTPSRequired determines whether an HTTPS connection is required for the
// purpose of installing system certificates.
func (r *ReconcileSystem) HTTPSRequiredForCertificates() bool {
	value := r.GetReconcilerOption(titaniumManager.Certificate, titaniumManager.HTTPSRequired)
	if value != nil {
		if required, ok := value.(bool); ok {
			return required
		} else {
			log.Info("unexpected option type",
				"option", titaniumManager.HTTPSRequired, "type", reflect.TypeOf(value))
		}
	}

	// If the option is not found or the option was specified in a form other
	// than a bool then assume the safest default value possible.
	return true
}

func (r *ReconcileSystem) PrivateKeyTranmissionAllowed(client *gophercloud.ServiceClient, info *v1info.SystemInfo) error {
	if r.HTTPSRequiredForCertificates() {
		if info.Capabilities.HTTPSEnabled == false {
			// Do not send private key information in the clear.
			msg := fmt.Sprintf("it is unsafe to install certificates while HTTPS is disabled")
			return common.NewSystemDependency(msg)
		}

		if strings.HasPrefix(client.Endpoint, titaniumManager.HTTPPrefix) {
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
func (r *ReconcileSystem) ReconcileCertificates(client *gophercloud.ServiceClient, instance *starlingxv1beta1.System, spec *starlingxv1beta1.SystemSpec, info *v1info.SystemInfo) error {
	var cert *x509.Certificate
	var result *certificates.Certificate

	if r.IsReconcilerEnabled(titaniumManager.Certificate) == false {
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
			if errors.IsNotFound(err) == false {
				err = perrors.Wrap(err, "failed to get certificate secret")
				return err
			}

			msg := fmt.Sprintf("waiting for %q certificate %q to be created", c.Type, c.Secret)
			r.WarningEvent(instance, common.ResourceDependency, msg)
			return common.NewMissingKubernetesResource(msg)
		}

		pemBlock, ok := secret.Data[starlingxv1beta1.SecretCertKey]
		if !ok {
			msg := fmt.Sprintf("missing %q key in certificate secret %s",
				starlingxv1beta1.SecretCertKey, c.Secret)
			return common.NewUserDataError(msg)
		}

		block, _ := pem.Decode([]byte(pemBlock))
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

			keyBytes, ok := secret.Data[starlingxv1beta1.SecretPrivKeyKey]
			if !ok {
				msg := fmt.Sprintf("missing %q key in certificate secret %s",
					starlingxv1beta1.SecretPrivKeyKey, c.Secret)
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

		if found == false {
			mode, ok := CertificateTypeModes[c.Type]
			if !ok {
				return perrors.Errorf("no conversion from certificate %q to mode", c.Type)
			}

			opts := certificates.CertificateOpts{
				Mode: mode,
				File: pemBlock,
			}

			log.Info("installing certificate", "signature", signature)

			result, err = certificates.Create(client, opts).Extract()
			if err != nil {
				err = perrors.Wrapf(err, "failed to create certificate: %s", common.FormatStruct(opts))
				return err
			}

			r.NormalEvent(instance, common.ResourceCreated,
				"certificate %q has been installed", result.Signature)

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

func (r *ReconcileSystem) ReconcileSystem(client *gophercloud.ServiceClient, instance *starlingxv1beta1.System, spec *starlingxv1beta1.SystemSpec, info *v1info.SystemInfo) error {

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

	err = r.ReconcileSNMP(client, instance, spec, info)
	if err != nil {
		return err
	}

	// At this point the system is ready for other resource types to be
	// reconciled (e.g., host, networks, etc).  We can set the ready flag and
	// notify other reconcilers but we still need to continue with other
	// attribute types that require hosts to be configured first.
	if r.GetSystemReady(instance.Namespace) == false {
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

	err = r.ReconcileFileSystems(client, instance, spec, info)
	if err != nil {
		return err
	}

	r.NormalEvent(instance, common.ResourceUpdated,
		"system has been provisioned")

	return nil
}

// statusUpdateRequired determines whether the resource status attribute
// needs to be updated to reflect the current system status.
func (r *ReconcileSystem) statusUpdateRequired(instance *starlingxv1beta1.System, info *v1info.SystemInfo, inSync bool) (result bool) {
	status := &instance.Status

	if status.ID != info.ID {
		result = true
		status.ID = info.ID
	}

	if status.InSync != inSync {
		result = true
		status.InSync = inSync
	}

	if strings.EqualFold(status.SystemType, info.SystemType) == false {
		result = true
		status.SystemType = strings.ToLower(info.SystemType)
	}

	if strings.EqualFold(status.SystemMode, info.SystemMode) == false {
		result = true
		status.SystemMode = strings.ToLower(info.SystemMode)
	}

	if strings.EqualFold(status.SoftwareVersion, info.SoftwareVersion) == false {
		result = true
		status.SoftwareVersion = strings.ToLower(info.SoftwareVersion)
	}

	return result
}

// BuildSystemDefaults takes the current set of system attributes and builds a
// fake system object that can be used as a reference for the current settings
// applied to the system.  The default settings are saved on the system status.
func (r *ReconcileSystem) BuildSystemDefaults(instance *starlingxv1beta1.System, system *v1info.SystemInfo) (*starlingxv1beta1.SystemSpec, error) {
	defaults, err := starlingxv1beta1.NewSystemSpec(system)
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
func (r *ReconcileSystem) GetSystemDefaults(instance *starlingxv1beta1.System) (*starlingxv1beta1.SystemSpec, error) {
	if instance.Status.Defaults == nil {
		return nil, nil
	}

	defaults := starlingxv1beta1.SystemSpec{}
	err := json.Unmarshal([]byte(*instance.Status.Defaults), &defaults)
	if err != nil {
		err = perrors.Wrap(err, "failed to unmarshal system defaults")
		return nil, err
	}

	return &defaults, nil
}

// MergeSystemSpecs invokes the mergo.Merge API with our desired modifiers.
func MergeSystemSpecs(a, b *starlingxv1beta1.SystemSpec) (*starlingxv1beta1.SystemSpec, error) {
	t := common.MergeTransformer{OverwriteSlices: true}
	err := mergo.Merge(a, b, mergo.WithOverride, mergo.WithTransformers(t))
	if err != nil {
		err = perrors.Wrap(err, "mergo.Merge failed to merge profiles")
		return nil, err
	}

	return a, nil
}

// ReconcileResource interacts with the system API in order to reconcile the
// state of a data network with the state stored in the k8s database.
func (r *ReconcileSystem) ReconcileResource(client *gophercloud.ServiceClient, instance *starlingxv1beta1.System) (err error) {

	systemInfo := &v1info.SystemInfo{}
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

	// Merge the system defaults with the desired attributes so that any
	// optional attributes not filled in by the user default to how the system
	// looked when it was first installed.
	spec, err := MergeSystemSpecs(defaults, &instance.Spec)
	if err != nil {
		return err
	}

	err = r.ReconcileSystem(client, instance, spec, systemInfo)
	inSync := err == nil

	if r.statusUpdateRequired(instance, systemInfo, inSync) {
		log.Info("updating status for system", "status", instance.Status)

		err2 := r.Status().Update(context.TODO(), instance)
		if err2 != nil {
			err2 = perrors.Wrap(err2, "failed to update system status")
			return err2
		}
	}

	log.V(1).Info("reconcile finished", "error", err)

	return err
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

	log.V(1).Info("reconcile called")

	// Fetch the SystemNamespace instance
	instance := &starlingxv1beta1.System{}
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
