/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2023-2024 Wind River Systems, Inc. */

package build

import (
	"reflect"
	"strings"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/cpus"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaces"
	v1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test filters utilities:", func() {
	Describe("Test ServiceParameterFilter", func() {
		system := v1.System{}
		deployment := Deployment{}
		Context("If default service parameter fileter", func() {
			It("defaults should be filtered", func() {
				list := make([]v1.ServiceParameterInfo, 0)
				for _, param := range utils.DefaultParameters {
					sp := v1.ServiceParameterInfo{
						Service:   param.Service,
						Section:   param.Section,
						ParamName: param.ParamName,
					}
					list = append(list, sp)
				}
				sp := v1.ServiceParameterInfo{
					Service:   "foo",
					Section:   "bar",
					ParamName: "foobar",
				}
				list = append(list, sp)
				spList := v1.ServiceParameterList(list)
				system.Spec.ServiceParameters = &spList

				spFilter := ServiceParameterFilter{}
				err := spFilter.Filter(&system, &deployment)
				Expect(err).To(BeNil())
				got := system.Spec.ServiceParameters
				expectSpArrary := make([]v1.ServiceParameterInfo, 0)
				expectSpArrary = append(expectSpArrary, sp)
				expectSpList := v1.ServiceParameterList(expectSpArrary)
				Expect(reflect.DeepEqual(got, &expectSpList)).To(BeTrue())
			})
		})

		Context("If no service parameter filter", func() {
			It("all should be filtered", func() {
				list := make([]v1.ServiceParameterInfo, 0)
				for _, param := range utils.DefaultParameters {
					sp := v1.ServiceParameterInfo{
						Service:   param.Service,
						Section:   param.Section,
						ParamName: param.ParamName,
					}
					list = append(list, sp)
				}
				sp := v1.ServiceParameterInfo{
					Service:   "foo",
					Section:   "bar",
					ParamName: "foobar",
				}
				list = append(list, sp)
				spList := v1.ServiceParameterList(list)
				system.Spec.ServiceParameters = &spList

				//the default service parameter filter should be always applied
				spFilter := ServiceParameterFilter{}
				err := spFilter.Filter(&system, &deployment)
				Expect(err).To(BeNil())
				noSpFilter := NoServiceParameterFilter{}
				err = noSpFilter.Filter(&system, &deployment)
				Expect(err).To(BeNil())
				got := system.Spec.ServiceParameters
				Expect(got).To(BeNil())
			})
		})
	})

	Describe("Test new service parameters", func() {
		Context("when default service parameters filter", func() {
			It("should return an ServiceParameterFilter", func() {
				got := NewServiceParametersSystemFilter()
				expect := ServiceParameterFilter{}
				Expect(reflect.DeepEqual(got, &expect)).To(BeTrue())
			})
		})

		Context("when no service parameters filter", func() {
			It("should return an NoServiceParameterFilter", func() {
				got := NewNoServiceParametersSystemFilter()
				expect := NoServiceParameterFilter{}
				Expect(reflect.DeepEqual(got, &expect)).To(BeTrue())
			})
		})
	})

	Describe("Test the HostKernelFilter on different types of hosts", func() {
		const (
			aio_index              = 0 // all-in-one controller
			ctrl_index             = 1 // standard controller
			wrk_index              = 2 // worker
			str_index              = 3 // storage
			number_of_tested_hosts = 4 // number of hosts
		)

		kernelfilter := NewHostKernelFilter()
		deployment := Deployment{}

		hosts := make([]v1.Host, number_of_tested_hosts)
		profiles := make([]v1.HostProfile, number_of_tested_hosts)
		deployment.Hosts = make([]*v1.Host, number_of_tested_hosts)
		deployment.Profiles = make([]*v1.HostProfile, number_of_tested_hosts)
		for i := 0; i < number_of_tested_hosts; i++ {
			deployment.Hosts[i] = &hosts[i]
			deployment.Profiles[i] = &profiles[i]
		}

		var populatespec = func(hostinfo v1info.HostInfo,
			profile *v1.HostProfile, host *v1.Host) {

			host.Spec.Profile = hostinfo.Hostname
			profile.Spec.Personality = &hostinfo.Personality
			profile.Spec.Kernel = &hostinfo.Kernel.ProvisionedKernel
			sf := strings.Split(hostinfo.SubFunctions, ",")
			profile.Spec.SubFunctions = make([]v1.SubFunction, len(sf))
			for i := 0; i < len(sf); i++ {
				profile.Spec.SubFunctions[i] = v1.SubFunctionFromString(sf[i])
			}
		}

		Context("when host is an all-in-one controller node", func() {

			host := deployment.Hosts[aio_index]
			profile := deployment.Profiles[aio_index]
			hostinfo := v1info.HostInfo{}

			hostinfo.Hostname = "controller-0"
			hostinfo.Personality = "controller"
			hostinfo.SubFunctions = "controller,worker,lowlatency"
			hostinfo.Kernel.Hostname = "controller-0"
			hostinfo.Kernel.RunningKernel = "lowlatency"
			hostinfo.Kernel.ProvisionedKernel = "lowlatency"

			populatespec(hostinfo, profile, host)
			It("should not remove the kernel parameter from the spec", func() {
				err := kernelfilter.Filter(profile, host, &deployment)
				Expect(err).To(BeNil())
				Expect(profile.Spec.Kernel).NotTo(BeNil(),
					"Kernel parameter should not be Nil")
			})
		})

		Context("when host is a standard controller node", func() {
			host := deployment.Hosts[ctrl_index]
			profile := deployment.Profiles[ctrl_index]
			hostinfo := v1info.HostInfo{}

			hostinfo.Hostname = "controller-1"
			hostinfo.Personality = "controller"
			hostinfo.SubFunctions = "controller"
			hostinfo.Kernel.Hostname = "controller-1"
			hostinfo.Kernel.RunningKernel = "standard"
			hostinfo.Kernel.ProvisionedKernel = "standard"

			populatespec(hostinfo, profile, host)

			It("should remove the kernel parameter from the spec", func() {
				err := kernelfilter.Filter(profile, host, &deployment)
				Expect(err).To(BeNil())
				Expect(profile.Spec.Kernel).To(BeNil(),
					"Kernel parameter should be Nil")
			})
		})

		Context("when host is a worker node", func() {
			host := deployment.Hosts[wrk_index]
			profile := deployment.Profiles[wrk_index]
			hostinfo := v1info.HostInfo{}

			hostinfo.Hostname = "worker-0"
			hostinfo.Personality = "worker"
			hostinfo.SubFunctions = "worker,lowlatency"
			hostinfo.Kernel.Hostname = "worker-0"
			hostinfo.Kernel.RunningKernel = "lowlatency"
			hostinfo.Kernel.ProvisionedKernel = "lowlatency"

			populatespec(hostinfo, profile, host)

			It("should not remove the kernel parameter from the spec", func() {
				err := kernelfilter.Filter(profile, host, &deployment)
				Expect(err).To(BeNil())
				Expect(profile.Spec.Kernel).NotTo(BeNil(),
					"Kernel parameter should not be Nil")
			})
		})

		Context("when host is a dedicated storage node", func() {
			host := deployment.Hosts[str_index]
			profile := deployment.Profiles[str_index]
			hostinfo := v1info.HostInfo{}

			hostinfo.Hostname = "storage-0"
			hostinfo.Personality = "storage"
			hostinfo.SubFunctions = "storage"
			hostinfo.Kernel.Hostname = "storage-0"
			hostinfo.Kernel.RunningKernel = "standard"
			hostinfo.Kernel.ProvisionedKernel = "standard"

			populatespec(hostinfo, profile, host)

			It("should remove the kernel parameter from the spec", func() {
				err := kernelfilter.Filter(profile, host, &deployment)
				Expect(err).To(BeNil())
				Expect(profile.Spec.Kernel).To(BeNil(),
					"Kernel parameter should be Nil")
			})
		})

	})

	Describe("Test platform network filters", func() {
		deployment := Deployment{}
		deployment.PlatformNetworks = make([]*v1.PlatformNetwork, 0)
		deployment.AddressPools = make([]*v1.AddressPool, 0)
		coreNetworkFilter := NewCoreNetworkFilter()
		var get_platform_network = func(nwk_type string, associatedAddressPools []string) *v1.PlatformNetwork {
			spec := v1.PlatformNetworkSpec{
				Type:                   nwk_type,
				AssociatedAddressPools: associatedAddressPools,
			}
			new_plat_nwk := v1.PlatformNetwork{}
			new_plat_nwk.Spec = spec
			return &new_plat_nwk
		}
		var get_associated_address_pool = func(pool_name string) *v1.AddressPool {
			spec := v1.AddressPoolSpec{
				Subnet: "192.168.11.32",
			}
			new_address_pool := v1.AddressPool{}
			new_address_pool.Name = pool_name
			new_address_pool.Spec = spec
			return &new_address_pool
		}
		network_types := []string{"oam", "mgmt", "admin", "storage"}
		ip_families := []string{"ipv4", "ipv6"}
		for _, net_type := range network_types {
			associated_pools := []string{net_type + "-ipv4", net_type + "-ipv6"}
			deployment.PlatformNetworks = append(deployment.PlatformNetworks, get_platform_network(net_type, associated_pools))
			for _, ip_family := range ip_families {
				deployment.AddressPools = append(deployment.AddressPools, get_associated_address_pool(net_type+"-"+ip_family))
			}
		}
		Context("when new core network filter", func() {
			It("should return a CoreNetworkFilter", func() {
				got := coreNetworkFilter
				expect := CoreNetworkFilter{}
				Expect(reflect.DeepEqual(got, &expect)).To(BeTrue())
			})
		})

		Context("When core network filter is applied", func() {
			It("deletes only oam/mgmt/admin platform networks", func() {
				err := coreNetworkFilter.Filter(deployment.PlatformNetworks[0], &deployment)
				Expect(err).To(BeNil())
				Expect(len(deployment.PlatformNetworks)).To(Equal(1), "CoreNetworkFilter should not delete any platform networks other than oam/mgmt/admin")
				Expect(len(deployment.AddressPools)).To(Equal(2))
			})
		})

	})

	Describe("Test  fileSystemFilter", func() {
		Context("When there is extra tye fs present", func() {
			It("filters out the extra fs", func() {
				filter := &FileSystemFilter{}

				// Create a test case with sample input
				system := &v1.System{
					Spec: v1.SystemSpec{
						Storage: &v1.SystemStorageInfo{
							FileSystems: &v1.ControllerFileSystemList{
								{ // Adding file systems for testing
									Name: "backup",
									Size: 100,
								},
								{
									Name: "database",
									Size: 200,
								},
								{
									Name: "instances",
									Size: 300,
								},
								{
									Name: "image-conversion",
									Size: 400,
								},
								{
									Name: "extra", // this will be filtered out
									Size: 500,
								},
							},
						},
					},
				}

				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type

				// Call the Filter method
				err := filter.Filter(system, deployment)
				Expect(err).To(BeNil())

				// Check if file systems have been filtered as expected
				expectedFilteredFileSystems := []v1.ControllerFileSystemInfo{
					{
						Name: "backup",
						Size: 100,
					},
					{
						Name: "database",
						Size: 200,
					},
					{
						Name: "instances",
						Size: 300,
					},
					{
						Name: "image-conversion",
						Size: 400,
					},
				}
				list := v1.ControllerFileSystemList(expectedFilteredFileSystems)
				Expect(&list).To(Equal(system.Spec.Storage.FileSystems))

			})
		})
	})

	Describe("Test  CACertificateFilter", func() {
		Context("When there is ssl_ca/openstack_ca/docker_registry/ssl/openldap type certificates are also present", func() {
			It("filters out the ssl_ca/openstack_ca/docker_registry/ssl/openldap type certificates", func() {
				filter := &CACertificateFilter{}

				// Create a test case with sample input
				system := &v1.System{
					Spec: v1.SystemSpec{
						Certificates: &v1.CertificateList{

							{ // Adding certificate info for testing
								Type: v1.PlatformCACertificate, // this will be filtered out
							},
							{
								Type: v1.OpenstackCACertificate, // this will be filtered out
							},
							{
								Type: v1.PlatformCertificate, // this will be filtered out
							},
							{
								Type: v1.DockerCertificate, // this will be filtered out
							},
							{
								Type: v1.OpenLDAPCertificate, // this will be filtered out
							},
							{
								Type: v1.TPMCertificate,
							},
						},
					},
				}

				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type

				// Call the Filter method
				err := filter.Filter(system, deployment)
				Expect(err).To(BeNil())

				// Check if file systems have been filtered as expected
				expectedFilteredCertificates := []v1.CertificateInfo{
					{
						Type: v1.TPMCertificate,
					},
				}
				list := v1.CertificateList(expectedFilteredCertificates)
				Expect(&list).To(Equal(system.Spec.Certificates))

			})
		})
	})

	Describe("Test ServiceParameterFilter", func() {
		Context("When spec has default service parameters", func() {
			It("It excludes default service parameters", func() {
				filter := &ServiceParameterFilter{}

				// Create a test case with sample input
				system := &v1.System{
					Spec: v1.SystemSpec{
						ServiceParameters: &v1.ServiceParameterList{

							{ // Adding certificate info for testing
								Service:   utils.ServiceTypeIdentity,
								Section:   utils.ServiceParamSectionIdentityConfig,
								ParamName: utils.ServiceParamIdentityConfigTokenExpiration,
							},
							{
								Service:   utils.ServiceTypePlatform,
								Section:   utils.ServiceParamSectionPlatformMaintenance,
								ParamName: utils.ServiceParamPlatMtceWorkerBootTimeout,
							},
							{
								// this will be filtered out
								Service:   "service",
								Section:   "extra",
								ParamName: "fake",
							},
						},
					},
				}

				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type

				// Call the Filter method
				err := filter.Filter(system, deployment)
				Expect(err).To(BeNil())

				// Check if file systems have been filtered as expected
				expectedFilteredServiceParams := []v1.ServiceParameterInfo{
					{
						Service:   "service",
						Section:   "extra",
						ParamName: "fake",
					},
				}
				list := v1.ServiceParameterList(expectedFilteredServiceParams)
				Expect(list).To(Equal(*system.Spec.ServiceParameters))
			})
		})
	})

	Describe("Test InterfaceRemoveUuidFilter", func() {
		Context("When there exists Uuids in interface", func() {
			It("Removes Uuid from interfaces", func() {
				filter := &InterfaceRemoveUuidFilter{}

				ethIn := v1.CommonInterfaceInfo{
					UUID: "ethIn",
					Name: "name",
				}
				bondsIn := v1.CommonInterfaceInfo{
					UUID: "bondsIn",
					Name: "name",
				}
				vlansIn := v1.CommonInterfaceInfo{
					UUID: "vlansIn",
					Name: "name",
				}
				vfsIn := v1.CommonInterfaceInfo{
					UUID: "vfsIn",
					Name: "name",
				}
				hEthIn := v1.CommonInterfaceInfo{
					UUID: "ethIn",
					Name: "name",
				}
				hBondsIn := v1.CommonInterfaceInfo{
					UUID: "bondsIn",
					Name: "name",
				}
				hVlansIn := v1.CommonInterfaceInfo{
					UUID: "vlansIn",
					Name: "name",
				}
				hVfsIn := v1.CommonInterfaceInfo{
					UUID: "vfsIn",
					Name: "name",
				}

				// Create a test case with sample input
				h := &v1.Host{
					Spec: v1.HostSpec{
						Overrides: &v1.HostProfileSpec{
							Interfaces: &v1.InterfaceInfo{
								Ethernet: v1.EthernetList{
									{
										CommonInterfaceInfo: hEthIn,
									},
								},
								VLAN: v1.VLANList{
									{
										CommonInterfaceInfo: hVlansIn,
									},
								},
								Bond: v1.BondList{
									{
										CommonInterfaceInfo: hBondsIn,
									},
								},
								VF: v1.VFList{
									{
										CommonInterfaceInfo: hVfsIn,
									},
								},
							},
						},
					},
				}
				hp := &v1.HostProfile{
					Spec: v1.HostProfileSpec{
						Interfaces: &v1.InterfaceInfo{
							Ethernet: v1.EthernetList{
								{
									CommonInterfaceInfo: ethIn,
								},
							},
							VLAN: v1.VLANList{
								{
									CommonInterfaceInfo: vlansIn,
								},
							},
							Bond: v1.BondList{
								{
									CommonInterfaceInfo: bondsIn,
								},
							},
							VF: v1.VFList{
								{
									CommonInterfaceInfo: vfsIn,
								},
							},
						},
					},
				}

				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type

				// Call the Filter method
				err := filter.Filter(hp, h, deployment)
				Expect(err).To(BeNil())

				// Check if UUIDs  have been filtered as expected
				for _, in := range hp.Spec.Interfaces.Ethernet {
					Expect(in.CommonInterfaceInfo.UUID).To(Equal(""))
				}
				for _, in := range hp.Spec.Interfaces.VLAN {
					Expect(in.CommonInterfaceInfo.UUID).To(Equal(""))
				}
				for _, in := range hp.Spec.Interfaces.VF {
					Expect(in.CommonInterfaceInfo.UUID).To(Equal(""))
				}
				for _, in := range hp.Spec.Interfaces.Bond {
					Expect(in.CommonInterfaceInfo.UUID).To(Equal(""))
				}
				for _, in := range h.Spec.Overrides.Interfaces.Ethernet {
					Expect(in.CommonInterfaceInfo.UUID).To(Equal(""))
				}
				for _, in := range h.Spec.Overrides.Interfaces.VLAN {
					Expect(in.CommonInterfaceInfo.UUID).To(Equal(""))
				}
				for _, in := range h.Spec.Overrides.Interfaces.VF {
					Expect(in.CommonInterfaceInfo.UUID).To(Equal(""))
				}
				for _, in := range h.Spec.Overrides.Interfaces.Bond {
					Expect(in.CommonInterfaceInfo.UUID).To(Equal(""))
				}
			})
		})
	})

	Describe("Test HostKernelFilter", func() {
		Context("When the host has worker node", func() {
			It("Should not filter kernel parameter", func() {
				filter := &HostKernelFilter{}
				kernel := "kernel"
				// Create a test case with sample input
				worker := hosts.PersonalityWorker
				spec := v1.HostProfileSpec{
					ProfileBaseAttributes: v1.ProfileBaseAttributes{
						Personality: &worker,
						Kernel:      &kernel,
					},
				}
				hp := &v1.HostProfile{
					Spec: spec,
				}
				h := &v1.Host{}
				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type
				// Call the Filter method
				err := filter.Filter(hp, h, deployment)
				Expect(err).To(BeNil())
				// Check if kernel is as expected
				Expect(hp.Spec.Kernel).To(Equal(&kernel))
			})
		})

		Context("When the host has storage node", func() {
			It("Should not filter kernel parameter", func() {
				filter := &HostKernelFilter{}
				kernel := "kernel"
				// Create a test case with sample input
				controller := hosts.PersonalityController
				spec := v1.HostProfileSpec{
					ProfileBaseAttributes: v1.ProfileBaseAttributes{
						Personality: &controller,
						Kernel:      &kernel,
					},
				}
				hp := &v1.HostProfile{
					Spec: spec,
				}
				h := &v1.Host{}
				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type
				// Call the Filter method
				err := filter.Filter(hp, h, deployment)
				Expect(err).To(BeNil())
				// Check if kernel have been filtered as expected
				var nilKernel *string = nil
				Expect(hp.Spec.Kernel).To(Equal(nilKernel))
			})
		})
	})
	Describe("Test Filter of Controller0", func() {
		Context("When its controller-0", func() {
			It("Filter from overrides", func() {
				filter := &Controller0Filter{}

				// Create a test case with sample input
				dynamic := v1.ProvioningModeDynamic
				static := v1.ProvioningModeStatic
				bootMac := "01:02:03:04:05:06"
				hp := &v1.HostProfile{}
				h := &v1.Host{
					Spec: v1.HostSpec{
						Overrides: &v1.HostProfileSpec{
							ProfileBaseAttributes: v1.ProfileBaseAttributes{
								BootMAC:          &bootMac,
								ProvisioningMode: &static,
							},
						},
						Match: &v1.MatchInfo{},
					},
				}
				h.Name = hosts.Controller0
				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type

				// Call the Filter method
				err := filter.Filter(hp, h, deployment)
				Expect(err).To(BeNil())
				var nilAddr *string = nil
				// Check if host BootMAC have been filtered as expected from overrides and added to the matchInfo
				Expect(h.Spec.Overrides.ProvisioningMode).To(Equal(&dynamic))
				Expect(h.Spec.Match.BootMAC).To(Equal(&bootMac))
				Expect(h.Spec.Overrides.BootMAC).To(Equal(nilAddr))
			})
		})
	})

	Describe("Test Location Filter", func() {
		Context("WHen the location is not nil", func() {
			It("filters out location from host spec to overrrides", func() {
				filter := &LocationFilter{}

				// Create a test case with sample input
				location := "A sample location"
				hLoc := "host location"
				hp := &v1.HostProfile{
					Spec: v1.HostProfileSpec{
						ProfileBaseAttributes: v1.ProfileBaseAttributes{
							Location: &location,
						},
					},
				}
				h := &v1.Host{
					Spec: v1.HostSpec{
						Overrides: &v1.HostProfileSpec{
							ProfileBaseAttributes: v1.ProfileBaseAttributes{
								Location: &hLoc,
							},
						},
					},
				}
				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type

				// Call the Filter method
				err := filter.Filter(hp, h, deployment)
				Expect(err).To(BeNil())
				var nilAddr *string = nil
				// Check if host BootMAC have been filtered as expected from overrides and added to the matchInfo
				Expect(h.Spec.Overrides.Location).To(Equal(&location))
				Expect(hp.Spec.Location).To(Equal(nilAddr))
			})
		})
	})
	Describe("Test AddressFilter", func() {
		Context("When the profile address exists", func() {
			It("FIlters address from host spec to host overrides", func() {
				filter := &AddressFilter{}

				// Create a test case with sample input
				profAddr := "profile address"
				hp := &v1.HostProfile{
					Spec: v1.HostProfileSpec{
						Addresses: v1.AddressList{
							{
								Address: profAddr,
							},
						},
					},
				}
				h := &v1.Host{
					Spec: v1.HostSpec{
						Overrides: &v1.HostProfileSpec{},
					},
				}
				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type

				// Call the Filter method
				err := filter.Filter(hp, h, deployment)
				Expect(err).To(BeNil())

				// Check if host BootMAC have been filtered as expected from overrides and added to the matchInfo
				var nilAddr v1.AddressList = nil
				Expect(h.Spec.Overrides.Addresses[0].Address).To(Equal(profAddr))
				Expect(hp.Spec.Addresses).To(Equal(nilAddr))
			})
		})
	})

	Describe("Test BMAddressFilter", func() {
		Context("When  BoardManagement and Adress is not nil", func() {
			It("filters BMAddress", func() {
				filter := &BMAddressFilter{}
				// Create a test case with sample input
				profileAddr := "profile address"

				hp := &v1.HostProfile{
					Spec: v1.HostProfileSpec{
						BoardManagement: &v1.BMInfo{
							Address: &profileAddr,
						},
					},
				}
				h := &v1.Host{
					Spec: v1.HostSpec{
						Overrides: &v1.HostProfileSpec{},
					},
				}
				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type

				// Call the Filter method
				err := filter.Filter(hp, h, deployment)
				Expect(err).To(BeNil())

				// Check if host BroadManagement have been filtered as expected from overrides and added to the profile
				var nilAddr *string = nil
				Expect(*h.Spec.Overrides.BoardManagement.Address).To(Equal(profileAddr))
				Expect(hp.Spec.BoardManagement.Address).To(Equal(nilAddr))
			})
		})
	})

	Describe("Test StorageMonitorFilter", func() {
		Context("When volumeGrps,OSDS,Fs are  nil", func() {
			It("filters profile spec storage", func() {
				filter := &StorageMonitorFilter{}
				// Create a test case with sample input
				size := 1
				mInfo := v1.MonitorInfo{
					Size: &size,
				}
				hp := &v1.HostProfile{
					Spec: v1.HostProfileSpec{
						Storage: &v1.ProfileStorageInfo{
							Monitor:      &mInfo,
							VolumeGroups: nil,
							OSDs:         nil,
							FileSystems:  nil,
						},
					},
				}

				h := &v1.Host{
					Spec: v1.HostSpec{
						Overrides: &v1.HostProfileSpec{},
					},
				}
				h.Spec.Overrides.Storage = nil
				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type

				// Call the Filter method
				err := filter.Filter(hp, h, deployment)
				Expect(err).To(BeNil())

				// Check if host BroadManagement have been filtered as expected from overrides and added to the profile
				var nilProfInfo *v1.ProfileStorageInfo = nil
				profStorageInfo := &v1.ProfileStorageInfo{
					Monitor: &mInfo,
				}
				Expect(h.Spec.Overrides.Storage).To(Equal(profStorageInfo))
				Expect(hp.Spec.Storage).To(Equal(nilProfInfo))
			})
		})
	})

	Describe("Test StorageMonitorFilter", func() {
		Context("When volumeGrps,OSDS,Fs are not nil", func() {
			It("doesnt filters profile spec storage ", func() {
				filter := &StorageMonitorFilter{}
				// Create a test case with sample input
				size := 1
				mInfo := v1.MonitorInfo{
					Size: &size,
				}
				hpSpec := &v1.ProfileStorageInfo{
					VolumeGroups: &v1.VolumeGroupList{},
					Monitor:      &mInfo,
					OSDs:         &v1.OSDList{},
					FileSystems:  &v1.FileSystemList{},
				}
				hp := &v1.HostProfile{
					Spec: v1.HostProfileSpec{
						Storage: hpSpec,
					},
				}

				h := &v1.Host{
					Spec: v1.HostSpec{
						Overrides: &v1.HostProfileSpec{},
					},
				}
				h.Spec.Overrides.Storage = nil
				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type

				// Call the Filter method
				err := filter.Filter(hp, h, deployment)
				Expect(err).To(BeNil())

				// Check if host BroadManagement have been filtered as expected from overrides and added to the profile
				var nilMonInfo *v1.MonitorInfo = nil
				profStorageInfo := &v1.ProfileStorageInfo{
					Monitor: &mInfo,
				}
				Expect(h.Spec.Overrides.Storage).To(Equal(profStorageInfo))
				Expect(hp.Spec.Storage.Monitor).To(Equal(nilMonInfo))
				Expect(hp.Spec.Storage).To(Equal(hpSpec))
			})
		})
	})

	Describe("Test LoopbackInterfaceFilter", func() {
		Context("Whenthere is a loopback iterface also present", func() {
			It("Filters the loopback interface", func() {
				filter := &LoopbackInterfaceFilter{}
				// Create a test case with sample input

				ethIn := &v1.InterfaceInfo{
					Ethernet: v1.EthernetList{},
				}
				ethIn.Ethernet = make([]v1.EthernetInfo, 0)
				hp := &v1.HostProfile{
					Spec: v1.HostProfileSpec{
						Interfaces: ethIn,
					},
				}
				var ethLb, ethInf2 v1.EthernetInfo
				ethInf2.Name = "eth2"
				ethIn.Ethernet = append(ethIn.Ethernet, ethInf2)
				list := ethIn.Ethernet

				ethLb.Name = interfaces.LoopbackInterfaceName
				ethIn.Ethernet = append(ethIn.Ethernet, ethLb)
				h := &v1.Host{
					Spec: v1.HostSpec{
						Overrides: &v1.HostProfileSpec{},
					},
				}
				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type

				// Call the Filter method
				err := filter.Filter(hp, h, deployment)
				Expect(err).To(BeNil())

				lbList := make([]v1.EthernetInfo, 0)
				lbList = append(lbList, ethLb)
				lbLists := v1.EthernetList(lbList)
				Expect(h.Spec.Overrides.Interfaces.Ethernet).To(Equal(lbLists))
				Expect(hp.Spec.Interfaces.Ethernet).To(Equal(list))
			})
		})
	})

	//TBD: should try other cases for this func InterfaceUnusedFilter
	Describe("Test InterfaceUnusedFilter", func() {
		Context("When profile interfaces is used", func() {
			It("returns the same interface because of the absence of unused interfaces", func() {
				filter := &InterfaceUnusedFilter{}
				// Create a test case with sample input

				ethIn := &v1.InterfaceInfo{
					Ethernet: v1.EthernetList{},
				}
				ethIn.Ethernet = make([]v1.EthernetInfo, 0)
				hp := &v1.HostProfile{
					Spec: v1.HostProfileSpec{
						Interfaces: &v1.InterfaceInfo{
							Ethernet: v1.EthernetList{},
						},
					},
				}
				var ethInf v1.EthernetInfo
				ethInf.Name = "EthName"

				hp.Spec.Interfaces.Ethernet = append(hp.Spec.Interfaces.Ethernet, ethInf)

				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type

				// Call the Filter method
				err := filter.Filter(hp, deployment)
				Expect(err).To(BeNil())

				ethList := make([]v1.EthernetInfo, 0)
				ethList = append(ethList, ethInf)
				ethLists := v1.EthernetList(ethList)
				Expect(hp.Spec.Interfaces.Ethernet).To(Equal(ethLists))
			})
		})
	})
	Describe("Test MemoryDefaultsFilter", func() {
		Context("When Func is platform,pagecount is 0 and other func is vswitch,pagecount is 1", func() {
			It("Should omit func platform and vswitch in profile memory", func() {
				filter := &MemoryDefaultsFilter{}
				// Create a test case with sample input
				hp := &v1.HostProfile{
					Spec: v1.HostProfileSpec{
						Memory: v1.MemoryNodeList{
							{
								Node: 1,
								Functions: v1.MemoryFunctionList{
									{
										Function:  "platform",
										PageSize:  "4KB",
										PageCount: 0,
									},
									{
										Function:  "vswitch",
										PageSize:  "4KB",
										PageCount: 1,
									},
									{
										Function:  "vm",
										PageSize:  "4KB",
										PageCount: 2,
									},
								},
							},
						},
					},
				}

				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type
				nodes := v1.MemoryNodeList{
					{
						Node: 1,
						Functions: v1.MemoryFunctionList{
							{
								Function:  "vm",
								PageSize:  "4KB",
								PageCount: 2,
							},
						},
					},
				}
				// Call the Filter method
				err := filter.Filter(hp, deployment)
				Expect(err).To(BeNil())
				Expect(hp.Spec.Memory).To(Equal(nodes))
			})
		})
	})
	Describe("Test ProcessorDefaultsFilter", func() {
		Context("When personality is controller", func() {
			It("Should ignore platform func in processors", func() {
				filter := &ProcessorDefaultsFilter{}
				personality := hosts.PersonalityController
				// Create a test case with sample input
				hp := &v1.HostProfile{
					Spec: v1.HostProfileSpec{
						ProfileBaseAttributes: v1.ProfileBaseAttributes{
							Personality: &personality,
						},
						Processors: v1.ProcessorNodeList{
							{
								Node: 1,
								Functions: v1.ProcessorFunctionList{
									{
										Function: cpus.CPUFunctionPlatform,
										Count:    2,
									},
									{
										Function: "vm",
										Count:    3,
									},
								},
							},
						},
					},
				}

				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type
				expNodes := v1.ProcessorNodeList{
					{
						Node: 1,
						Functions: v1.ProcessorFunctionList{
							{
								Function: "vm",
								Count:    3,
							},
						},
					},
				}
				// Call the Filter method
				err := filter.Filter(hp, deployment)
				Expect(err).To(BeNil())
				Expect(hp.Spec.Processors).To(Equal(expNodes))
			})
		})
		Context("When personality is worker", func() {
			It("Should ignore platform func in processors", func() {
				filter := &ProcessorDefaultsFilter{}
				personality := hosts.PersonalityWorker
				// Create a test case with sample input
				hp := &v1.HostProfile{
					Spec: v1.HostProfileSpec{
						ProfileBaseAttributes: v1.ProfileBaseAttributes{
							Personality: &personality,
						},
						Processors: v1.ProcessorNodeList{
							{
								Node: 1,
								Functions: v1.ProcessorFunctionList{
									{
										Function: cpus.CPUFunctionPlatform,
										Count:    1,
									},
									{
										Function: "vswitch",
										Count:    1,
									},
								},
							},
						},
					},
				}

				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type

				expNodes := v1.ProcessorNodeList{
					{
						Node: 1,
						Functions: v1.ProcessorFunctionList{
							{
								Function: "vswitch",
								Count:    1,
							},
						},
					},
				}
				// Call the Filter method
				err := filter.Filter(hp, deployment)
				Expect(err).To(BeNil())
				Expect(hp.Spec.Processors).To(Equal(expNodes))
			})
		})
	})
	Describe("Test ProcessorClearAllFilter", func() {
		Context("When processors is not nil", func() {
			It("Removes all processor configurations", func() {
				filter := &ProcessorClearAllFilter{}
				hp := &v1.HostProfile{
					Spec: v1.HostProfileSpec{
						Processors: v1.ProcessorNodeList{},
					},
				}
				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type
				// Call the Filter method
				err := filter.Filter(hp, deployment)
				Expect(err).To(BeNil())
				Expect(hp.Spec.Processors).To(BeNil())
			})
		})
	})
	Describe("Test VolumeGroupFilter", func() {
		Context("When there is volume group that is in blacklist", func() {
			It("Removes volume group i.e present in blacklist", func() {
				filter := &VolumeGroupFilter{
					Blacklist: []string{"vg1", "vg2"},
				}
				lvmType := "thin"
				size := 1
				hp := &v1.HostProfile{
					Spec: v1.HostProfileSpec{
						Storage: &v1.ProfileStorageInfo{
							VolumeGroups: &v1.VolumeGroupList{
								{
									Name:    "vg1",
									LVMType: &lvmType,
								},
								{
									Name:    "vg3",
									LVMType: &lvmType,
									PhysicalVolumes: v1.PhysicalVolumeList{
										{
											Type: "disk",
											Path: "/a/b/c",
											Size: &size,
										},
									},
								},
							},
						},
					},
				}
				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type
				expVgs := v1.VolumeGroupList{
					{

						Name:    "vg3",
						LVMType: &lvmType,
						PhysicalVolumes: v1.PhysicalVolumeList{
							{
								Type: "disk",
								Path: "/a/b/c",
								Size: &size,
							},
						},
					},
				}
				// Call the Filter method
				err := filter.Filter(hp, deployment)
				Expect(err).To(BeNil())
				Expect(*hp.Spec.Storage.VolumeGroups).To(Equal(expVgs))
			})
		})
	})
	Describe("Test InterfaceDefaultsFilter", func() {
		Context("When Interface has defaults", func() {
			It("Defaults to be filtereout to nil", func() {
				filter := &InterfaceDefaultsFilter{}
				defMTU := interfaces.DefaultMTU
				mtu := 1400
				hp := &v1.HostProfile{
					Spec: v1.HostProfileSpec{
						Interfaces: &v1.InterfaceInfo{
							Ethernet: v1.EthernetList{
								{
									CommonInterfaceInfo: v1.CommonInterfaceInfo{
										MTU: &defMTU,
									},
								},
							},
							VLAN: v1.VLANList{
								{
									CommonInterfaceInfo: v1.CommonInterfaceInfo{},
								},
							},
							VF: v1.VFList{
								{
									CommonInterfaceInfo: v1.CommonInterfaceInfo{
										MTU: &mtu,
									},
								},
							},
						},
					},
				}
				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type

				expInterfaces := v1.InterfaceInfo{
					Ethernet: v1.EthernetList{
						{
							CommonInterfaceInfo: v1.CommonInterfaceInfo{},
						},
					},
					VLAN: v1.VLANList{
						{
							CommonInterfaceInfo: v1.CommonInterfaceInfo{},
						},
					},
					VF: v1.VFList{
						{
							CommonInterfaceInfo: v1.CommonInterfaceInfo{
								MTU: &mtu,
							},
						},
					},
				}

				// Call the Filter method
				err := filter.Filter(hp, deployment)
				Expect(err).To(BeNil())
				Expect(*hp.Spec.Interfaces).To(Equal(expInterfaces))
			})
		})
	})
	Describe("Test InterfaceMTUFilter checkMTU func", func() {
		Context("When highwatermarks is not defaultMTU", func() {
			It("InterfaceInfo mtu should equals to highwatermarks", func() {
				filter := &InterfaceMTUFilter{
					highwatermarks: map[string]int{"admin": 1600},
				}
				mtu := 1400

				info := &v1.CommonInterfaceInfo{
					MTU:              &mtu,
					PlatformNetworks: &v1.PlatformNetworkItemList{"admin"},
				}
				// Call the CheckMTU method
				filter.CheckMTU(info)
				Expect(*info.MTU).To(Equal(filter.highwatermarks["admin"]))
			})
		})
		Context("When highwatermarks equals defaultMTU", func() {
			It("Highwatermarks should equals to InterfaceInfo mtu", func() {
				filter := &InterfaceMTUFilter{
					highwatermarks: map[string]int{"admin": 1600},
				}
				mtu := 1400

				info := &v1.CommonInterfaceInfo{
					MTU:              &mtu,
					PlatformNetworks: &v1.PlatformNetworkItemList{"admin"},
				}
				// Call the CheckMTU method
				filter.CheckMTU(info)
				Expect(filter.highwatermarks["admin"]).To(Equal(*info.MTU))
			})
		})
		Context("When interfaceInfo mtu is higher than  highwatermarks", func() {
			It("InterfaceInfo mtu should equals to highwatermarks", func() {
				filter := &InterfaceMTUFilter{
					highwatermarks: map[string]int{"admin": 1600},
				}
				mtu := 1400
				info := &v1.CommonInterfaceInfo{
					MTU:              &mtu,
					PlatformNetworks: &v1.PlatformNetworkItemList{"admin"},
				}
				// Call the CheckMTU method
				filter.CheckMTU(info)
				Expect(filter.highwatermarks["admin"]).To(Equal(*info.MTU))
			})
		})
	})
	Describe("Test InterfaceMTUFilter CheckMemberMTU func", func() {
		Context("When ethernet name is in BondInfo's members", func() {
			It("Ethernet's MTU equals to bondInfo's MTU", func() {
				filter := &InterfaceMTUFilter{
					highwatermarks: map[string]int{"admin": 1600},
				}
				mtu := 1400
				ethnName := "eth0"
				ethernet := v1.EthernetList{
					{
						CommonInterfaceInfo: v1.CommonInterfaceInfo{
							Name: ethnName,
						},
					},
				}
				info := &v1.BondInfo{
					Members: []string{"fake1", ethnName, "fake2"},
					CommonInterfaceInfo: v1.CommonInterfaceInfo{
						MTU: &mtu,
					},
				}
				// Call the CheckMTU method
				filter.CheckMemberMTU(info, ethernet)
				Expect(*ethernet[0].MTU).To(Equal(*info.MTU))
			})
		})
	})
	Describe("Test InterfaceMTUFilter filter func", func() {
		Context("When hostprofile interfaces is not nil", func() {
			It("Filter and update the MTU", func() {
				filter := &InterfaceMTUFilter{
					highwatermarks: map[string]int{"admin": 1600},
				}
				ethnName := "eth0"
				defMTU := interfaces.DefaultMTU
				mtu := 1400
				mtuBond := 1456
				hp := &v1.HostProfile{
					Spec: v1.HostProfileSpec{
						Interfaces: &v1.InterfaceInfo{
							Ethernet: v1.EthernetList{
								{
									CommonInterfaceInfo: v1.CommonInterfaceInfo{
										Name: ethnName,
										MTU:  &defMTU,
									},
								},
							},
							VLAN: v1.VLANList{
								{
									CommonInterfaceInfo: v1.CommonInterfaceInfo{
										MTU: &mtu,
									},
								},
							},
							VF: v1.VFList{
								{
									CommonInterfaceInfo: v1.CommonInterfaceInfo{
										MTU: &mtu,
									},
								},
							},
							Bond: v1.BondList{
								{
									CommonInterfaceInfo: v1.CommonInterfaceInfo{
										MTU: &mtuBond,
									},
									Members: []string{"fake1", ethnName, "fake2"},
								},
							},
						},
					},
				}

				expInterfaces := &v1.InterfaceInfo{
					Ethernet: v1.EthernetList{
						{
							CommonInterfaceInfo: v1.CommonInterfaceInfo{
								MTU:  &mtuBond,
								Name: ethnName,
							},
						},
					},
					VLAN: v1.VLANList{
						{
							CommonInterfaceInfo: v1.CommonInterfaceInfo{
								MTU: &mtu,
							},
						},
					},
					VF: v1.VFList{
						{
							CommonInterfaceInfo: v1.CommonInterfaceInfo{
								MTU: &mtu,
							},
						},
					},
					Bond: v1.BondList{
						{
							CommonInterfaceInfo: v1.CommonInterfaceInfo{
								MTU: &mtuBond,
							},
							Members: []string{"fake1", ethnName, "fake2"},
						},
					},
				}

				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type

				// Call the Filter method
				err := filter.Filter(hp, deployment)
				Expect(err).To(BeNil())
				Expect(*hp.Spec.Interfaces).To(Equal(*expInterfaces))
			})
		})
	})

	Describe("Test InterfaceNamingFilter filter func", func() {
		Context("When hostprofile interfaces is not nil", func() {
			It("Filter and update the interface name", func() {
				filter := &InterfaceNamingFilter{
					updates: map[string]string{},
				}
				ethnName := "eth0"
				defMTU := interfaces.DefaultMTU
				mtu := 1400
				mtuBond := 1456
				metric := 1
				hp := &v1.HostProfile{
					Spec: v1.HostProfileSpec{
						Routes: v1.RouteList{
							{
								Interface: "eth0",
								Network:   "admin",
								Prefix:    1,
								Gateway:   "Gateway",
								Metric:    &metric,
							},
						},
						Interfaces: &v1.InterfaceInfo{
							Ethernet: v1.EthernetList{
								{
									CommonInterfaceInfo: v1.CommonInterfaceInfo{
										Name: ethnName,
										MTU:  &defMTU,
									},
								},
							},
							VLAN: v1.VLANList{
								{
									CommonInterfaceInfo: v1.CommonInterfaceInfo{
										MTU: &mtu,
									},
								},
							},
							VF: v1.VFList{
								{
									CommonInterfaceInfo: v1.CommonInterfaceInfo{
										MTU: &mtu,
									},
								},
							},
							Bond: v1.BondList{
								{
									CommonInterfaceInfo: v1.CommonInterfaceInfo{
										MTU: &mtuBond,
									},
									Members: []string{"fake1", ethnName, "fake2"},
								},
							},
						},
					},
				}

				expInterfaces := &v1.InterfaceInfo{
					Ethernet: v1.EthernetList{
						{
							CommonInterfaceInfo: v1.CommonInterfaceInfo{
								MTU:  &defMTU,
								Name: ethnName,
							},
						},
					},
					VLAN: v1.VLANList{
						{
							CommonInterfaceInfo: v1.CommonInterfaceInfo{
								MTU: &mtu,
							},
						},
					},
					VF: v1.VFList{
						{
							CommonInterfaceInfo: v1.CommonInterfaceInfo{
								MTU: &mtu,
							},
						},
					},
					Bond: v1.BondList{
						{
							CommonInterfaceInfo: v1.CommonInterfaceInfo{
								MTU: &mtuBond,
							},
							Members: []string{"fake1", ethnName, "fake2"},
						},
					},
				}

				deployment := &Deployment{} // Dummy deployment for testing, assuming you have a Deployment type

				// Call the Filter method
				err := filter.Filter(hp, deployment)
				Expect(err).To(BeNil())
				Expect(*hp.Spec.Interfaces).To(Equal(*expInterfaces))
			})
		})
	})

	Describe("Test InterfaceNamingFilter checkInterface func", func() {
		Context("When platform network is admin", func() {
			It("Interface info name should not change", func() {
				filter := &InterfaceNamingFilter{
					updates: map[string]string{},
				}
				infoName := "eth0"
				info := &v1.CommonInterfaceInfo{
					Name:             infoName,
					PlatformNetworks: &v1.PlatformNetworkItemList{"admin"},
				}
				// Call the CheckInterface method
				filter.CheckInterface(info)
				Expect(info.Name).To(Equal(infoName))
			})
		})
		Context("When platform network is pxeboot", func() {
			It("Interface info name should be pxeboot", func() {
				updates := make(map[string]string, 1)
				filter := &InterfaceNamingFilter{
					updates: updates,
				}
				infoName := "eth0"
				info := &v1.CommonInterfaceInfo{
					Name:             infoName,
					PlatformNetworks: &v1.PlatformNetworkItemList{pxebootNetwork},
				}
				// Call the CheckInterface method
				filter.CheckInterface(info)
				Expect(filter.updates[infoName]).To(Equal(pxebootIface))
				Expect(info.Name).To(Equal(pxebootIface))
			})
		})
		Context("When platform network is mgmt", func() {
			It("Interface info name should be mgmt", func() {
				filter := &InterfaceNamingFilter{
					updates: map[string]string{},
				}
				infoName := "eth0"
				info := &v1.CommonInterfaceInfo{
					Name:             infoName,
					PlatformNetworks: &v1.PlatformNetworkItemList{mgmtNetwork},
				}
				// Call the CheckInterface method
				filter.CheckInterface(info)
				Expect(filter.updates[infoName]).To(Equal(mgmtIface))
				Expect(info.Name).To(Equal(mgmtIface))
			})
		})
		Context("When platform network is cluster-host", func() {
			It("Interface info name should be cluster-host", func() {
				filter := &InterfaceNamingFilter{
					updates: map[string]string{},
				}
				infoName := "eth0"
				info := &v1.CommonInterfaceInfo{
					Name:             infoName,
					PlatformNetworks: &v1.PlatformNetworkItemList{clusterNetwork},
				}
				// Call the CheckInterface method
				filter.CheckInterface(info)
				Expect(filter.updates[infoName]).To(Equal(clusterIface))
				Expect(info.Name).To(Equal(clusterIface))
			})
		})
		Context("When platform network is oam", func() {
			It("Interface info name should be oam", func() {
				filter := &InterfaceNamingFilter{
					updates: map[string]string{},
				}
				infoName := "eth0"
				info := &v1.CommonInterfaceInfo{
					Name:             infoName,
					PlatformNetworks: &v1.PlatformNetworkItemList{oamNetwork},
				}
				// Call the CheckInterface method
				filter.CheckInterface(info)
				Expect(filter.updates[infoName]).To(Equal(oamIface))
				Expect(info.Name).To(Equal(oamIface))
			})
		})
	})
	Describe("Test AddressPoolFilter filter func", func() {
		Context("When addresspool has controllerAddress as empty", func() {
			It("Updated the controllerAddresses to nil", func() {
				emptyString := ""
				addr0 := "192.168.192.01"
				addr1 := "192.168.192.02"
				in := &AddressPoolFilter{}
				platform_network := &v1.PlatformNetwork{}
				deployment := &Deployment{
					AddressPools: []*v1.AddressPool{
						{
							Spec: v1.AddressPoolSpec{
								Controller0Address: &emptyString,
								Controller1Address: &emptyString,
							},
						},
						{
							Spec: v1.AddressPoolSpec{
								Controller0Address: &addr0,
								Controller1Address: &addr1,
							},
						},
					},
				}
				expAddrPools := []*v1.AddressPool{
					{
						Spec: v1.AddressPoolSpec{},
					},
					{
						Spec: v1.AddressPoolSpec{
							Controller0Address: &addr0,
							Controller1Address: &addr1,
						},
					},
				}
				// Call the filter method
				err := in.Filter(platform_network, deployment)
				Expect(err).To(BeNil())
				Expect(deployment.AddressPools).To(Equal(expAddrPools))
			})
		})
	})
})
