/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2024 Wind River Systems, Inc. */

package build

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"

	"github.com/ghodss/yaml"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresspools"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/datanetworks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/licenses"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/networkAddressPools"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/networks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinstances"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/ptpinterfaces"
	perrors "github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const yamlSeparator = "---\n"

// Builder is the deployment builder interface which exists to allow easier
// mocking for unit test development.
type Builder interface {
	Build() (*Deployment, error)
	AddSystemFilters(filters []SystemFilter)
	AddProfileFilters(filters []ProfileFilter)
	AddHostFilters(filters []HostFilter)
	AddPlatformNetworkFilters(filters []PlatformNetworkFilter)
}

// DeploymentBuilder is the concrete implementation of the builder interface
// which is capable of building a full deployment model based on a running
// system.
type DeploymentBuilder struct {
	client                 *gophercloud.ServiceClient
	namespace              string
	name                   string
	progressWriter         io.Writer
	systemFilters          []SystemFilter
	profileFilters         []ProfileFilter
	hostFilters            []HostFilter
	platformNetworkFilters []PlatformNetworkFilter
}

var defaultSystemFilters = []SystemFilter{
	NewServiceParametersSystemFilter(),
}

var defaultHostFilters = []HostFilter{
	NewController0Filter(),
	NewLoopbackInterfaceFilter(),
	NewLocationFilter(),
	NewAddressFilter(),
	NewBMAddressFilter(),
	NewStorageMonitorFilter(),
	NewInterfaceRemoveUuidFilter(),
	NewHostKernelFilter(),
}

var defaultPlatformNetworkFilters = []PlatformNetworkFilter{
	NewAddressPoolFilter(),
}

// NewDeploymentBuilder returns an instantiation of a deployment builder
// structure.
func NewDeploymentBuilder(client *gophercloud.ServiceClient, namespace string, name string, progressWriter io.Writer) *DeploymentBuilder {
	return &DeploymentBuilder{
		client:                 client,
		namespace:              namespace,
		name:                   name,
		progressWriter:         progressWriter,
		systemFilters:          defaultSystemFilters,
		platformNetworkFilters: defaultPlatformNetworkFilters,
		hostFilters:            defaultHostFilters}
}

// parseIncompleteSecret is a convenience unitilty function to parse an incompleteSecret
// from v1.Secret to IncompleteSecret to add the warning message to the data of
// secret to request the action from users.
func parseIncompleteSecret(secret *v1.Secret) *IncompleteSecret {
	warningMsg := "Warning: Incomplete secret, please replace it with the secret content"
	if secret.Type == v1.SecretTypeTLS {
		return &IncompleteSecret{
			TypeMeta:   secret.TypeMeta,
			ObjectMeta: secret.ObjectMeta,
			Type:       secret.Type,
			Data: map[string]string{
				v1.TLSCertKey:              warningMsg,
				v1.TLSPrivateKeyKey:        warningMsg,
				v1.ServiceAccountRootCAKey: warningMsg,
			},
		}
	}

	if secret.Type == v1.SecretTypeBasicAuth {
		return &IncompleteSecret{
			TypeMeta:   secret.TypeMeta,
			ObjectMeta: secret.ObjectMeta,
			Type:       secret.Type,
			Data: map[string]string{
				v1.BasicAuthUsernameKey: string(secret.Data["username"]),
				v1.BasicAuthPasswordKey: warningMsg,
			},
		}
	}

	// Return empty Data if the secret type is not listed above
	return &IncompleteSecret{
		TypeMeta:   secret.TypeMeta,
		ObjectMeta: secret.ObjectMeta,
		Type:       secret.Type,
		Data: map[string]string{
			"Fake Data": warningMsg,
		},
	}
}

// IncompleteSecret defines a struct that contains a warning message in the secret
// data if the secret is incomplete
type IncompleteSecret struct {
	TypeMeta   metav1.TypeMeta
	ObjectMeta metav1.ObjectMeta
	Type       v1.SecretType
	Data       map[string]string
}

// Deployment defines the structure used to store all of the details of a
// system deployment.  It includes all of the standard kubernetes objects as
// well as all of the CRD objects required to represent a full running system.
type Deployment struct {
	Namespace         v1.Namespace
	Secrets           []*v1.Secret
	IncompleteSecrets []*IncompleteSecret
	System            starlingxv1.System
	PlatformNetworks  []*starlingxv1.PlatformNetwork
	DataNetworks      []*starlingxv1.DataNetwork
	Profiles          []*starlingxv1.HostProfile
	Hosts             []*starlingxv1.Host
	PtpInstances      []*starlingxv1.PtpInstance
	PtpInterfaces     []*starlingxv1.PtpInterface
	AddressPools      []*starlingxv1.AddressPool
}

