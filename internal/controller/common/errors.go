/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package common

import (
	"fmt"
	"regexp"
)

// Defines common log strings from commonly performed validation checks across
// all different reconcilers.
const (
	NoChangesAfterReconciled           = "configuration changes ignored after initial synchronization has completed"
	ChangedAllowedAfterReconciled      = "manual override; allowing configuration changes after initial synchronization"
	NoProvisioningAfterReconciled      = "resource provisioning ignored after initial synchronization has completed"
	ProvisioningAllowedAfterReconciled = "manual override; allowing resource provisioning after initial synchronization"
)

// extractFaultString extracts the fault message from the fault error.
// It searches for the faultstring pattern and returns the extracted message,
// or the original error string if no pattern is found.
func extractFaultString(err error) string {
	var reason string
	if err != nil {
		reason = err.Error()
	}

	faultRe := regexp.MustCompile(`faultstring.*?:.*?"(.*?)\\`)
	if result := faultRe.FindStringSubmatch(reason); len(result) > 0 {
		return result[1]
	}

	return reason
}

// BaseError defines the common error reporting struct for all other errors
// defined in this package
type BaseError struct {
	message string
}

// Error implements the Error interface for all structures that are derived
// from this one.
func (in BaseError) Error() string {
	return in.message
}

// ErrSystemDependency defines an error to be used when reporting that the
// system itself or a set of multiple resources are not in the correct state to
// proceed with an operation.
type ErrSystemDependency struct {
	BaseError
}

// ErrMissingKubernetesResource defines an error to be used when reporting that
// an operation is unable to find a required resource from the
// kubernetes API.  This error is not intended for system resources that are
// missing.  For those use ErrMissingSystemResource
type ErrMissingKubernetesResource struct {
	BaseError
}

// ErrResourceStatusDependency defines an error to be used when reporting that
// an operation is unable to continue because a resource is not in the correct
// state.
type ErrResourceStatusDependency struct {
	BaseError
}

// ErrResourceConfigurationDependency defines an error to be used when reporting
// that an operation is unable to continue because a resource is not configured
// correctly.
type ErrResourceConfigurationDependency struct {
	BaseError
}

// ErrUnlockError defines an error to be returned when the RPC call
// to unlock the host fails.
type ErrUnlockError struct {
	BaseError
}

// ErrUserDataError defines an error to be used when reporting that an operation
// is unable to continue because the requested configuration is incorrect or
// incomplete.
type ErrUserDataError struct {
	BaseError
}

// ValidationError defines a new error type used to differentiate data
// validation errors from other types of errors.
type ValidationError struct {
	BaseError
}

// HTTPSClientRequired defines a new error type used to signal that a
// a configuration changes requires an HTTPS URL before continuing.
type HTTPSClientRequired struct {
	BaseError
}

// ChangeAfterReconciled defines a new error type used to signal that a
// a configuration changes was received after the resource has already been
// synchronized with the system state.
type ChangeAfterReconciled struct {
	BaseError
}

// PlatformNetworkReconciliationError defines an error to be used when reconciliation
// of platform network / address pool resources fail and the reconciliation request
// needs to be requeued.
type PlatformNetworkReconciliationError struct {
	BaseError
}

// HostNotifyError defines an error to be used when active host controller
// is already being notified while the address pool / platform network
// reconciler attempts to notify.
// Reconciliation request can be requeued immediately for such instances.
type HostNotifyError struct {
	BaseError
}

// NewSystemDependency defines a constructor for the ErrSystemDependency error
// type.
func NewSystemDependency(msg string) error {
	return ErrSystemDependency{BaseError{msg}}
}

// NewMissingKubernetesResource defines a constructor for the
// ErrMissingKubernetesResource error type.
func NewMissingKubernetesResource(msg string) error {
	return ErrMissingKubernetesResource{BaseError{msg}}
}

// NewResourceStatusDependency defines a constructor for the
// ErrResourceStatusDependency error type.
func NewResourceStatusDependency(msg string) error {
	return ErrResourceStatusDependency{BaseError{msg}}
}

// NewResourceConfigurationDependency defines a constructor for the
// ErrResourceStatusDependency error type.
func NewResourceConfigurationDependency(msg string) error {
	return ErrResourceConfigurationDependency{BaseError{msg}}
}

// NewUnlockError defines a constructor for the UnlockError error type.
func NewUnlockError(hostname string, in error) error {
	msg := fmt.Sprintf("Failed to unlock %s: %s", hostname, extractFaultString(in))
	return ErrUnlockError{BaseError{msg}}
}

// NewUserDataError defines a constructor for the ErrUserDataError error type.
func NewUserDataError(msg string) error {
	return ErrUserDataError{BaseError{msg}}
}

// NewValidationError defines a constructor for the ValidationError error type.
func NewValidationError(msg string) error {
	return ValidationError{BaseError{msg}}
}

// NewHTTPSClientRequired defines a constructor for the HTTPClientRequired error
// type.
func NewHTTPSClientRequired(msg string) error {
	return HTTPSClientRequired{BaseError{msg}}
}

// NewChangeAfterInSync defines a constructor for the ChangeAfterReconciled error
// type.
func NewChangeAfterInSync(msg string) error {
	return ChangeAfterReconciled{BaseError{msg}}
}

// NewPlatformNetworkReconciliationError defines a constructor for the
// PlatformNetworkReconciliationError error type.
func NewPlatformNetworkReconciliationError(msg string) error {
	return PlatformNetworkReconciliationError{BaseError{msg}}
}

// NewHostNotifyError defines a constructor for the
// HostNotifyError error type.
func NewHostNotifyError(msg string) error {
	return HostNotifyError{BaseError{msg}}
}
