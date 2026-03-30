/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022, 2025-2026 Wind River Systems, Inc. */
package controller

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
)

// createProfile is a helper to create a HostProfile in the test namespace.
func createProfile(ctx context.Context, name string, base *string) *starlingxv1.HostProfile {
	p := &starlingxv1.HostProfile{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: TestNamespace},
		Spec:       starlingxv1.HostProfileSpec{Base: base},
	}
	ExpectWithOffset(1, k8sClient.Create(ctx, p)).To(Succeed())
	ExpectWithOffset(1, k8sClient.Get(ctx, types.NamespacedName{
		Name: name, Namespace: TestNamespace,
	}, p)).To(Succeed())
	return p
}

// createHost is a helper to create a Host referencing a profile.
// Sets a dummy annotation so the map survives API server round-trip.
func createHost(ctx context.Context, name, profile, mac string) {
	h := &starlingxv1.Host{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: TestNamespace,
			Annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": "{}",
			},
		},
		Spec: starlingxv1.HostSpec{
			Profile: profile,
			Match:   &starlingxv1.MatchInfo{BootMAC: &mac},
		},
	}
	ExpectWithOffset(1, k8sClient.Create(ctx, h)).To(Succeed())
}

// cleanupResources deletes all HostProfiles and Hosts in the test namespace.
func cleanupResources(ctx context.Context) {
	err := k8sClient.DeleteAllOf(ctx, &starlingxv1.Host{}, client.InNamespace(TestNamespace))
	Expect(err).ToNot(HaveOccurred())

	err = k8sClient.DeleteAllOf(ctx, &starlingxv1.HostProfile{}, client.InNamespace(TestNamespace))
	Expect(err).ToNot(HaveOccurred())
}

var _ = Describe("HostProfile controller", func() {

	var reconciler *HostProfileReconciler

	BeforeEach(func() {
		reconciler = &HostProfileReconciler{
			Client: k8sClient,
			Scheme: k8sManager.GetScheme(),
		}
	})

	AfterEach(func() {
		cleanupResources(context.Background())
	})

	Describe("ProfileUses", func() {
		It("should return true when controller-1-profile inherits from controller-0-profile", func() {
			ctx := context.Background()
			baseName := "controller-0-profile"
			createProfile(ctx, "controller-0-profile", nil)
			createProfile(ctx, "controller-1-profile", &baseName)

			uses, err := reconciler.ProfileUses(TestNamespace, "controller-1-profile", "controller-0-profile")
			Expect(err).ToNot(HaveOccurred())
			Expect(uses).To(BeTrue())
		})

		It("should return true when compute-0-profile indirectly inherits from worker-base-profile", func() {
			ctx := context.Background()
			workerBase := "worker-base-profile"
			computeBase := "compute-base-profile"
			createProfile(ctx, "worker-base-profile", nil)
			createProfile(ctx, "compute-base-profile", &workerBase)
			createProfile(ctx, "compute-0-profile", &computeBase)

			uses, err := reconciler.ProfileUses(TestNamespace, "compute-0-profile", "worker-base-profile")
			Expect(err).ToNot(HaveOccurred())
			Expect(uses).To(BeTrue())
		})

		It("should return false when storage-0-profile does not reference compute-base-profile", func() {
			ctx := context.Background()
			createProfile(ctx, "storage-0-profile", nil)

			uses, err := reconciler.ProfileUses(TestNamespace, "storage-0-profile", "compute-base-profile")
			Expect(err).ToNot(HaveOccurred())
			Expect(uses).To(BeFalse())
		})

		It("should return false when the base profile does not exist", func() {
			ctx := context.Background()
			missing := "removed-base-profile"
			createProfile(ctx, "compute-0-profile", &missing)

			uses, err := reconciler.ProfileUses(TestNamespace, "compute-0-profile", "worker-base-profile")
			Expect(err).ToNot(HaveOccurred())
			Expect(uses).To(BeFalse())
		})
	})

	Describe("UpdateHosts", func() {
		It("should annotate controllers that directly reference the updated profile", func() {
			ctx := context.Background()

			profile := createProfile(ctx, "controller-0-profile", nil)
			createHost(ctx, "controller-0", "controller-0-profile", "aa:bb:cc:00:00:01")
			createHost(ctx, "controller-1", "controller-0-profile", "aa:bb:cc:00:00:02")

			Expect(reconciler.UpdateHosts(profile)).To(Succeed())

			key := fmt.Sprintf("profile/%s", profile.Name)

			c0 := &starlingxv1.Host{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: "controller-0", Namespace: TestNamespace,
			}, c0)).To(Succeed())
			Expect(c0.Annotations).To(HaveKeyWithValue(key, profile.ResourceVersion))

			c1 := &starlingxv1.Host{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: "controller-1", Namespace: TestNamespace,
			}, c1)).To(Succeed())
			Expect(c1.Annotations).To(HaveKeyWithValue(key, profile.ResourceVersion))
		})

		It("should annotate compute nodes that reference the profile via a base chain", func() {
			ctx := context.Background()
			baseName := "compute-base-profile"

			base := createProfile(ctx, "compute-base-profile", nil)
			createProfile(ctx, "compute-0-profile", &baseName)
			createProfile(ctx, "compute-1-profile", &baseName)

			createHost(ctx, "compute-0", "compute-0-profile", "aa:bb:cc:00:00:03")
			createHost(ctx, "compute-1", "compute-1-profile", "aa:bb:cc:00:00:04")

			Expect(reconciler.UpdateHosts(base)).To(Succeed())

			key := fmt.Sprintf("profile/%s", base.Name)

			w0 := &starlingxv1.Host{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: "compute-0", Namespace: TestNamespace,
			}, w0)).To(Succeed())
			Expect(w0.Annotations).To(HaveKeyWithValue(key, base.ResourceVersion))

			w1 := &starlingxv1.Host{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: "compute-1", Namespace: TestNamespace,
			}, w1)).To(Succeed())
			Expect(w1.Annotations).To(HaveKeyWithValue(key, base.ResourceVersion))
		})

		It("should not annotate storage nodes that use a different profile", func() {
			ctx := context.Background()

			profile := createProfile(ctx, "storage-0-profile", nil)
			createHost(ctx, "storage-0", "storage-1-profile", "aa:bb:cc:00:00:05")

			Expect(reconciler.UpdateHosts(profile)).To(Succeed())

			h := &starlingxv1.Host{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: "storage-0", Namespace: TestNamespace,
			}, h)).To(Succeed())
			key := fmt.Sprintf("profile/%s", profile.Name)
			Expect(h.Annotations).ToNot(HaveKey(key))
		})

		It("should be idempotent when the resource version has not changed", func() {
			ctx := context.Background()

			profile := createProfile(ctx, "storage-0-profile", nil)
			createHost(ctx, "storage-0", "storage-0-profile", "aa:bb:cc:00:00:06")

			Expect(reconciler.UpdateHosts(profile)).To(Succeed())

			before := &starlingxv1.Host{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: "storage-0", Namespace: TestNamespace,
			}, before)).To(Succeed())

			Expect(reconciler.UpdateHosts(profile)).To(Succeed())

			after := &starlingxv1.Host{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: "storage-0", Namespace: TestNamespace,
			}, after)).To(Succeed())

			Expect(after.ResourceVersion).To(Equal(before.ResourceVersion))
		})
	})
})
