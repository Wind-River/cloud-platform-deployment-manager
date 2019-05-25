/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package common

// BaseError defines the common error reporting struct for all other errors
// defined in this package
type BaseError struct {
	message string
}

// ErrSystemDependency defines an error to be used when reporting that the
// system itself or a set of multiple resources are not in the correct state to
// proceed with an operation.
type ErrSystemDependency struct {
	BaseError
}

// Error returns the message associated with an error of this type.
func (in ErrSystemDependency) Error() string {
	return in.message
}

// ErrMissingKubernetesResource defines an error to be used when reporting that
// an operation is unable to find a required resource from the
// kubernetes API.  This error is not intended for system resources that are
// missing.  For those use ErrMissingSystemResource
type ErrMissingKubernetesResource struct {
	BaseError
}

// Error returns the message associated with an error of this type.
func (in ErrMissingKubernetesResource) Error() string {
	return in.message
}

// ErrResourceStatusDependency defines an error to be used when reporting that
// an operation is unable to continue because a resource is not in the correct
// state.
type ErrResourceStatusDependency struct {
	BaseError
}

// Error returns the message associated with an error of this type.
func (in ErrResourceStatusDependency) Error() string {
	return in.message
}

// ErrUserDataError defines an error to be used when reporting that an operation
// is unable to continue because the requested configuration is incorrect or
// incomplete.
type ErrUserDataError struct {
	BaseError
}

// Error returns the message associated with an error of this type.
func (in ErrUserDataError) Error() string {
	return in.message
}

// ValidationError defines a new error type used to differentiate data
// validation errors from other types of errors.
type ValidationError struct {
	BaseError
}

// Error defines the struct to string conversion function specific to the
// controller error struct.
func (in ValidationError) Error() string {
	return in.message
}

// HTTPSClientRequired defines a new error type used to signal that a
// a configuration changes requires an HTTPS URL before continuing.
type HTTPSClientRequired struct {
	BaseError
}

// Error defines the struct to string conversion function specific to the
// controller error struct.
func (in HTTPSClientRequired) Error() string {
	return in.message
}

// ChangeAfterInSync defines a new error type used to signal that a
// a configuration changes was received after the resource has already been
// synchronized with the system state.
type ChangeAfterInSync struct {
	BaseError
}

// Error defines the struct to string conversion function specific to the
// controller error struct.
func (in ChangeAfterInSync) Error() string {
	return in.message
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
	return ErrResourceStatusDependency{BaseError{msg}}
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

// NewChangeAfterInSync defines a constructor for the ChangeAfterInSync error
// type.
func NewChangeAfterInSync(msg string) error {
	return ChangeAfterInSync{BaseError{msg}}
}
