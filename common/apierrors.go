/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2026 Wind River Systems, Inc. */

package common

import (
	"errors"
	"strings"

	perrors "github.com/pkg/errors"
)

// apiResponseError is the subset of the gophercloud StarlingX FaultError that
// carries the details of a rejected (non 2xx) API response.  FaultError is
// installed as the StarlingX client's default error context, so every rejected
// StarlingX request is returned as one, regardless of the status code.  The
// interface is matched with errors.As so that errors annotated with
// perrors.Wrap (which implements Unwrap) are also recognized.
type apiResponseError interface {
	error
	GetStatusCode() int
	GetBody() []byte
}

// extractAPIResponseError returns the gophercloud response error carried by
// err, or false if err was not produced by a rejected (non 2xx) StarlingX API
// response.
func extractAPIResponseError(err error) (apiResponseError, bool) {
	var respErr apiResponseError
	if errors.As(err, &respErr) {
		return respErr, true
	}
	return nil, false
}

// IsAPIResponseError returns true if the error (or the error it wraps) was
// produced because the system API rejected a request with a non 2xx status.
func IsAPIResponseError(err error) bool {
	_, ok := extractAPIResponseError(err)
	return ok
}

// APIErrorStatusCode returns the HTTP status code associated to a rejected
// system API request, or 0 if the error is not an API response error.
func APIErrorStatusCode(err error) int {
	if resp, ok := extractAPIResponseError(err); ok {
		return resp.GetStatusCode()
	}
	return 0
}

// APIErrorReason returns a concise, human readable explanation of why an
// operation failed.  For rejected system API requests the reason reported by
// the server (i.e., the sysinv faultstring) is returned: the gophercloud
// FaultError already extracts it from the response body and strips any remote
// Python traceback, so its Error() string is the reason.  For any other error
// the error string is returned.
func APIErrorReason(err error) string {
	if err == nil {
		return ""
	}
	if resp, ok := extractAPIResponseError(err); ok {
		return resp.Error()
	}
	return err.Error()
}

// APIErrorOperation returns the annotations added on top of the root cause of
// an error with perrors.Wrap (e.g., "failed to create route ...").  These
// describe the operation that was being attempted.  An empty string is
// returned if the error was not annotated.
func APIErrorOperation(err error) string {
	// Note: gophercloud errors embed the response body ([]byte) so they are
	// not comparable; check for a wrapper instead of comparing err to its
	// cause, which would panic.
	if _, wrapped := err.(interface{ Cause() error }); !wrapped {
		return ""
	}
	cause := perrors.Cause(err)
	if cause == nil {
		return ""
	}
	return strings.TrimSuffix(err.Error(), ": "+cause.Error())
}