// progressUpdate is a utility method to write a progress log to the provided
// i/o writer interface.
func (db *DeploymentBuilder) progressUpdate(messagefmt string, args ...interface{}) {
	_, _ = fmt.Fprintf(db.progressWriter, messagefmt, args...)
	// Suppress errors
}

// AddSystemFilters adds a list of system filters to the set already present
// on the deployment builder (if any).
func (db *DeploymentBuilder) AddSystemFilters(filters []SystemFilter) {
	db.systemFilters = append(db.systemFilters, filters...)
}

// AddProfileFilters adds a list of profile filters to the set already present
// on the deployment builder (if any).
func (db *DeploymentBuilder) AddProfileFilters(filters []ProfileFilter) {
	db.profileFilters = append(db.profileFilters, filters...)
}

// AddHostFilters adds a list of profile filters to the set already present
// on the deployment builder (if any).
func (db *DeploymentBuilder) AddHostFilters(filters []HostFilter) {
	db.hostFilters = append(db.hostFilters, filters...)
}

func (db *DeploymentBuilder) AddPlatformNetworkFilters(filters []PlatformNetworkFilter) {
	db.platformNetworkFilters = append(db.platformNetworkFilters, filters...)
}

// Build is the main method which produces a deployment object based on a
// running system.
func (db *DeploymentBuilder) Build() (*Deployment, error) {
	deployment := Deployment{}

	db.progressUpdate("building deployment for system %q in namespace %q\n", db.name, db.namespace)

	db.progressUpdate("building namespace configuration\n")

	err := db.buildNamespace(&deployment)
	if err != nil {
		return nil, err
	}

	db.progressUpdate("building system configuration\n")

	systemInfo, err := db.buildSystem(&deployment)
	if err != nil {
		return nil, err
	}

	db.progressUpdate("building system endpoint secret configuration\n")

	err = db.buildEndpointSecret(&deployment)
	if err != nil {
		return nil, err
	}

	db.progressUpdate("building system license secret configuration\n")

	err = db.buildLicenseSecret(&deployment, systemInfo.License)
	if err != nil {
		return nil, err
	}

	db.progressUpdate("building certificate secret configurations\n")

	err = db.buildCertificateSecrets(&deployment)
	if err != nil {
		return nil, err
	}

	db.progressUpdate("building data network configurations\n")

	err = db.buildDataNetworks(&deployment)
	if err != nil {
		return nil, err
	}

	db.progressUpdate("building platform network and address pool configurations\n")

	err = db.buildPlatformNetworks(&deployment)
	if err != nil {
		return nil, err
	}

	db.progressUpdate("building PTP instance configurations\n")

	err = db.buildPTPInstances(&deployment)
	if err != nil {
		return nil, err
	}

	db.progressUpdate("building PTP interface configurations\n")

	err = db.buildPTPInterfaces(&deployment)
	if err != nil {
		return nil, err
	}

	db.progressUpdate("building host and profile configurations\n")

	err = db.buildHostsAndProfiles(&deployment)
	if err != nil {
		return nil, err
	}

	db.progressUpdate("re-running profile filters for second pass\n")

	err = db.filterHostProfiles(&deployment)
	if err != nil {
		return nil, err
	}

	db.progressUpdate("simplifying profile configurations\n")

	err = db.simplifyHostProfiles(&deployment)
	if err != nil {
		return nil, err
	}

	return &deployment, nil
}

// removeStatusFields is a utility function that removes any "status" attributes
// from the final deployment yaml.  The final deployment yaml is intended to be
// used as input to provision a new system and so all fields that would be
// rejected by the kubernetes API must be removed prior to use.
func removeStatusFields(a string) string {
	re := regexp.MustCompile("(?ms)^status.*?^(---|$)")
	return re.ReplaceAllString(a, "$1")
}

