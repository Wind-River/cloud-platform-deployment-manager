/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package host

import (
	"context"
	"fmt"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/hosts"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/interfaces"
	"github.com/imdario/mergo"
	perrors "github.com/pkg/errors"
	starlingxv1beta1 "github.com/wind-river/titanium-deployment-manager/pkg/apis/starlingx/v1beta1"
	"github.com/wind-river/titanium-deployment-manager/pkg/controller/common"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// MergeProfiles invokes the mergo.Merge API with our desired modifiers.
func MergeProfiles(a, b *starlingxv1beta1.HostProfileSpec) (*starlingxv1beta1.HostProfileSpec, error) {
	t := common.DefaultMergeTransformer
	err := mergo.Merge(a, b, mergo.WithOverride, mergo.WithTransformers(t))
	if err != nil {
		err = perrors.Wrap(err, "mergo.Merge failed to merge profiles")
		return nil, err
	}

	return a, nil
}

// GetHostProfile retrieves a HostProfileSpec from the kubernetes API
func (r *ReconcileHost) GetHostProfile(namespace, profile string) (*starlingxv1beta1.HostProfileSpec, error) {
	instance := &starlingxv1beta1.HostProfile{}
	name := types.NamespacedName{Namespace: namespace, Name: profile}

	err := r.Get(context.TODO(), name, instance)
	if err != nil {
		if !errors.IsNotFound(err) {
			err = perrors.Wrapf(err, "failed to get profile: %s", name)
			return nil, err
		} else {
			msg := fmt.Sprintf("host profile %q not present", name)
			return nil, common.NewResourceConfigurationDependency(msg)
		}
	}

	return &instance.Spec, nil
}

// DeleteHostProfile deletes a HostProfile from the kubernetes API
func (r *ReconcileHost) DeleteHostProfile(namespace, profile string) error {
	instance := &starlingxv1beta1.HostProfile{}
	name := types.NamespacedName{Namespace: namespace, Name: profile}

	err := r.Get(context.TODO(), name, instance)
	if !errors.IsNotFound(err) {
		err = perrors.Wrapf(err, "failed to get profile: %s", name)
		return err

	} else if err != nil {
		err = r.Delete(context.TODO(), instance)
		if err != nil {
			err = perrors.Wrapf(err, "failed to delete profile: %s", name)
			return err
		}
	}

	return nil
}

// mergeProfileChain merges the profile attributes from each profile in the
// inheritance chain.  This is done recursively and fields set in lower profiles
// take precedence over its parent/base profile attributes.  Arrays are handled
// by looking for equivalent entries in the base profile attribute and replacing
// their values.  Array entries that are not found in the base profile are
// added to the array.
func (r *ReconcileHost) mergeProfileChain(namespace string, current *starlingxv1beta1.HostProfileSpec, visited map[string]bool) (*starlingxv1beta1.HostProfileSpec, error) {
	if current.Base != nil {
		if value, ok := visited[*current.Base]; ok && value {
			msg := fmt.Sprintf("profile loop detected at: %s", *current.Base)
			return nil, common.NewValidationError(msg)
		}

		parent, err := r.GetHostProfile(namespace, *current.Base)
		if err != nil {
			return nil, err
		}

		parent, err = r.mergeProfileChain(namespace, parent, visited)
		if err != nil {
			return nil, err
		}

		return MergeProfiles(parent, current)
	}

	defaultCopy := DefaultHostProfile.DeepCopy()
	return MergeProfiles(defaultCopy, current)
}

// BuildCompositeProfile combines the default profile, the profile inheritance
// chain, and host specific overrides to form a final composite profile that
// will be applied to the host at configuration time.
func (r *ReconcileHost) BuildCompositeProfile(host *starlingxv1beta1.Host) (*starlingxv1beta1.HostProfileSpec, error) {
	// Start with the explicit profile attached to the host.
	profile, err := r.GetHostProfile(host.Namespace, host.Spec.Profile)
	if err != nil {
		return nil, err
	}

	// Initialize map to track which profiles have already been visited so
	// that we can catch loops.
	visited := make(map[string]bool)

	// Traverse the list of profiles until the root profile is found.
	// Attributes from lower profiles (those closest to the host level) are
	// merged into the higher level profile.
	composite, err := r.mergeProfileChain(host.Namespace, profile, visited)
	if err != nil {
		return composite, err
	}

	// Finally, if the user had provided any per-host overrides then apply
	// over the composite profile.
	if host.Spec.Overrides != nil {
		// Merge the host overrides into the composite profile
		composite, err = MergeProfiles(composite, host.Spec.Overrides)
		if err != nil {
			return composite, err
		}
	}

	if composite.Interfaces != nil && len(composite.Interfaces.Ethernet) == 0 {
		// In some cases it is necessary to set the "ethernet" attribute to
		// an empty array in order to override the list of interfaces from a
		// parent profile, but we never want to override the system defaults
		// which will be applied later so reset this value to nil so that
		// the values from the defaults will be taken.
		composite.Interfaces.Ethernet = nil
	}

	return composite, nil
}

// validateProfileUniqueInterfaces ensures that interface names are unique.  The
// system API will check for this on its own but guaranteeing that the interface
// data is as clean as possible helps simplify some of the coding choice in
// the interface reconciliation code.
func (r *ReconcileHost) validateProfileUniqueInterfaces(host *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec) error {
	present := make(map[string]bool)

	for _, e := range profile.Interfaces.Ethernet {
		if _, ok := present[e.Name]; ok {
			msg := fmt.Sprintf("interfaces names must be unique; Ethernet %s is a duplicate.", e.Name)
			return common.NewValidationError(msg)
		}
	}

	for _, e := range profile.Interfaces.Bond {
		if _, ok := present[e.Name]; ok {
			msg := fmt.Sprintf("interfaces names must be unique; Bond %s is a duplicate.", e.Name)
			return common.NewValidationError(msg)
		}
	}

	for _, e := range profile.Interfaces.VLAN {
		if _, ok := present[e.Name]; ok {
			msg := fmt.Sprintf("interfaces names must be unique; VLAN %s is a duplicate.", e.Name)
			return common.NewValidationError(msg)
		}
	}

	return nil
}

// validateLoopbackInterface validates that if a loopback interface is specified
// that it references a port with the same name.  This is to ensure that the
// interface reconciliation code can make some assumptions about the naming
// strategy and therefore be simplified.
func (r *ReconcileHost) validateLoopbackInterface(host *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec) error {
	for _, e := range profile.Interfaces.Ethernet {
		if e.Name == interfaces.LoopbackInterfaceName || e.Port.Name == interfaces.LoopbackInterfaceName {
			if e.Name != e.Port.Name {
				msg := "the virtual loopback interface must reference a port with the same name"
				return common.NewValidationError(msg)
			}
		}
	}
	return nil
}

// validateProfileInterfaces does minimal validation over the list of
// interfaces to be configured.
func (r *ReconcileHost) validateProfileInterfaces(host *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec) error {
	if profile.Interfaces == nil {
		msg := "'interfaces' profile attribute is required for all hosts"
		return common.NewValidationError(msg)
	}

	err := r.validateProfileUniqueInterfaces(host, profile)
	if err != nil {
		return err
	}
	err = r.validateLoopbackInterface(host, profile)
	if err != nil {
		return err
	}
	return nil
}

// validateBoardManagement performs validation of the Board Management
// host attributes.
func (r *ReconcileHost) validateBoardManagement(host *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec) error {
	if profile.BoardManagement == nil {
		return nil
	}

	bmInfo := profile.BoardManagement
	if bmInfo.Type == nil {
		msg := "Board Management 'type' is a required attribute"
		return common.NewValidationError(msg)
	}

	if bmInfo.Credentials == nil {
		msg := "Board Management 'credentials' is a required attribute"
		return common.NewValidationError(msg)
	} else if bmInfo.Credentials.Password == nil {
		msg := "Board Management 'password' is a required attribute"
		return common.NewValidationError(msg)
	}

	if bmInfo.Address == nil {
		msg := "Board Management 'address' is a required attribute"
		return common.NewValidationError(msg)
	}

	return nil
}

// validateProfileSpec is a private method to validate the contents of a profile
// spec resource.
func (r *ReconcileHost) validateProfileSpec(host *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec) error {
	if profile.Personality == nil {
		msg := "'personality' is a mandatory profile attribute"
		return common.NewValidationError(msg)
	}

	if !profile.HasWorkerSubFunction() {
		if profile.Processors != nil {
			msg := "'processor' profile attributes are only supported on nodes which include the worker subfunction"
			return common.NewValidationError(msg)
		}
	}

	if *profile.Personality != hosts.PersonalityWorker {
		if profile.Storage != nil && profile.Storage.Monitor != nil {
			msg := "'monitor' profile attributes are only permitted on worker nodes"
			return common.NewValidationError(msg)
		}
	}

	if profile.ProvisioningMode == nil {
		msg := "'provisioningMode' is a mandatory profile attribute"
		return common.NewValidationError(msg)
	}

	if *profile.ProvisioningMode == starlingxv1beta1.ProvioningModeStatic {
		if host.Name == hosts.Controller0 {
			msg := "controller-0 must use dynamic provisioning"
			return common.NewValidationError(msg)
		}

		if profile.BootMAC == nil {
			msg := "'bootMAC' profile attribute is required for static provisioning"
			return common.NewValidationError(msg)
		}
	} else {
		if host.Spec.Match == nil {
			msg := "'match' host attribute is required for dynamic provisioning"
			return common.NewValidationError(msg)
		}
	}

	err := r.validateProfileInterfaces(host, profile)
	if err != nil {
		return err
	}

	err = r.validateBoardManagement(host, profile)
	if err != nil {
		return err
	}

	return nil
}

// ValidateProfile examines a composite profile and performs basic validation to
// ensure that all required attributes have been supplied.   This must be done
// at runtime rather than at schema validation time because most fields are
// marked as optional in the schema so that profile inheritance can be used to
// specify only subsets of attributes at each profile level (e.g., an interface
// profile does not need to set personality or administrative state, but some
// profile in the inheritance chain must).  Therefore each individual profile
// itself may not be valid but when attached to a host the full chain of
// profiles must produce a valid set of attributes.
func (r *ReconcileHost) ValidateProfile(host *starlingxv1beta1.Host, profile *starlingxv1beta1.HostProfileSpec) error {
	err := r.validateProfileSpec(host, profile)
	if err != nil {
		return err
	}

	return nil
}
