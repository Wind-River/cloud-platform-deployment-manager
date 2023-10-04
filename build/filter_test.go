/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2023 Wind River Systems, Inc. */

package build

import (
	v1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"
	"reflect"

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
				spFilter.Filter(&system, &deployment)
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
				spFilter.Filter(&system, &deployment)

				noSpFilter := NoServiceParameterFilter{}
				noSpFilter.Filter(&system, &deployment)
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
})
