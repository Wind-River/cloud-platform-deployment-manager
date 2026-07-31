/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022, 2024-2025 Wind River Systems, Inc. */

package v1

import (
	"context"
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/memory"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/physicalvolumes"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var hostprofilelog = logf.Log.WithName("hostprofile-resource")

func SetupHostProfileWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&starlingxv1.HostProfile{}).
		WithDefaulter(&HostProfileCustomDefaulter{}).
		WithValidator(&HostProfileCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-starlingx-windriver-com-v1-hostprofile,mutating=true,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=hostprofiles,verbs=create;update,versions=v1,name=mhostprofile.kb.io,admissionReviewVersions=v1,timeoutSeconds=30

type HostProfileCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &HostProfileCustomDefaulter{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (d *HostProfileCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	hostProfile, ok := obj.(*starlingxv1.HostProfile)
	if !ok {
		return fmt.Errorf("expected a HostProfile object but got %T", obj)
	}
	hostprofilelog.Info("default", "name", hostProfile.Name)
	return nil
}

func validateMemoryFunction(node starlingxv1.MemoryNodeInfo, function starlingxv1.MemoryFunctionInfo) error {
	if function.Function == memory.MemoryFunctionPlatform {
		if starlingxv1.PageSize(function.PageSize) != starlingxv1.PageSize4K {
			return errors.New("platform memory must be allocated from 4K pages")
		}
	}

	if starlingxv1.PageSize(function.PageSize) == starlingxv1.PageSize4K {
		if function.Function != memory.MemoryFunctionPlatform {
			return errors.New("4K pages can only be reserved for platform memory")
		}
	}

	return nil
}

func validateMemoryInfo(obj *starlingxv1.HostProfile) error {

	for _, n := range obj.Spec.Memory {
		present := make(map[string]bool)
		for _, f := range n.Functions {
			key := fmt.Sprintf("%s-%s", f.Function, f.PageSize)
			if _, ok := present[key]; ok {
				msg := fmt.Sprintf("duplicate memory entries are not allowed for node %d function %s pagesize %s.",
					n.Node, f.Function, f.PageSize)
				return errors.New(msg)
			}
			present[key] = true

			err := validateMemoryFunction(n, f)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func validateProcessorInfo(obj *starlingxv1.HostProfile) error {
	for _, n := range obj.Spec.Processors {
		present := make(map[string]bool)
		for _, f := range n.Functions {
			key := f.Function
			if _, ok := present[key]; ok {
				msg := fmt.Sprintf("duplicate processor entries are not allowed for node %d function %s.",
					n.Node, f.Function)
				return errors.New(msg)
			}
			present[key] = true
		}
	}

	return nil
}

func validatePhysicalVolumeInfo(obj *starlingxv1.PhysicalVolumeInfo) error {
	if obj.Type == physicalvolumes.PVTypePartition {
		if obj.Size == nil {
			msg := "partition specifications must include a 'size' attribute"
			return errors.New(msg)
		}
	}

	return nil
}

func validateVolumeGroupInfo(obj *starlingxv1.VolumeGroupInfo) error {
	for _, pv := range obj.PhysicalVolumes {
		err := validatePhysicalVolumeInfo(&pv)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateStorageInfo(obj *starlingxv1.HostProfile) error {
	for _, vg := range obj.Spec.Storage.VolumeGroups {
		err := validateVolumeGroupInfo(&vg)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateOVSAccessInfo validates the OVS access configuration for interfaces.
func validateOVSAccessInfo(obj *starlingxv1.HostProfile) error {
	if obj.Spec.Interfaces == nil {
		return nil
	}

	ethernet := obj.Spec.Interfaces.Ethernet
	if ethernet == nil {
		return nil
	}

	// Build a map of ethernet interface names to their info for quick lookup
	ethByName := make(map[string]*starlingxv1.EthernetInfo)
	for i := range ethernet {
		ethByName[ethernet[i].Name] = &ethernet[i]
	}

	ovsAccessCount := 0

	for _, eth := range ethernet {
		if eth.OVSAccess == nil || !*eth.OVSAccess {
			continue
		}

		ovsAccessCount++

		// OVSAccess requires a lower interface
		if eth.Lower == "" {
			return fmt.Errorf("interface %q has ovsAccess enabled but no lower interface specified", eth.Name)
		}

		// The lower interface must be pci-sriov class
		lowerEth, found := ethByName[eth.Lower]
		if !found {
			return fmt.Errorf("interface %q has ovsAccess enabled but lower interface %q not found", eth.Name, eth.Lower)
		}

		if lowerEth.Class != "pci-sriov" {
			return fmt.Errorf("interface %q has ovsAccess enabled but lower interface %q is not class pci-sriov (is %q)", eth.Name, eth.Lower, lowerEth.Class)
		}

		// The lower pci-sriov interface must not have platform networks
		if len(lowerEth.PlatformNetworks) > 0 {
			return fmt.Errorf("interface %q has ovsAccess enabled but lower pci-sriov interface %q has platform networks assigned", eth.Name, eth.Lower)
		}

		// Only one interface per host can have ovsAccess=true
		if ovsAccessCount > 1 {
			return fmt.Errorf("only one ethernet interface per host can have ovsAccess enabled, found multiple")
		}
	}

	// If any ethernet has ovsAccess=true, check that no VLAN is configured
	// on the same pci-sriov lower interface
	if ovsAccessCount > 0 && obj.Spec.Interfaces.VLAN != nil {
		for _, eth := range ethernet {
			if eth.OVSAccess == nil || !*eth.OVSAccess {
				continue
			}
			for _, vlan := range obj.Spec.Interfaces.VLAN {
				if vlan.Lower == eth.Lower {
					return fmt.Errorf("VLAN interface %q cannot be configured on pci-sriov %q which has an upper ethernet with ovsAccess enabled", vlan.Name, eth.Lower)
				}
			}
		}
	}

	// If any ethernet has ovsAccess=true, check that no bond uses the same
	// pci-sriov as a member
	if ovsAccessCount > 0 && obj.Spec.Interfaces.Bond != nil {
		for _, eth := range ethernet {
			if eth.OVSAccess == nil || !*eth.OVSAccess {
				continue
			}
			for _, bond := range obj.Spec.Interfaces.Bond {
				for _, member := range bond.Members {
					if member == eth.Lower {
						return fmt.Errorf("bond interface %q cannot use pci-sriov %q as a member because it has an upper ethernet with ovsAccess enabled", bond.Name, eth.Lower)
					}
				}
			}
		}
	}

	return nil
}

// validateChannelsInfo validates interface channel configuration.
//   - channels (PFChannels): only applicable to ethernet or ae interface types.
//   - sriov_vf_channels (VFChannels): only applicable to pci-sriov class with
//     vf-driver 'netdevice', or vf interface type. The vf-driver must be
//     'netdevice' when sriov_vf_channels is specified.
func validateChannelsInfo(obj *starlingxv1.HostProfile) error {
	if obj.Spec.Interfaces == nil {
		return nil
	}

	// Validate ethernet interfaces
	for _, eth := range obj.Spec.Interfaces.Ethernet {
		// VFChannels on pci-sriov ethernet requires vf-driver netdevice
		if eth.VFChannels != nil && eth.Class == "pci-sriov" {
			if eth.VFDriver == nil || *eth.VFDriver != "netdevice" {
				return fmt.Errorf("interface %q: sriov_vf_channels requires vf-driver to be 'netdevice' for pci-sriov interfaces", eth.Name)
			}
		}
	}

	// Validate VF interfaces, VFChannels requires vf-driver netdevice
	for _, vf := range obj.Spec.Interfaces.VF {
		if vf.VFChannels == nil {
			continue
		}

		if vf.VFDriver == nil || *vf.VFDriver != "netdevice" {
			return fmt.Errorf("interface %q: sriov_vf_channels requires vf-driver to be 'netdevice' for vf interfaces", vf.Name)
		}
	}

	// Validate bond interfaces, PFChannels is valid on ae (bond) types,
	// VFChannels is not applicable
	for _, bond := range obj.Spec.Interfaces.Bond {
		if bond.VFChannels != nil {
			return fmt.Errorf("interface %q: sriov_vf_channels is not applicable to bond (ae) interfaces", bond.Name)
		}
	}

	// Validate VLAN interfaces, neither channels nor sriov_vf_channels
	// are applicable to vlan type
	for _, vlan := range obj.Spec.Interfaces.VLAN {
		if vlan.PFChannels != nil {
			return fmt.Errorf("interface %q: channels is not applicable to vlan interfaces", vlan.Name)
		}
		if vlan.VFChannels != nil {
			return fmt.Errorf("interface %q: sriov_vf_channels is not applicable to vlan interfaces", vlan.Name)
		}
	}

	return nil
}

func validateHostProfile(r *starlingxv1.HostProfile) error {
	if r.Spec.Base != nil && *r.Spec.Base == "" {
		return errors.New("profile base name must not be empty")
	}

	if r.Spec.Memory != nil {
		err := validateMemoryInfo(r)
		if err != nil {
			return err
		}
	}

	if r.Spec.Processors != nil {
		err := validateProcessorInfo(r)
		if err != nil {
			return err
		}
	}

	if r.Spec.Storage != nil {
		err := validateStorageInfo(r)
		if err != nil {
			return err
		}
	}

	if r.Spec.Interfaces != nil {
		err := validateOVSAccessInfo(r)
		if err != nil {
			return err
		}

		err = validateChannelsInfo(r)
		if err != nil {
			return err
		}
	}

	hostprofilelog.Info(AllowedReason)
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-starlingx-windriver-com-v1-hostprofile,mutating=false,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=hostprofiles,versions=v1,name=vhostprofile.kb.io,admissionReviewVersions=v1,timeoutSeconds=30

type HostProfileCustomValidator struct{}

var _ webhook.CustomValidator = &HostProfileCustomValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *HostProfileCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	hostProfile, ok := obj.(*starlingxv1.HostProfile)
	if !ok {
		return nil, fmt.Errorf("expected a HostProfile object but got %T", obj)
	}
	hostprofilelog.Info("validate create", "name", hostProfile.Name)
	return nil, validateHostProfile(hostProfile)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *HostProfileCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	hostProfile, ok := newObj.(*starlingxv1.HostProfile)
	if !ok {
		return nil, fmt.Errorf("expected a HostProfile object but got %T", newObj)
	}
	hostprofilelog.Info("validate update", "name", hostProfile.Name)
	return nil, validateHostProfile(hostProfile)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *HostProfileCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	hostProfile, ok := obj.(*starlingxv1.HostProfile)
	if !ok {
		return nil, fmt.Errorf("expected a HostProfile object but got %T", obj)
	}
	hostprofilelog.Info("validate delete", "name", hostProfile.Name)
	return nil, nil
}