// removeCreationTimestamp is a utility function that removes the creation
// timestamp attribute from the final deployment yaml.  The final deployment
// yaml is intended to be used as input to provision a new system and so all
// fields that would be rejected by the kubernetes API must be removed prior to
// use.
func removeCreationTimestamp(a string) string {
	re := regexp.MustCompile("(?m)^.*?creationTimestamp:.*?$[\r\n]")
	return re.ReplaceAllString(a, "")
}

// ToYAML is a utility method to publish the system deployment instance as
// a YAML document.  Each distinct resource within the document will be
// separated by a "---" line.
func (d *Deployment) ToYAML() (string, error) {
	var b bytes.Buffer

	b.Write([]byte(yamlSeparator))

	buf, err := yaml.Marshal(d.Namespace)
	if err != nil {
		err = perrors.Wrap(err, "failed to render namespace to YAML")
		return "", err
	}

	b.Write(buf)
	b.Write([]byte(yamlSeparator))

	for _, s := range d.Secrets {
		buf, err := yaml.Marshal(s)
		if err != nil {
			err = perrors.Wrap(err, "failed to render secret to YAML")
			return "", err
		}

		b.Write(buf)
		b.Write([]byte(yamlSeparator))
	}

	for _, s := range d.IncompleteSecrets {
		buf, err := yaml.Marshal(s)
		if err != nil {
			err = perrors.Wrap(err, "failed to render secret to YAML")
			return "", err
		}

		b.Write(buf)
		b.Write([]byte(yamlSeparator))
	}

	buf, err = yaml.Marshal(d.System)
	if err != nil {
		err = perrors.Wrap(err, "failed to render system to YAML")
		return "", err
	}

	b.Write(buf)
	b.Write([]byte(yamlSeparator))

	for _, n := range d.PlatformNetworks {
		buf, err := yaml.Marshal(n)
		if err != nil {
			err = perrors.Wrap(err, "failed to render platform network to YAML")
			return "", err
		}

		b.Write(buf)
		b.Write([]byte(yamlSeparator))
	}

	for _, n := range d.AddressPools {
		buf, err := yaml.Marshal(n)
		if err != nil {
			err = perrors.Wrap(err, "failed to render address pools to YAML")
			return "", err
		}

		b.Write(buf)
		b.Write([]byte(yamlSeparator))
	}

	for _, n := range d.DataNetworks {
		buf, err := yaml.Marshal(n)
		if err != nil {
			err = perrors.Wrap(err, "failed to render data network to YAML")
			return "", err
		}

		b.Write(buf)
		b.Write([]byte(yamlSeparator))
	}

	for _, n := range d.PtpInstances {
		buf, err := yaml.Marshal(n)
		if err != nil {
			err = perrors.Wrap(err, "failed to render ptp instance to YAML")
			return "", err
		}

		b.Write(buf)
		b.Write([]byte(yamlSeparator))
	}

	for _, n := range d.PtpInterfaces {
		buf, err := yaml.Marshal(n)
		if err != nil {
			err = perrors.Wrap(err, "failed to render ptp interface to YAML")
			return "", err
		}

		b.Write(buf)
		b.Write([]byte(yamlSeparator))
	}

	for _, p := range d.Profiles {
		buf, err := yaml.Marshal(p)
		if err != nil {
			err = perrors.Wrap(err, "failed to render profile to YAML")
			return "", err
		}

		b.Write(buf)
		b.Write([]byte(yamlSeparator))
	}

	for _, h := range d.Hosts {
		buf, err := yaml.Marshal(h)
		if err != nil {
			err = perrors.Wrap(err, "failed to render host to YAML")
			return "", err
		}

		b.Write(buf)
		b.Write([]byte(yamlSeparator))
	}

	return removeCreationTimestamp(removeStatusFields(b.String())), nil
}

func (db *DeploymentBuilder) buildNamespace(d *Deployment) error {
	namespace, err := starlingxv1.NewNamespace(db.namespace)
	if err != nil {
		return err
	}

	namespace.DeepCopyInto(&d.Namespace)

	return nil
}

