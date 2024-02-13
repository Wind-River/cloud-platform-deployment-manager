/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2023-2024 Wind River Systems, Inc. */

package build

import (
	"reflect"
	"strings"

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
		coreNetworkFilter := NewCoreNetworkFilter()
		var get_platform_network = func(nwk_type string) *v1.PlatformNetwork {
			gw := "10.10.10.1"
			order := "random"
			spec := v1.PlatformNetworkSpec{
				Type:               nwk_type,
				Subnet:             "10.10.10.0",
				FloatingAddress:    "10.10.10.2",
				Controller0Address: "10.10.10.3",
				Controller1Address: "10.10.10.4",
				Prefix:             24,
				Gateway:            &gw,
				Allocation: v1.AllocationInfo{
					Type:  "dynamic",
					Order: &order,
					Ranges: []v1.AllocationRange{
						v1.AllocationRange{
							Start: "10.10.10.2",
							End:   "10.10.10.50",
						},
					},
				},
			}
			new_plat_nwk := v1.PlatformNetwork{}
			new_plat_nwk.Spec = spec
			return &new_plat_nwk
		}
		deployment.PlatformNetworks = []*v1.PlatformNetwork{
			get_platform_network("oam"),
			get_platform_network("mgmt"),
			get_platform_network("admin"),
			get_platform_network("storage")}

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
			})
		})

	})

})
