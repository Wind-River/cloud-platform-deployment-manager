/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package build

import (
	"bytes"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/addresspools"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/datanetworks"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/networks"
	utils "github.com/wind-river/titanium-deployment-manager/pkg/common"
	"github.com/wind-river/titanium-deployment-manager/pkg/manager"
	"io"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"regexp"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	perrors "github.com/pkg/errors"
	"github.com/wind-river/titanium-deployment-manager/pkg/apis/starlingx/v1beta1"
	v1info "github.com/wind-river/titanium-deployment-manager/pkg/platform"
	"k8s.io/api/core/v1"
)

const yamlSeparator = "---\n"

// Builder is the deployment builder interface which exists to allow easier
// mocking for unit test development.
type Builder interface {
	Build() (*Deployment, error)
	AddSystemFilters(filters []SystemFilter)
	AddProfileFilters(filters []ProfileFilter)
	AddHostFilters(filters []HostFilter)
}

// DeploymentBuilder is the concrete implementation of the builder interface
// which is capable of building a full deployment model based on a running
// system.
type DeploymentBuilder struct {
	client         *gophercloud.ServiceClient
	namespace      string
	name           string
	progressWriter io.Writer
	systemFilters  []SystemFilter
	profileFilters []ProfileFilter
	hostFilters    []HostFilter
}

var defaultHostFilters = []HostFilter{
	NewController0Filter(),
	NewLoopbackInterfaceFilter(),
	NewAddressFilter(),
	NewBMAddressFilter(),
	NewStorageMonitorFilter(),
}

// NewDeploymentBuilder returns an instantiation of a deployment builder
// structure.
func NewDeploymentBuilder(client *gophercloud.ServiceClient, namespace string, name string, progressWriter io.Writer) *DeploymentBuilder {
	return &DeploymentBuilder{
		client:         client,
		namespace:      namespace,
		name:           name,
		progressWriter: progressWriter,
		hostFilters:    defaultHostFilters}
}

