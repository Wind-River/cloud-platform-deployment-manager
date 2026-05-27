/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2026 Wind River Systems, Inc. */
package host

import (
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/osds"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/volumegroups"
	th "github.com/gophercloud/gophercloud/testhelper"
	gcClient "github.com/gophercloud/gophercloud/testhelper/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	ctrlcommon "github.com/wind-river/cloud-platform-deployment-manager/internal/controller/common"
	v1info "github.com/wind-river/cloud-platform-deployment-manager/platform"
	"k8s.io/client-go/tools/record"
)

var _ = Describe("VolumeGroup operations", func() {
	var reconciler *HostReconciler

	BeforeEach(func() {
		logger := logr.Discard()
		reconciler = &HostReconciler{
			ReconcilerEventLogger: &ctrlcommon.EventLogger{
				EventRecorder: record.NewFakeRecorder(100),
				Logger:        logger,
			},
		}
	})

	Context("AddVolumeGroup", func() {
		It("should create a new volume group with capabilities", func() {
			th.Mux.HandleFunc("/ilvgs", func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal("POST"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintf(w, `{"uuid":"vg-uuid-1","lvm_vg_name":"lvm-provisioner"}`)
			})

			host := &v1info.HostInfo{}
			host.ID = "f47ac10b-58cc-4372-a567-0e02b2c3d479"
			lvmType := "thick"
			lvmFunction := "lvm-csi"
			vgInfo := &starlingxv1.VolumeGroupInfo{
				Name:        "lvm-provisioner",
				LVMType:     &lvmType,
				LVMFunction: &lvmFunction,
				PhysicalVolumes: starlingxv1.PhysicalVolumeList{
					{Path: "/dev/disk/by-path/pci-0000:00:0d.0-ata-3.0", Type: "disk"},
				},
			}
			caps := &volumegroups.CapabilitiesOpts{
				LVMType:     &lvmType,
				LVMFunction: &lvmFunction,
			}

			client := gcClient.ServiceClient()
			updated, err := reconciler.AddVolumeGroup(client, host, vgInfo, caps)
			Expect(err).ToNot(HaveOccurred())
			Expect(updated).To(BeTrue())
		})
	})

	Context("UpdateVolumeGroup", func() {
		It("should update capabilities when they differ", func() {
			vgID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
			th.Mux.HandleFunc("/ilvgs/"+vgID, func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal("PATCH"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintf(w, `{"uuid":"%s","lvm_vg_name":"cgts-vg","capabilities":{"lvm_function":"lvm-csi"}}`, vgID)
			})

			vgHost := &volumegroups.VolumeGroup{
				ID:           vgID,
				Capabilities: volumegroups.Capabilities{},
			}
			vgHost.Name = "cgts-vg"

			lvmFunction := "lvm-csi"
			caps := &volumegroups.CapabilitiesOpts{
				LVMFunction: &lvmFunction,
			}

			client := gcClient.ServiceClient()
			updated, err := reconciler.UpdateVolumeGroup(client, vgHost, caps)
			Expect(err).ToNot(HaveOccurred())
			Expect(updated).To(BeTrue())
		})
	})

	Context("DeleteVolumeGroups", func() {
		It("should delete volume groups not present in profile", func() {
			deletedID := "d4e5f6a7-b8c9-0123-4567-89abcdef0123"
			th.Mux.HandleFunc("/ilvgs/"+deletedID, func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal("DELETE"))
				w.WriteHeader(http.StatusNoContent)
			})

			lvmType := "thick"
			lvmTypeThin := "thin"
			lvmFunction := "lvm-csi"

			// Host has two volume groups
			host := &v1info.HostInfo{
				VolumeGroups: []volumegroups.VolumeGroup{
					{
						ID:           "c3d4e5f6-a7b8-9012-3456-789abcdef012",
						Capabilities: volumegroups.Capabilities{LVMType: &lvmType, LVMFunction: &lvmFunction},
						LVMInfo:      volumegroups.LVMInfo{Name: "lvm-provisioner"},
					},
					{
						ID:           deletedID,
						Capabilities: volumegroups.Capabilities{LVMType: &lvmTypeThin, LVMFunction: &lvmFunction},
						LVMInfo:      volumegroups.LVMInfo{Name: "lvm-provisioner-thin"},
					},
				},
			}

			// Profile only has lvm-provisioner
			profile := &starlingxv1.HostProfileSpec{
				Storage: &starlingxv1.ProfileStorageInfo{
					VolumeGroups: starlingxv1.VolumeGroupList{
						{
							Name:        "lvm-provisioner",
							LVMType:     &lvmType,
							LVMFunction: &lvmFunction,
							PhysicalVolumes: starlingxv1.PhysicalVolumeList{
								{Path: "/dev/disk/by-path/pci-0000:00:0d.0-ata-3.0", Type: "disk"},
							},
						},
					},
				},
			}

			instance := &starlingxv1.Host{}
			client := gcClient.ServiceClient()
			updated, err := reconciler.DeleteVolumeGroups(client, instance, profile, host)
			Expect(err).ToNot(HaveOccurred())
			Expect(updated).To(BeTrue())
		})
	})
})

var _ = Describe("osdUpdateRequired", func() {
	Context("when journal is nil in spec", func() {
		It("should return false", func() {
			osdInfo := &starlingxv1.OSDInfo{Function: "osd", Path: "/dev/sda"}
			osd := &osds.OSD{}
			_, result := osdUpdateRequired(osdInfo, osd)
			Expect(result).To(BeFalse())
		})
	})

	Context("when journal is set and osd has no location", func() {
		It("should return true with journal location and size", func() {
			osdInfo := &starlingxv1.OSDInfo{
				Function: "osd",
				Path:     "/dev/sda",
				Journal:  &starlingxv1.JournalInfo{Location: "/dev/sdb", Size: 1},
			}
			osd := &osds.OSD{}
			opts, result := osdUpdateRequired(osdInfo, osd)
			Expect(result).To(BeTrue())
			Expect(*opts.JournalLocation).To(Equal("/dev/sdb"))
			Expect(*opts.JournalSize).To(Equal(1))
		})
	})

	Context("when journal sizes differ", func() {
		It("should return true with updated size", func() {
			osdInfo := &starlingxv1.OSDInfo{
				Function: "osd",
				Path:     "/dev/sda",
				Journal:  &starlingxv1.JournalInfo{Location: "/dev/sdb", Size: 2},
			}
			loc := "/dev/sdb"
			sizeMiB := 1024 // 1 GiB in MiB
			osd := &osds.OSD{
				JournalInfo: osds.JournalInfo{
					Location: &loc,
					Size:     &sizeMiB,
				},
			}
			opts, result := osdUpdateRequired(osdInfo, osd)
			Expect(result).To(BeTrue())
			Expect(*opts.JournalSize).To(Equal(2))
		})
	})

	Context("when journal sizes match", func() {
		It("should return false", func() {
			osdInfo := &starlingxv1.OSDInfo{
				Function: "osd",
				Path:     "/dev/sda",
				Journal:  &starlingxv1.JournalInfo{Location: "/dev/sdb", Size: 1},
			}
			loc := "/dev/sdb"
			sizeMiB := 1024 // 1 GiB = 1024 MiB, Gibibytes() = 1024/1024 = 1
			osd := &osds.OSD{
				JournalInfo: osds.JournalInfo{
					Location: &loc,
					Size:     &sizeMiB,
				},
			}
			_, result := osdUpdateRequired(osdInfo, osd)
			Expect(result).To(BeFalse())
		})
	})
})
