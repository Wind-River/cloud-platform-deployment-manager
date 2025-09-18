/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2025 Wind River Systems, Inc. */

package common

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Test common factory config", func() {
	const namespace = "deployment"

	configMap := v1.ConfigMap{}

	BeforeEach(func() {
		configMap.SetName("factory-install")
		configMap.SetNamespace("deployment")
		configMap.Data = map[string]string{}
	})

	Context("FactoryConfigAllowed", func() {
		It("should allow if config-map is configured", func() {
			configMap.Data[FactoryInstalled] = "true"
			_, ok := FactoryReconfigAllowed(namespace, &configMap)
			Expect(ok).To(BeTrue())
		})

		It("should prohibit if config-map factory-install is false", func() {
			configMap.Data[FactoryInstalled] = "false"
			_, ok := FactoryReconfigAllowed(namespace, &configMap)
			Expect(ok).To(BeFalse())
		})

		It("should prohibit if config-map factory-install data is empty", func() {
			configMap.Data = nil
			_, ok := FactoryReconfigAllowed(namespace, &configMap)
			Expect(ok).To(BeFalse())
		})

		It("should prohibit if config-map factory-install contains factory-config-finalized is true", func() {
			configMap.Data[FactoryInstalled] = "true"
			configMap.Data[FactoryConfigFinalized] = "true"
			_, ok := FactoryReconfigAllowed(namespace, &configMap)
			Expect(ok).To(BeFalse())
		})
	})
})