// Deployment defines the structure used to store all of the details of a
// system deployment.  It includes all of the standard kubernetes objects as
// well as all of the CRD objects required to represent a full running system.
type Deployment struct {
	Namespace         v1.Namespace
	Secrets           []*v1.Secret
	IncompleteSecrets []*v1.Secret
	System            v1beta1.System
	PlatformNetworks  []*v1beta1.PlatformNetwork
	DataNetworks      []*v1beta1.DataNetwork
	Profiles          []*v1beta1.HostProfile
	Hosts             []*v1beta1.Host
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

	err = db.buildSystem(&deployment)
	if err != nil {
		return nil, err
	}

	db.progressUpdate("building system endpoint secret configuration\n")

	err = db.buildEndpointSecret(&deployment)
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

	db.progressUpdate("building platform network configurations\n")

	err = db.buildPlatformNetworks(&deployment)
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

	for _, n := range d.DataNetworks {
		buf, err := yaml.Marshal(n)
		if err != nil {
			err = perrors.Wrap(err, "failed to render data network to YAML")
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
	namespace, err := v1beta1.NewNamespace(db.namespace)
	if err != nil {
		return err
	}

	namespace.DeepCopyInto(&d.Namespace)

	return nil
}

func (db *DeploymentBuilder) filterSystem(system *v1beta1.System, deployment *Deployment) error {
	for _, f := range db.systemFilters {
		err := f.Filter(system, deployment)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DeploymentBuilder) buildSystem(d *Deployment) error {
	// Collect a snapshot of the system info.
	systemInfo := v1info.SystemInfo{}
	err := systemInfo.PopulateSystemInfo(db.client)
	if err != nil {
		return err
	}

	// Build a System object from the system snapshot
	system, err := v1beta1.NewSystem(db.namespace, db.name, systemInfo)
	if err != nil {
		return err
	}

	db.progressUpdate("...filtering system attributes\n")

	err = db.filterSystem(system, d)
	if err != nil {
		return err
	}

	system.DeepCopyInto(&d.System)

	return nil
}

func NewEndpointSecretFromEnv(name string, namespace string) (*v1.Secret, error) {
	username := os.Getenv(manager.UsernameKey)
	password := os.Getenv(manager.PasswordKey)

	secret := v1.Secret{
		TypeMeta: v12.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: v12.ObjectMeta{
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

func (db *DeploymentBuilder) buildCertificateSecrets(d *Deployment) error {
	// Create any system level secrets that are required to instantiate
	// certificates.
	if d.System.Spec.Certificates != nil {
		for _, c := range *d.System.Spec.Certificates {
			cert, err := v1beta1.NewCertificateSecret(c.Secret, db.namespace)
			if err != nil {
				return err
			}
			d.IncompleteSecrets = append(d.IncompleteSecrets, cert)
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

	nets := make([]*v1beta1.DataNetwork, 0)
	for _, dn := range results {
		net, err := v1beta1.NewDataNetwork(dn.Name, db.namespace, dn)
		if err != nil {
			return err
		}
		nets = append(nets, net)
	}

	d.DataNetworks = nets

	return nil
}

func (db *DeploymentBuilder) buildPlatformNetworks(d *Deployment) error {
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

	nets := make([]*v1beta1.PlatformNetwork, 0)
	for _, p := range pools {
		skip := false
		for _, n := range results {
			if n.PoolUUID == p.ID {
				// TODO(alegacy): for now we only support networks used for data
				//  interfaces which are realized in the system as a standalone
				//  pool without a network so if we find a matching network then
				//  skip it.
				skip = true
				break
			}
		}

		if skip == false {
			net, err := v1beta1.NewPlatformNetwork(p.Name, db.namespace, p)
			if err != nil {
				return err
			}
			nets = append(nets, net)
		}
	}

	d.PlatformNetworks = nets

	return nil
}

func isInterfaceInUse(ifname string, info *v1beta1.InterfaceInfo) bool {
	for _, v := range info.VLAN {
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

func (db *DeploymentBuilder) filterHost(profile *v1beta1.HostProfile, host *v1beta1.Host, deployment *Deployment) error {
	for _, f := range db.hostFilters {
		err := f.Filter(profile, host, deployment)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DeploymentBuilder) filterHostProfile(profile *v1beta1.HostProfile, deployment *Deployment) error {
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
		host, err := v1beta1.NewHost(hostname, db.namespace, hostInfo)
		if err != nil {
			return err
		}

		db.progressUpdate("...Building host profile configuration for %q\n", hostname)

		// Create a full profile for this one host.
		profile, err := v1beta1.NewHostProfile(hostname, db.namespace, hostInfo)
		if err != nil {
			return err
		}

		// The host already has the boot MAC stored
		profile.Spec.BootMAC = nil

		// Force the provisioning mode to static until there is a need to make
		// this optional.
		static := v1beta1.ProvioningModeStatic
		profile.Spec.ProvisioningMode = &static

		// Link the host to this profile, but we may change the reference later
		// if we can determine that it shares the same profile as another
		// host.
		host.Spec.Profile = profile.Name

		// If the host is configured with a board management controller then we
		// need to generate a secret to be filled in at a later time with the
		// BMC password (if applicable)
		if profile.Spec.BoardManagement != nil && bmSecretGenerated == false {
			bm := profile.Spec.BoardManagement
			if bm.Credentials.Password != nil && h.BMUsername != nil {
				secret, err := v1beta1.NewBMSecret(bm.Credentials.Password.Secret, db.namespace, *h.BMUsername)
				if err != nil {
					return err
				}

				// This only needs to be filled in once since in all likelihood
				// all nodes will be configured with the same credentials.  The
				// user is free to clone and modify the config on a per host
				// basis if needed.
				bmSecretGenerated = true
				d.IncompleteSecrets = append(d.IncompleteSecrets, secret)
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
	profiles := make([]*v1beta1.HostProfile, 0)
	for _, host := range d.Hosts {
		var profile *v1beta1.HostProfile
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
