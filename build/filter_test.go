/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2023-2024 Wind River Systems, Inc. */

package build

import (
	"reflect"

	v1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"

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
