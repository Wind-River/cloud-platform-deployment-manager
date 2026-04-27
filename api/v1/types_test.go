/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2026 Wind River Systems, Inc. */

package v1

import (
	"encoding/json"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/clusters"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddressInfo", func() {
	Describe("IsKeyEqual", func() {
		It("should return true when addresses match regardless of prefix", func() {
			a := AddressInfo{Address: "193.34.56.87", Prefix: 1}
			b := AddressInfo{Address: "193.34.56.87", Prefix: 4}
			Expect(a.IsKeyEqual(b)).To(BeTrue())
		})

		It("should return false when addresses differ", func() {
			a := AddressInfo{Address: "193.34.56.87", Prefix: 1}
			b := AddressInfo{Address: "193.34.56.89", Prefix: 4}
			Expect(a.IsKeyEqual(b)).To(BeFalse())
		})
	})
})

var _ = Describe("RouteInfo", func() {
	Describe("IsKeyEqual", func() {
		It("should return false when gateways differ", func() {
			a := RouteInfo{Interface: "Interface", Network: "11.22.33.44", Prefix: 2, Gateway: "1.1.1.1"}
			b := RouteInfo{Interface: "Interface", Network: "11.22.33.44", Prefix: 2, Gateway: "1.1.1.2"}
			Expect(a.IsKeyEqual(b)).To(BeFalse())
		})

		It("should return false when prefix differs", func() {
			a := RouteInfo{Interface: "Interface", Network: "11.22.33.44", Prefix: 2}
			b := RouteInfo{Interface: "Interface", Network: "11.22.33.44", Prefix: 6}
			Expect(a.IsKeyEqual(b)).To(BeFalse())
		})

		It("should return true when interface, network, prefix, and gateway all match (preservation)", func() {
			metric1 := 1
			metric2 := 100
			a := RouteInfo{Interface: "eth0", Network: "10.0.0.0", Prefix: 8, Gateway: "10.0.0.1", Metric: &metric1}
			b := RouteInfo{Interface: "eth0", Network: "10.0.0.0", Prefix: 8, Gateway: "10.0.0.1", Metric: &metric2}
			Expect(a.IsKeyEqual(b)).To(BeTrue())
		})

		It("should return false when interfaces differ (preservation)", func() {
			a := RouteInfo{Interface: "eth0", Network: "10.0.0.0", Prefix: 8, Gateway: "10.0.0.1"}
			b := RouteInfo{Interface: "eth1", Network: "10.0.0.0", Prefix: 8, Gateway: "10.0.0.1"}
			Expect(a.IsKeyEqual(b)).To(BeFalse())
		})

		It("should return false when networks differ (preservation)", func() {
			a := RouteInfo{Interface: "eth0", Network: "10.0.0.0", Prefix: 8, Gateway: "10.0.0.1"}
			b := RouteInfo{Interface: "eth0", Network: "192.168.0.0", Prefix: 8, Gateway: "192.168.0.1"}
			Expect(a.IsKeyEqual(b)).To(BeFalse())
		})

		It("should return false when prefixes differ (preservation)", func() {
			a := RouteInfo{Interface: "eth0", Network: "10.0.0.0", Prefix: 8, Gateway: "10.0.0.1"}
			b := RouteInfo{Interface: "eth0", Network: "10.0.0.0", Prefix: 24, Gateway: "10.0.0.1"}
			Expect(a.IsKeyEqual(b)).To(BeFalse())
		})
	})
})

var _ = Describe("HostProfileSpec", func() {
	Describe("HasWorkerSubFunction", func() {
		It("should return true when subfunctions include worker", func() {
			personality := hosts.PersonalityWorker
			spec := &HostProfileSpec{
				ProfileBaseAttributes: ProfileBaseAttributes{
					Personality:  &personality,
					SubFunctions: []SubFunction{"worker"},
				},
			}
			Expect(spec.HasWorkerSubFunction()).To(BeTrue())
		})

		It("should return false when subfunctions do not include worker", func() {
			spec := &HostProfileSpec{
				ProfileBaseAttributes: ProfileBaseAttributes{
					SubFunctions: []SubFunction{"storage"},
				},
			}
			Expect(spec.HasWorkerSubFunction()).To(BeFalse())
		})

		It("should return true when personality is worker with no subfunctions", func() {
			personality := hosts.PersonalityWorker
			spec := &HostProfileSpec{
				ProfileBaseAttributes: ProfileBaseAttributes{
					Personality: &personality,
				},
			}
			Expect(spec.HasWorkerSubFunction()).To(BeTrue())
		})
	})
})

var _ = Describe("OSDInfo", func() {
	Describe("GetClusterName", func() {
		It("should return the cluster name when set", func() {
			name := "ClusterName"
			info := &OSDInfo{ClusterName: &name}
			Expect(info.GetClusterName()).To(Equal(name))
		})

		It("should return CephClusterName when nil", func() {
			info := &OSDInfo{ClusterName: nil}
			Expect(info.GetClusterName()).To(Equal(clusters.CephClusterName))
		})
	})
})

