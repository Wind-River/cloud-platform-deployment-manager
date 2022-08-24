/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */
package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
)

var _ = Describe("HostProfile controller", func() {

	Context("HostProfile with data", func() {
		It("Should created successfully", func() {
			ctx := context.Background()
			created := &starlingxv1.HostProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				}}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			// Mock is needed for the further testing
			// Currently there is no update in HostProfile instance
			// So we test only for create
		})
	})
})