func (db *DeploymentBuilder) filterSystem(system *starlingxv1.System, deployment *Deployment) error {
	for _, f := range db.systemFilters {
		err := f.Filter(system, deployment)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DeploymentBuilder) buildSystem(d *Deployment) (*v1info.SystemInfo, error) {
	// Collect a snapshot of the system info.
	systemInfo := v1info.SystemInfo{}
	err := systemInfo.PopulateSystemInfo(db.client)
	if err != nil {
		return nil, err
	}

	// Build a System object from the system snapshot
	system, err := starlingxv1.NewSystem(db.namespace, db.name, systemInfo)
	if err != nil {
		return nil, err
	}

	db.progressUpdate("...filtering system attributes\n")

	err = db.filterSystem(system, d)
	if err != nil {
		return nil, err
	}

	system.DeepCopyInto(&d.System)

	return &systemInfo, nil
}

func NewEndpointSecretFromEnv(name string, namespace string) (*v1.Secret, error) {
	username := os.Getenv(manager.UsernameKey)
	password := os.Getenv(manager.PasswordKey)

	secret := v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: v1.SecretTypeOpaque,
		Data: map[string][]byte{
			manager.UsernameKey: []byte(username),
			manager.PasswordKey: []byte(password),
		},
		StringData: map[string]string{
			manager.RegionNameKey:  os.Getenv(manager.RegionNameKey),
			manager.DomainNameKey:  os.Getenv(manager.DomainNameKey),
			manager.ProjectNameKey: os.Getenv(manager.ProjectNameKey),
			manager.AuthUrlKey:     os.Getenv(manager.AuthUrlKey),
			manager.InterfaceKey:   os.Getenv(manager.InterfaceKey),
		},
	}

	keystoneRegion := os.Getenv(manager.KeystoneRegionNameKey)
	if keystoneRegion != "" {
		// This only appears to be applicable under certain circumstances so
		// rather than include an empty string for those times when it is not
		// needed then only include it when it is not blank.
		secret.StringData[manager.KeystoneRegionNameKey] = keystoneRegion
	}

	return &secret, nil
}

func (db *DeploymentBuilder) buildEndpointSecret(d *Deployment) error {
	cert, err := NewEndpointSecretFromEnv(manager.SystemEndpointSecretName, db.namespace)
	if err != nil {
		return err
	}

	d.Secrets = append(d.Secrets, cert)

	return nil
}

func (db *DeploymentBuilder) buildLicenseSecret(d *Deployment, license *licenses.License) error {
	if d.System.Spec.License != nil {
		secret, err := starlingxv1.NewLicenseSecret(d.System.Spec.License.Secret, db.namespace, license.Content)
		if err != nil {
			return err
		}
		d.Secrets = append(d.Secrets, secret)
	}

	return nil
}

func (db *DeploymentBuilder) buildCertificateSecrets(d *Deployment) error {
	// Create any system level secrets that are required to instantiate
	// certificates.
	if d.System.Spec.Certificates != nil {
		for _, c := range *d.System.Spec.Certificates {
			cert, err := starlingxv1.NewCertificateSecret(c.Secret, db.namespace)
			if err != nil {
				return err
			}
			incompleteSecret := parseIncompleteSecret(cert)
			d.IncompleteSecrets = append(d.IncompleteSecrets, incompleteSecret)
		}
	}

	return nil
}

func (db *DeploymentBuilder) buildDataNetworks(d *Deployment) error {
	results, err := datanetworks.ListDataNetworks(db.client)
	if err != nil {
		err = perrors.Wrap(err, "failed to list data networks")
		return err
	}

	nets := make([]*starlingxv1.DataNetwork, 0)
	for _, dn := range results {
		net, err := starlingxv1.NewDataNetwork(dn.Name, db.namespace, dn)
		if err != nil {
			return err
		}
		nets = append(nets, net)
	}

	d.DataNetworks = nets

	return nil
}

func (db *DeploymentBuilder) GetAssociatedAddressPools(networkName string, networkAddressPools []networkAddressPools.NetworkAddressPool, addressPools []addresspools.AddressPool) []addresspools.AddressPool {
	var associated_address_pools []addresspools.AddressPool
	for _, networkAddressPool := range networkAddressPools {
		if networkAddressPool.NetworkName == networkName {
			addressPool := utils.GetSystemAddrPoolByName(addressPools, networkAddressPool.AddressPoolName)
			associated_address_pools = append(associated_address_pools, *addressPool)

		}
	}
	return associated_address_pools
}

