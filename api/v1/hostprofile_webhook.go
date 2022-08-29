/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */

package v1

import (
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/memory"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/physicalvolumes"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var hostprofilelog = logf.Log.WithName("hostprofile-resource")

func (r *HostProfile) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-starlingx-windriver-com-v1-hostprofile,mutating=true,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=hostprofiles,verbs=create;update,versions=v1,name=mhostprofile.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &HostProfile{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *HostProfile) Default() {
	hostprofilelog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

func validateMemoryFunction(node MemoryNodeInfo, function MemoryFunctionInfo) error {
	if function.Function == memory.MemoryFunctionPlatform {
		if PageSize(function.PageSize) != PageSize4K {
			return errors.New("platform memory must be allocated from 4K pages.")
		}
	}

	if PageSize(function.PageSize) == PageSize4K {
		if function.Function != memory.MemoryFunctionPlatform {
			return errors.New("4K pages can only be reserved for platform memory.")
		}
	}

	return nil
}

func validateMemoryInfo(obj *HostProfile) error {

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

func validateProcessorInfo(obj *HostProfile) error {
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

func validatePhysicalVolumeInfo(obj *PhysicalVolumeInfo) error {
	if obj.Type == physicalvolumes.PVTypePartition {
		if obj.Size == nil {
			msg := "partition specifications must include a 'size' attribute"
			return errors.New(msg)
		}
	}

	return nil
}

func validateVolumeGroupInfo(obj *VolumeGroupInfo) error {
	for _, pv := range obj.PhysicalVolumes {
		err := validatePhysicalVolumeInfo(&pv)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateStorageInfo(obj *HostProfile) error {
	if obj.Spec.Storage.VolumeGroups != nil {
		for _, vg := range *obj.Spec.Storage.VolumeGroups {
			err := validateVolumeGroupInfo(&vg)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *HostProfile) validateHostProfile() error {
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

	hostprofilelog.Info(AllowedReason)
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-starlingx-windriver-com-v1-hostprofile,mutating=false,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=hostprofiles,versions=v1,name=vhostprofile.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &HostProfile{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *HostProfile) ValidateCreate() error {
	hostprofilelog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return r.validateHostProfile()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *HostProfile) ValidateUpdate(old runtime.Object) error {
	hostprofilelog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return r.validateHostProfile()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *HostProfile) ValidateDelete() error {
	hostprofilelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
