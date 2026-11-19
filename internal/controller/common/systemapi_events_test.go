/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2026 Wind River Systems, Inc. */

package common

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("System API events", func() {
	const podNamespace = "platform-deployment-manager"

	var clientset *fake.Clientset
	var events *SystemAPIEvents

	list := func() []v1.Event {
		items, err := clientset.CoreV1().Events(podNamespace).List(context.Background(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		return items.Items
	}

	record := func(kind, name string, status int, reason string) {
		request := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "deployment", Name: name}}
		events.Record(kind, request, ErrSystemAPI{Operation: "update memory", StatusCode: status, Reason: reason})
	}

	BeforeEach(func() {
		clientset = fake.NewSimpleClientset()
		events = newSystemAPIEvents(clientset, scheme.Scheme, &v1.ObjectReference{
			APIVersion: "apps/v1", Kind: "Deployment", Namespace: podNamespace, Name: "dm"})
	})

	It("should create the events in the namespace of the deployment, whatever the resource", func() {
		record("Host", "controller-0", 400, "bad host")
		record("PlatformNetwork", "oam", 400, "bad network")

		Eventually(list).Should(HaveLen(2))
		for _, event := range list() {
			Expect(event.Namespace).To(Equal(podNamespace))
			Expect(event.InvolvedObject.Kind).To(Equal("Deployment"))
			Expect(event.InvolvedObject.Name).To(Equal("dm"))
			Expect(event.Type).To(Equal(v1.EventTypeWarning))
			Expect(event.Reason).To(Equal(SystemAPIRejected))
		}
	})

	It("should not combine or drop unrelated failures", func() {
		for i := 0; i < 40; i++ {
			record("Host", fmt.Sprintf("host-%d", i), 400, fmt.Sprintf("reason %d", i))
		}

		Eventually(list).Should(HaveLen(40))
		for _, event := range list() {
			Expect(event.Message).ToNot(ContainSubstring("combined from similar events"))
		}
	})

	It("should fold a failure repeated on every retry into one event with a count", func() {
		for i := 0; i < 5; i++ {
			record("Host", "controller-0", 400, "same reason")
		}

		Eventually(func() int32 {
			items := list()
			if len(items) != 1 {
				return -1
			}
			return items[0].Count
		}).Should(Equal(int32(5)))
	})
})

var _ = Describe("System API events namespace", func() {
	var original string

	BeforeEach(func() {
		original = namespaceFile
		DeferCleanup(func() { namespaceFile = original })
		Expect(os.Unsetenv(NamespaceEnv)).To(Succeed())
	})

	write := func(content string) string {
		path := filepath.Join(GinkgoT().TempDir(), "namespace")
		Expect(os.WriteFile(path, []byte(content), 0o600)).To(Succeed())
		return path
	}

	It("should use the namespace of the pod", func() {
		namespaceFile = write("platform-deployment-manager\n")

		Expect(getNamespace()).To(Equal("platform-deployment-manager"))
	})

	It("should let the environment override the namespace of the pod", func() {
		namespaceFile = write("platform-deployment-manager\n")
		GinkgoT().Setenv(NamespaceEnv, "other")

		Expect(getNamespace()).To(Equal("other"))
	})

	It("should be empty outside of a pod", func() {
		namespaceFile = filepath.Join(GinkgoT().TempDir(), "missing")

		Expect(getNamespace()).To(BeEmpty())
	})
})
