/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2026 Wind River Systems, Inc. */

package common

import (
	"fmt"
	"net/http"
	"strings"

	perrors "github.com/pkg/errors"
	utils "github.com/wind-river/cloud-platform-deployment-manager/common"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Defines common log strings from commonly performed validation checks across
// all different reconcilers.
const (
	NoChangesAfterReconciled           = "configuration changes ignored after initial synchronization has completed"
	ChangedAllowedAfterReconciled      = "manual override; allowing configuration changes after initial synchronization"
	NoProvisioningAfterReconciled      = "resource provisioning ignored after initial synchronization has completed"
	ProvisioningAllowedAfterReconciled = "manual override; allowing resource provisioning after initial synchronization"
)

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

// ErrSystemAPI defines an error to be used when a request to a StarlingX system
// API (i.e., sysinv-api) failed or was rejected.  It carries the operation that
// was attempted, the HTTP status code returned and the reason reported by the
// server (the sysinv faultstring, without any remote traceback) so that the
// error can be classified and reported uniformly regardless of which API
// request produced it.
//
// Call sites do not need to create these explicitly; the reconciler error
// handler converts any rejected gophercloud request (other than authentication
// failures) into an ErrSystemAPI.
// Create one with NewSystemAPIError only to describe the operation more
// precisely or to override the default retry policy with WithRetry.
type ErrSystemAPI struct {
	// Operation describes the request that failed (e.g., "unlock host
	// controller-0").  May be empty.
	Operation string

	// StatusCode is the HTTP status code returned by the API, or 0 if no
	// valid response was received.
	StatusCode int

	// Reason is the human readable explanation reported by the server.
	Reason string

	// retry optionally overrides the retry policy derived from StatusCode.
	retry *reconcile.Result

	// cause is the original error returned by gophercloud.
	cause error
}

// Error implements the error interface.
func (in ErrSystemAPI) Error() string {
	msg := in.Reason
	if in.StatusCode != 0 {
		msg = fmt.Sprintf("%s (HTTP %d)", msg, in.StatusCode)
	}
	if in.Operation != "" {
		return fmt.Sprintf("failed to %s: %s", strings.TrimPrefix(in.Operation, "failed to "), msg)
	}
	return msg
}

// Unwrap returns the original API error so that errors.As/errors.Is can still
// inspect the underlying gophercloud error.
func (in ErrSystemAPI) Unwrap() error {
	return in.cause
}

// WithRetry returns a copy of the error which overrides the retry policy that
// would otherwise be derived from the HTTP status code.  For example, unlock
// rejections are usually transient and worth retrying sooner.
func (in ErrSystemAPI) WithRetry(result reconcile.Result) ErrSystemAPI {
	in.retry = &result
	return in
}

// Transient returns true if the request is likely to succeed if retried
// without any change to the configuration (i.e., request timeout, rate
// limiting, or no valid response received).
func (in ErrSystemAPI) Transient() bool {
	switch in.StatusCode {
	case 0, http.StatusRequestTimeout, http.StatusTooManyRequests:
		return true
	}
	return false
}

// ServerError returns true if the server failed to process the request.  Note
// that sysinv also reports some invalid input (e.g., a ValueError raised by a
// controller) as an HTTP 500.
func (in ErrSystemAPI) ServerError() bool {
	return in.StatusCode >= http.StatusInternalServerError
}

// Retry returns the retry policy for this error.  Unless overridden with
// WithRetry the policy is derived from the HTTP status code:
//   - no response, 408, 429: transient, retry soon
//   - 5xx: server error
//   - other 4xx (incl. sysinv 409 Conflict): rejected, the configuration
//     likely needs to be corrected
func (in ErrSystemAPI) Retry() reconcile.Result {
	switch {
	case in.retry != nil:
		return *in.retry
	case in.Transient():
		return RetryTransientError
	case in.ServerError():
		return RetryServerError
	default:
		return RetryUserError
	}
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

// NewSystemAPIError defines a constructor for the ErrSystemAPI error type.  The
// operation describes the request that was attempted; if empty, any context
// added to the error with perrors.Wrap is used instead.  A nil error can be
// passed when the API returned no error but also no usable result.
func NewSystemAPIError(operation string, in error) ErrSystemAPI {
	if existing, ok := perrors.Cause(in).(ErrSystemAPI); ok {
		if operation != "" {
			existing.Operation = operation
		}
		return existing
	}

	if operation == "" && in != nil {
		operation = utils.APIErrorOperation(in)
	}

	reason := "no valid response received from the system API"
	if in != nil {
		reason = utils.APIErrorReason(in)
	}

	return ErrSystemAPI{
		Operation:  operation,
		StatusCode: utils.APIErrorStatusCode(in),
		Reason:     reason,
		cause:      perrors.Cause(in),
	}
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