var _ = Describe("SubFunctionFromString", func() {
	It("should convert a string to SubFunction", func() {
		Expect(SubFunctionFromString("worker")).To(Equal(SubFunction("worker")))
	})
})

var _ = Describe("CertificateInfo", func() {
	Describe("DeepEqual", func() {
		It("should return true when type and secret match ignoring signature", func() {
			a := &CertificateInfo{Type: "ssl", Secret: "secret", Signature: ""}
			b := &CertificateInfo{Type: "ssl", Secret: "secret"}
			Expect(a.DeepEqual(b)).To(BeTrue())
		})

		It("should return true when all fields match", func() {
			a := &CertificateInfo{Type: "ssl", Secret: "secret", Signature: "sig"}
			b := &CertificateInfo{Type: "ssl", Secret: "secret", Signature: "sig"}
			Expect(a.DeepEqual(b)).To(BeTrue())
		})
	})

	Describe("IsKeyEqual", func() {
		It("should return true when type and secret match", func() {
			a := CertificateInfo{Type: "ssl", Secret: "secret", Signature: ""}
			b := CertificateInfo{Type: "ssl", Secret: "secret"}
			Expect(a.IsKeyEqual(b)).To(BeTrue())
		})

		It("should return true when all fields match", func() {
			a := CertificateInfo{Type: "ssl", Secret: "secret", Signature: "sig"}
			b := CertificateInfo{Type: "ssl", Secret: "secret", Signature: "sig"}
			Expect(a.IsKeyEqual(b)).To(BeTrue())
		})
	})
})

var _ = Describe("SystemSpec", func() {
	Describe("DeepEqual", func() {
		Context("with Certificates", func() {
			It("should match unordered arrays", func() {
				a := SystemSpec{
					Certificates: CertificateList{
						{Type: "ssl", Secret: "secret2"},
						{Type: "ssl", Secret: "secret1"},
						{Type: "ssl", Secret: "secret3"},
					},
				}
				b := SystemSpec{
					Certificates: CertificateList{
						{Type: "ssl", Secret: "secret1"},
						{Type: "ssl", Secret: "secret3"},
						{Type: "ssl", Secret: "secret2"},
					},
				}
				Expect(a.DeepEqual(&b)).To(BeTrue())
			})

			It("should not match different arrays", func() {
				a := SystemSpec{
					Certificates: CertificateList{{Type: "ssl", Secret: "secret"}},
				}
				b := SystemSpec{}
				Expect(a.DeepEqual(&b)).To(BeFalse())
			})
		})

		Context("with ServiceParameters", func() {
			It("should match unordered arrays", func() {
				a := SystemSpec{
					ServiceParameters: ServiceParameterList{
						{Service: "sysinv", Section: "global", ParamName: "param1", ParamValue: "1"},
						{Service: "sysinv", Section: "global", ParamName: "param2", ParamValue: "2"},
					},
				}
				b := SystemSpec{
					ServiceParameters: ServiceParameterList{
						{Service: "sysinv", Section: "global", ParamName: "param2", ParamValue: "2"},
						{Service: "sysinv", Section: "global", ParamName: "param1", ParamValue: "1"},
					},
				}
				Expect(a.DeepEqual(&b)).To(BeTrue())
			})

			It("should not match different arrays", func() {
				a := SystemSpec{
					ServiceParameters: ServiceParameterList{
						{Service: "sysinv", Section: "global", ParamName: "param", ParamValue: "value"},
					},
				}
				b := SystemSpec{}
				Expect(a.DeepEqual(&b)).To(BeFalse())
			})
		})
	})
})

var _ = Describe("PtpInstanceSpec", func() {
	Describe("UnmarshalJSON", func() {
		It("should unmarshall when parameters are omitted", func() {
			var spec PtpInstanceSpec
			Expect(json.Unmarshal([]byte(`{"service": "ptp4l"}`), &spec)).To(Succeed())
			Expect(spec.InstanceParameters).To(BeNil())
			Expect(spec.Service).To(Equal("ptp4l"))
		})

		It("should unmarshall empty array parameters", func() {
			var spec PtpInstanceSpec
			Expect(json.Unmarshal([]byte(`{"service": "ptp4l", "parameters": []}`), &spec)).To(Succeed())
			Expect(spec.InstanceParameters).To(Equal(map[string][]string{}))
		})

		It("should unmarshall array parameters into global section", func() {
			var spec PtpInstanceSpec
			Expect(json.Unmarshal([]byte(`{"service": "ptp4l", "parameters": ["param1", "param2"]}`), &spec)).To(Succeed())
			Expect(spec.InstanceParameters).To(Equal(map[string][]string{
				"global": {"param1", "param2"},
			}))
		})

		It("should unmarshall sectioned parameters", func() {
			jsonSpec := `{
				"service": "ptp4l",
				"parameters": {
					"global": ["param1", "param2"],
					"unicast_master_table_x": ["param3"]
				}
			}`
			var spec PtpInstanceSpec
			Expect(json.Unmarshal([]byte(jsonSpec), &spec)).To(Succeed())
			Expect(spec.InstanceParameters).To(Equal(map[string][]string{
				"global":                 {"param1", "param2"},
				"unicast_master_table_x": {"param3"},
			}))
		})
	})
})
