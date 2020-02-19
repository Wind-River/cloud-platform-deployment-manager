/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// List of valid certificate types.  These must align with the system
	// API expected values.
	PlatformCertificate    = "ssl"
	PlatformCACertificate  = "ssl_ca"
	OpenstackCertificate   = "openstack"
	OpenstackCACertificate = "openstack_ca"
	DockerCertificate      = "docker_registry"
	TPMCertificate         = "tpm_mode"
)

const (
	// List of secret data attribute keys
	SecretCertKey           = "tls.crt"
	SecretPrivKeyKey        = "tls.key"
	SecretCaCertKey         = "ca.crt"
	SecretLicenseContentKey = "content"
)

// CertificateInfo defines the attributes required to define an instance of a
// certificate to be installed via the system API.  The structure of the
// system API is not uniform for all certificate types therefore some attention
// is required when defining these resources.
type CertificateInfo struct {
	// Type represents the intended usage of the certificate
	// +kubebuilder:validation:Enum=ssl,ssl_ca,openstack,openstack_ca,docker_registry,tpm_mode
	Type string `json:"type"`

	// Secret is the name of a TLS secret containing the public certificate and
	// private key.  The secret must be of type kubernetes.io/tls and must
	// contain specific data attributes.  Specifically, all secrets must, at a
	// minimum contain the "tls.crt" key since all certificates will at least
	// require public certificate PEM data.  The remaining two keys "tls.key"
	// and "ca.crt" are optional depending on the certificate type. For the
	// "platform", "openstack", "tpm", and "docker" certificate types both the
	// "tls.crt" and "tls.key" certificates are needed while for the "*_ca"
	// version of those same certificate types only the "tls.crt" attribute is
	// required.  The "ca.crt" attribute is only required for the "platform" or
	// "tpm" certificate types, and only if the supplied public certificate is
	// signed by a non-standard root CA.
	Secret string `json:"secret"`
}

// DeepEqual overrides the code generated DeepEqual method because the
// credential information built from the running configuration never includes
// enough information to rebuild the certificate (i.e., the private key is not
// returned at the API) so when the profile is created dynamically it can only
// point to a Secret named by the system.
func (in *CertificateInfo) DeepEqual(other *CertificateInfo) bool {
	if other != nil {
		return in.Type == other.Type
	}

	return false
}

// PrivateKeyExpected determines whether a certificate requires a private key
// to be supplied to the system API.
func (in *CertificateInfo) PrivateKeyExpected() bool {
	// The two CA type certificate exist purely to add a known CA/root
	// certificate to the system and do not require a private key.
	return in.Type != PlatformCACertificate && in.Type != OpenstackCACertificate
}

// CertificateList defines a type to represent a slice of certificate info
// objects.
// +deepequal-gen:unordered-array=true
type CertificateList []CertificateInfo

// LicenseInfo defines the attributes which specify an individual License
// resource.
type LicenseInfo struct {
	// Secret is the name of a TLS secret containing the license file contents.
	// It must refer to a Opaque Kubernetes Secret.
	Secret string `json:"secret"`
}

// DeepEqual overrides the code generated DeepEqual method because the License
// information is stored in a Secret and we cannot compare it easily since it
// is not directly a part of the SystemSpec.
func (in *LicenseInfo) DeepEqual(other *LicenseInfo) bool {
	return other != nil
}

// ServiceParameterInfo defines the attributes required to define an instance of a
// service parameter to be installed via the system API.
type ServiceParameterInfo struct {
	// Service identifies the service for this service parameter
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9\-_]+$
	// +kubebuilder:validation:MaxLength=16
	Service string `json:"service"`

	// Section identifies the section for this service parameter
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9\-_]+$
	// +kubebuilder:validation:MaxLength=128
	Section string `json:"section"`

	// ParamName identifies the name for this service parameter
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9\-_]+$
	// +kubebuilder:validation:MaxLength=255
	ParamName string `json:"paramname"`

	// ParamValue identifies the value for this service parameter
	// +kubebuilder:validation:MaxLength=4096
	ParamValue string `json:"paramvalue"`

	// Personality identifies the personality for this service parameter
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Personality *string `json:"personality,omitempty"`

	// Resource identifies the resource for this service parameter
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Resource *string `json:"resource,omitempty"`
}

// ServiceParameterList defines a type to represent a slice of service parameter info
// objects.
// +deepequal-gen:unordered-array=true
type ServiceParameterList []ServiceParameterInfo

