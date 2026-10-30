/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2026 Wind River Systems, Inc. */

package common

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	"github.com/gophercloud/gophercloud"
	perrors "github.com/pkg/errors"
)

// faultRe matches the faultstring embedded (and JSON escaped) in the body of a
// rejected StarlingX API request.
var faultRe = regexp.MustCompile(`faultstring.*?:.*?"(.*?)\\`)

// responseCodeType is the type of the gophercloud error carrying the details
// of a rejected (non 2xx) API response.
var responseCodeType = reflect.TypeOf(gophercloud.ErrUnexpectedResponseCode{})

// apiResponseError returns the HTTP response details carried by a gophercloud
// error.  The error is unwrapped first so that errors annotated with
// perrors.Wrap are also recognized.  The second return value is false if the
// error was not produced by a rejected (non 2xx) API response.
//
// gophercloud reports a rejected response either as an ErrUnexpectedResponseCode
// or as one of the ErrDefaultXXX types (ErrDefault400, ErrDefault409, ...),
// which all embed it.  The embedded struct is looked up generically so that
// every status-specific type is handled without listing each one.
func apiResponseError(err error) (*gophercloud.ErrUnexpectedResponseCode, bool) {
	if err == nil {
		return nil, false
	}

	v := reflect.ValueOf(perrors.Cause(err))
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, false
	}

	if v.Type() != responseCodeType {
		// Look for the embedded ErrUnexpectedResponseCode of an ErrDefaultXXX.
		field, ok := v.Type().FieldByName(responseCodeType.Name())
		if !ok || !field.Anonymous || field.Type != responseCodeType {
			return nil, false
		}
		v = v.FieldByIndex(field.Index)
	}

	resp := v.Interface().(gophercloud.ErrUnexpectedResponseCode)
	return &resp, true
}

// IsAPIResponseError returns true if the error (or the error it wraps) was
// produced because the system API rejected a request with a non 2xx status.
func IsAPIResponseError(err error) bool {
	_, ok := apiResponseError(err)
	return ok
}

// APIErrorStatusCode returns the HTTP status code associated to a rejected
// system API request, or 0 if the error is not an API response error.
func APIErrorStatusCode(err error) int {
	if resp, ok := apiResponseError(err); ok {
		return resp.Actual
	}
	return 0
}

// parseFaultBody extracts a human readable message from the body of a rejected
// system API request.  The StarlingX APIs encode their errors as:
//
//	{"error_message": "{\"debuginfo\": null, \"faultcode\": \"Client\", \"faultstring\": \"<reason>\"}"}
//
// Other OpenStack style bodies such as {"error": {"message": "..."}} or
// {"message": "..."} are also handled.  An empty string is returned if no
// message could be extracted.
func parseFaultBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return ""
	}

	var outer map[string]interface{}
	if err := json.Unmarshal([]byte(text), &outer); err == nil {
		if msg, ok := outer["error_message"].(string); ok {
			var inner map[string]interface{}
			if err := json.Unmarshal([]byte(msg), &inner); err == nil {
				if fault, ok := inner["faultstring"].(string); ok && fault != "" {
					return fault
				}
			}
			if msg != "" {
				return msg
			}
		}
		if fault, ok := outer["faultstring"].(string); ok && fault != "" {
			return fault
		}
		if nested, ok := outer["error"].(map[string]interface{}); ok {
			if msg, ok := nested["message"].(string); ok && msg != "" {
				return msg
			}
		}
		if msg, ok := outer["message"].(string); ok && msg != "" {
			return msg
		}
	}

	if result := faultRe.FindStringSubmatch(text); len(result) > 0 {
		return result[1]
	}

	return text
}

// APIErrorReason returns a concise, human readable explanation of why an
// operation failed.  For rejected system API requests the reason reported by
// the server (i.e., the sysinv faultstring) is returned instead of the raw
// response body.  For any other error the error string is returned.
func APIErrorReason(err error) string {
	if err == nil {
		return ""
	}

	if resp, ok := apiResponseError(err); ok {
		if reason := stripRemoteTraceback(parseFaultBody(resp.Body)); reason != "" {
			return reason
		}
		return perrors.Cause(err).Error()
	}

	reason := err.Error()
	if result := faultRe.FindStringSubmatch(reason); len(result) > 0 {
		reason = result[1]
	}
	return stripRemoteTraceback(reason)
}

// remoteTracebackMarker is the marker of a Python traceback.  sysinv-api does
// not install its NoExceptionTracebackHook, so exceptions raised by
// sysinv-conductor and relayed over RPC are returned to the client with the
// remote traceback appended to the faultstring.
const remoteTracebackMarker = "Traceback (most recent call last):"

// stripRemoteTraceback removes any Python traceback (and trailing whitespace)
// appended to a server provided error message.
func stripRemoteTraceback(msg string) string {
	if idx := strings.Index(msg, remoteTracebackMarker); idx >= 0 {
		msg = msg[:idx]
	}
	return strings.TrimSpace(msg)
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

// LogAPIError logs an error returned by a system API request.  A request the
// system API rejected is an expected outcome that is retried, so it is logged
// at the informational level with only the reason reported by the server (the
// sysinv faultstring), the operation and the status code, rather than at the
// error level with the raw response body and a stack trace.  Any other error is
// logged as an error.
func LogAPIError(log logr.Logger, err error, msg string, keysAndValues ...interface{}) {
	if IsAPIResponseError(err) {
		kv := append([]interface{}{}, keysAndValues...)
		if op := APIErrorOperation(err); op != "" {
			kv = append(kv, "operation", op)
		}
		if code := APIErrorStatusCode(err); code != 0 {
			kv = append(kv, "status", code)
		}
		kv = append(kv, "reason", APIErrorReason(err))
		log.Info(msg, kv...)
		return
	}
	log.Error(err, msg, keysAndValues...)
}