func (db *DeploymentBuilder) buildPlatformNetworks(d *Deployment) error {
	var associated_address_pools []addresspools.AddressPool
	var associated_pools_name []string
	var addressPool *starlingxv1.AddressPool
	results, err := networks.ListNetworks(db.client)
	if err != nil {
		err = perrors.Wrap(err, "failed to list platform networks")
		return err
	}

	pools, err := addresspools.ListAddressPools(db.client)
	if err != nil {
		err = perrors.Wrap(err, "failed to list address pools")
		return err
	}

	networkAddrPools, err := networkAddressPools.ListNetworkAddressPools(db.client)
	if err != nil {
		err = perrors.Wrap(err, "failed to list network address pools")
		return err
	}

	d.PlatformNetworks = make([]*starlingxv1.PlatformNetwork, 0)
	always_generate_networks := []string{"storage", "mgmt", "oam", "admin"}
	sort.Strings(always_generate_networks)
	for _, n := range results {
		associated_pools_name = []string{}
		index := sort.SearchStrings(always_generate_networks, n.Type)
		if index < len(always_generate_networks) && always_generate_networks[index] == n.Type {
			// With "--minimal-config", Exports platform networks / address pools belonging to only storage network
			// Without "--minimal-config", Exports all address pools and platform networks belongs to any of storage, admin, mgmt, oam networks
			associated_address_pools = db.GetAssociatedAddressPools(n.Name, networkAddrPools, pools)
			for _, pools := range associated_address_pools {
				associated_pools_name = append(associated_pools_name, pools.Name)
				addressPool, err = starlingxv1.NewAddressPool(db.namespace, pools)
				if err != nil {
					return err
				}
				d.AddressPools = append(d.AddressPools, addressPool)
			}
			net, err := starlingxv1.NewPlatformNetwork(db.namespace, associated_pools_name, n)
			if err != nil {
				return err
			}
			// We are executing platform network filters after appending because CoreNetworkFilter works on
			// d.PlatformNetworks rather than individual PlatformNetwork resource.
			// We are passing pointer to PlatformNetwork resource (net), hence, calling filterPlatformNetworks
			// before or after append wouldn't make any difference and either way it would work.
			d.PlatformNetworks = append(d.PlatformNetworks, net)
			err = db.filterPlatformNetworks(net, d)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (db *DeploymentBuilder) buildPTPInstances(d *Deployment) error {
	results, err := ptpinstances.ListPTPInstances(db.client)
	if err != nil {
		err = perrors.Wrap(err, "failed to list PTP instances")
		return err
	}
	if results == nil {
		return nil
	}

	instances := make([]*starlingxv1.PtpInstance, 0)
	for _, n := range results {
		single, err := starlingxv1.NewPTPInstance(n.Name, db.namespace, n)
		if err != nil {
			return err
		}
		instances = append(instances, single)
	}

	d.PtpInstances = instances

	return nil
}

func (db *DeploymentBuilder) buildPTPInterfaces(d *Deployment) error {
	results, err := ptpinterfaces.ListPTPInterfaces(db.client)
	if err != nil {
		err = perrors.Wrap(err, "failed to list PTP interfaces")
		return err
	}
	if results == nil {
		return nil
	}

	interfaces := make([]*starlingxv1.PtpInterface, 0)
	for _, n := range results {
		single, err := starlingxv1.NewPTPInterface(n.Name, db.namespace, n)
		if err != nil {
			return err
		}
		interfaces = append(interfaces, single)
	}

	d.PtpInterfaces = interfaces

	return nil
}

func isInterfaceInUse(ifname string, info *starlingxv1.InterfaceInfo) bool {
	for _, v := range info.VLAN {
		if ifname == v.Lower {
			return true
		}
	}

	for _, v := range info.VF {
		if ifname == v.Lower {
			return true
		}
	}

	for _, b := range info.Bond {
		if utils.ContainsString(b.Members, ifname) {
			return true
		}
	}

	return false
}

func (db *DeploymentBuilder) filterHost(profile *starlingxv1.HostProfile, host *starlingxv1.Host, deployment *Deployment) error {
	for _, f := range db.hostFilters {
		err := f.Filter(profile, host, deployment)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DeploymentBuilder) filterPlatformNetworks(platform_network *starlingxv1.PlatformNetwork, deployment *Deployment) error {
	for _, f := range db.platformNetworkFilters {
		err := f.Filter(platform_network, deployment)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DeploymentBuilder) filterHostProfile(profile *starlingxv1.HostProfile, deployment *Deployment) error {
	for _, f := range db.profileFilters {
		err := f.Filter(profile, deployment)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DeploymentBuilder) resetProfileFilters() {
	for _, f := range db.profileFilters {
		f.Reset()
	}
}

func (db *DeploymentBuilder) buildHostsAndProfiles(d *Deployment) error {
	results, err := hosts.ListHosts(db.client)
	if err != nil {
		err = perrors.Wrap(err, "failed to list hosts")
		return err
	}

	bmSecretGenerated := false
	for _, h := range results {
		db.resetProfileFilters()

		// Always use the hostname when it is available but fall back to the
		// host UUID if it is not configured.
		hostname := h.Hostname
		if hostname == "" {
			hostname = h.ID
		}

		db.progressUpdate("...Building host configuration for %q\n", hostname)

		hostInfo := v1info.HostInfo{}
		// Create a snapshot of the configuration for this host.
		err := hostInfo.PopulateHostInfo(db.client, h.ID)
		if err != nil {
			return err
		}

		// Create a host record for this entity.
		host, err := starlingxv1.NewHost(hostname, db.namespace, hostInfo)
		if err != nil {
			return err
		}

		db.progressUpdate("...Building host profile configuration for %q\n", hostname)

		// Create a full profile for this one host.
		profile, err := starlingxv1.NewHostProfile(hostname, db.namespace, hostInfo)
		if err != nil {
			return err
		}

		// The host already has the boot MAC stored
		profile.Spec.BootMAC = nil

		// Force the provisioning mode to static until there is a need to make
		// this optional.
		static := starlingxv1.ProvioningModeStatic
		profile.Spec.ProvisioningMode = &static

		// Link the host to this profile, but we may change the reference later
		// if we can determine that it shares the same profile as another
		// host.
		host.Spec.Profile = profile.Name

		// If the host is configured with a board management controller then we
		// need to generate a secret to be filled in at a later time with the
		// BMC password (if applicable)
		if profile.Spec.BoardManagement != nil && !bmSecretGenerated {
			bm := profile.Spec.BoardManagement
			if bm.Credentials != nil && bm.Credentials.Password != nil && h.BMUsername != nil {
				secret, err := starlingxv1.NewBMSecret(bm.Credentials.Password.Secret, db.namespace, *h.BMUsername)
				if err != nil {
					return err
				}

				// This only needs to be filled in once since in all likelihood
				// all nodes will be configured with the same credentials.  The
				// user is free to clone and modify the config on a per host
				// basis if needed.
				bmSecretGenerated = true
				incompleteSecret := parseIncompleteSecret(secret)
				d.IncompleteSecrets = append(d.IncompleteSecrets, incompleteSecret)
			}
		}

		db.progressUpdate("...Running profile filters for %q\n", profile.Name)

		// Some values need to be moved from the profile to the host overrides
		// to reflect that certain attributes are host specific.
		err = db.filterHost(profile, host, d)
		if err != nil {
			return err
		}

		// Some values are extraneous and can be removed to simplify the
		// final result.
		err = db.filterHostProfile(profile, d)
		if err != nil {
			return err
		}

		d.Hosts = append(d.Hosts, host)
		d.Profiles = append(d.Profiles, profile)
	}

	return nil
}

func (db *DeploymentBuilder) filterHostProfiles(d *Deployment) error {
	// Re-run the filters so that any two-pass filters can finalize their
	// actions
	for _, profile := range d.Profiles {
		db.progressUpdate("...Running profile filters for %q\n", profile.Name)

		err := db.filterHostProfile(profile, d)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DeploymentBuilder) simplifyHostProfiles(d *Deployment) error {
	profiles := make([]*starlingxv1.HostProfile, 0)
	for _, host := range d.Hosts {
		var profile *starlingxv1.HostProfile
		for _, p := range d.Profiles {
			if p.Name == host.Spec.Profile {
				profile = p
				break
			}
		}

		if profile == nil {
			return fmt.Errorf("unable to find profile %q", host.Spec.Profile)
		}

		for _, tmp := range d.Profiles {
			if tmp.Spec.DeepEqual(&profile.Spec) {
				// If a profile is identical to another one then don't bother
				// adding a separate duplicate entry.  Simply re-use the first
				// one found and discard the duplicate.
				host.Spec.Profile = tmp.Name
				break
			}
		}

		if host.Spec.Profile == profile.Name {
			profiles = append(profiles, profile)
		} else {
			db.progressUpdate("...Profile %q not unique using %q instead\n",
				profile.Name, host.Spec.Profile)
		}
	}

	d.Profiles = profiles

	return nil
}