// +deepequal-gen:ignore-nil-fields=true
type StorageBackend struct {
	// SystemName uniquely identifies the storage backend instance.
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9\-_]+$
	// +kubebuilder:validation:MaxLength=255
	Name string `json:"name"`

	// Type specifies the storage backend type.
	// +kubebuilder:validation:Enum=file,lvm,ceph
	Type string `json:"type"`

	// Services is a list of services to enable for this backend instance.  Each
	// backend type supports a limited set
	// of services.  Refer to customer documentation for more information.
	// +kubebuilder:validation:Enum=cinder,glance,nova,swift,rbd-provisioner
	Services []string `json:"services,omitempty"`

	// ReplicationFactor is the number of storage hosts required in each
	// replication group for storage redundancy.
	// This attribute is only applicable for Ceph storage backends.
	// +kubebuilder:validation:Minimum=2
	// +kubebuilder:validation:Maximum=3
	// +kubebuilder:validation:ExclusiveMinimum=false
	// +kubebuilder:validation:ExclusiveMaximum=false
	// +optional
	ReplicationFactor *int `json:"replicationFactor,omitempty"`

	// PartitionSize is the controller disk partition size to be allocated for
	// the Ceph monitor - in gigabytes.
	// This attribute is only applicable for Ceph storage backends.
	// +kubebuilder:validation:Minimum=20
	// +kubebuilder:validation:ExclusiveMinimum=false
	// +optional
	PartitionSize *int `json:"partitionSize,omitempty"`
}

// DRBDConfiguration defines the DRBD file system settings for the system.
type DRBDConfiguration struct {
	// LinkUtilization defines the maximum link utilisation percentage during
	// sync activities.
	// +kubebuilder:validation:Minimum=20
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:validation:ExclusiveMinimum=false
	// +kubebuilder:validation:ExclusiveMaximum=false
	LinkUtilization int `json:"linkUtilization"`
}

// StorageBackendList defines a type to represent a slice of storage backends.
// +deepequal-gen:unordered-array=true
type StorageBackendList []StorageBackend

// ControllerFileSystemInfo defines the attributes of a single controller
// filesystem resource.
type ControllerFileSystemInfo struct {
	// Name defines the system defined name of the filesystem resource.
	// +kubebuilder:validation:Enum=cgcs,platform,cinder,database,docker-distribution,etcd,extension,patch-vault
	Name string `json:"name"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:ExclusiveMinimum=false
	Size int `json:"size"`
}

// ControllerFileSystemList defines a type to represent a slice of controller filesystem
// resources.
// +deepequal-gen:unordered-array=true
type ControllerFileSystemList []ControllerFileSystemInfo

// SystemStorageInfo defines the system level storage attributes that are
// configurable.
// +deepequal-gen:ignore-nil-fields=true
type SystemStorageInfo struct {
	// Backends is a set of backend storage methods to be configured.  Only
	Backends *StorageBackendList `json:"backends,omitempty"`

	// DRBD defines the set of DRBD configuration attributes for the system.
	DRBD *DRBDConfiguration `json:"drbd,omitempty"`

	// Filesystems defines the set of controller file system definitions.
	FileSystems *ControllerFileSystemList `json:"filesystems,omitempty"`
}

// PTPInfo defines the system level precision time protocol attributes that are
// configurable.
// +deepequal-gen:ignore-nil-fields=true
type PTPInfo struct {
	// Mode defines the precision time protocol mode of the system.
	// +kubebuilder:validation:Enum=hardware,software,legacy
	// +optional
	Mode *string `json:"mode,omitempty"`

	// Transport defines the network transport protocol used to implement the
	// precision time protocol.
	// +kubebuilder:validation:Enum=l2,udp
	// +optional
	Transport *string `json:"transport,omitempty"`

	// Mechanism defines the high level messaging architecture used to implement
	// the precision time procotol.
	// +kubebuilder:validation:Enum=p2p,e2e
	// +optional
	Mechanism *string `json:"mechanism,omitempty"`
}

// TrapDestInfo defines the SNMP community information related to a single
// SNMP trap destination.
type TrapDestInfo struct {
	Community string `json:"community"`
	IPAddress string `json:"address"`
}

// TrapDestList represents a list of SNMP trap destinations.
// +deepequal-gen:unordered-array=true
type TrapDestList []TrapDestInfo

// SNMPInfo defines the system level SNMP attributes that are configurable.
// +deepequal-gen:ignore-nil-fields=true
type SNMPInfo struct {
	// Communities defines the list of SNMP communities to be configured.
	Communities *StringList `json:"communities,omitempty"`

	// TrapDestinations defines the list of SNMP Trap Destinations to be
	// configured.
	TrapDestinations *TrapDestList `json:"trapDestinations,omitempty"`
}

// SystemSpec defines the desired state of System
// +deepequal-gen:ignore-nil-fields=true
type SystemSpec struct {
	// Description is a free form string describing the intended purpose of the
	// system.
	// +optional
	Description *string `json:"description,omitempty"`

	// Location is a short description of the system's physical location.
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9\-_\. ]+$
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Location *string `json:"location,omitempty"`

	// Contact is a method to reach the person responsible for the system.  For
	// example it could be an email address,
	// phone number, or physical address.
	// +kubebuilder:validation:Pattern=^[a-zA-Z0-9@\-_\. ]+$
	// +kubebuilder:validation:MaxLength=255
	// +optional
	Contact *string `json:"contact,omitempty"`

	// Nameservers is an array of Domain SystemName servers.  Each server can be
	// specified as either an IPv4 or IPv6
	// address.
	// +optional
	DNSServers *StringList `json:"dnsServers,omitempty"`

	// NTPServers is an array of Network Time Protocol servers.  Each server can
	// be specified as either an IPv4 or IPv6
	// address, or a FQDN hostname.
	// +optional
	NTPServers *StringList `json:"ntpServers,omitempty"`

	// PTP defines the Precision Time Protocol configuration for the system.
	PTP *PTPInfo `json:"ptp,omitempty"`

	// Certificates is a list of references to certificates that must be
	// installed.
	// +optional
	Certificates *CertificateList `json:"certificates,omitempty"`

	// License is a reference to a license file that must be installed.
	// +optional
	License *LicenseInfo `json:"license,omitempty"`

	// ServiceParameters is a list of service parameters
	// +optional
	ServiceParameters *ServiceParameterList `json:"serviceParameters,omitempty"`

	// Storage is a set of storage specific attributes to be configured for the
	// system.
	// +optional
	Storage *SystemStorageInfo `json:"storage,omitempty"`

	// SNMP is the set of SNMP specific attributes to be configured for the
	// system.
	// +optional
	SNMP *SNMPInfo `json:"snmp,omitempty"`

	// VSwitchType is the desired vswitch implementation to be configured. This
	// is intentionally left unvalidated to avoid issues with proprietary
	// vswitch implementation.
	// +optional
	VSwitchType *string `json:"vswitchType,omitempty"`
}

// IsKeyEqual compares two controller file system array elements and determines
// if they refer to the same instance.  All other attributes will be merged
// during profile merging.
func (in ControllerFileSystemInfo) IsKeyEqual(x ControllerFileSystemInfo) bool {
	return in.Name == x.Name
}

// IsKeyEqual compares two SNMP trap destination array elements and determines
// if they refer to the same instance.  All other attributes will be merged
// during profile merging.
func (in TrapDestInfo) IsKeyEqual(x TrapDestInfo) bool {
	return in.Community == x.Community
}

// SystemStatus defines the observed state of System
type SystemStatus struct {
	// ID defines the unique identifier assigned by the system.
	ID string `json:"id"`

	// SystemType defines the current system type reported by the system API.
	SystemType string `json:"systemType"`

	// SystemMode defines the current system mode reported by the system API.
	SystemMode string `json:"systemMode"`

	// SoftwareVersion defines the current software version reported by the
	// system API.
	SoftwareVersion string `json:"softwareVersion"`

	// Defines whether the resource has been provisioned on the target system.
	InSync bool `json:"inSync"`

	// Reconciled defines whether the System has been successfully reconciled
	// at least once.  If further changes are made they will be ignored by the
	// reconciler.
	Reconciled bool `json:"reconciled"`

	// Defaults defines the configuration attributed collected before applying
	// any user configuration values.
	// +optional
	Defaults *string `json:"defaults,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// System defines the attributes that represent the system level attributes
// of a StarlingX system.  This is a composition of the following StarlingX
// API endpoints.
//
//   https://docs.starlingx.io/api-ref/stx-config/api-ref-sysinv-v1-config.html#system
//   https://docs.starlingx.io/api-ref/stx-config/api-ref-sysinv-v1-config.html#dns
//   https://docs.starlingx.io/api-ref/stx-config/api-ref-sysinv-v1-config.html#ntp
//   https://docs.starlingx.io/api-ref/stx-config/api-ref-sysinv-v1-config.html#system-certificate-configuration
//   https://docs.starlingx.io/api-ref/stx-config/api-ref-sysinv-v1-config.html#storage-backends
//
// +k8s:openapi-gen=true
// +deepequal-gen=false
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="mode",type="string",JSONPath=".status.systemMode",description="The configured system mode."
// +kubebuilder:printcolumn:name="type",type="string",JSONPath=".status.systemType",description="The configured system type."
// +kubebuilder:printcolumn:name="version",type="string",JSONPath=".status.softwareVersion",description="The current software version"
// +kubebuilder:printcolumn:name="insync",type="boolean",JSONPath=".status.inSync",description="The current synchronization state."
type System struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SystemSpec   `json:"spec,omitempty"`
	Status SystemStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SystemList contains a list of System
// +deepequal-gen=false
type SystemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []System `json:"items"`
}

func init() {
	SchemeBuilder.Register(&System{}, &SystemList{})
}

// HTTPSEnabled determine whether HTTPS needs to be enabled.  Rather than model
// this attribute explicitly we determine the result dynamically.
func (in *System) HTTPSEnabled() bool {
	if in.Spec.Certificates != nil {
		for _, c := range *in.Spec.Certificates {
			if c.Type == PlatformCertificate {
				return true
			}
		}
	}

	return false
}
