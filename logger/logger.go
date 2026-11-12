/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2026 Wind River Systems, Inc. */

// Package logger provides a thin wrapper around logr.Logger that knows how to
// log errors returned by the StarlingX system API.  A request the system API
// rejected (a non 2xx response) is an expected outcome that the reconcilers
// retry, so logging it at the error level with the raw response body and a
// stack trace is noise.  Logger.Error detects such errors and instead logs
// them at the informational level with only the reason reported by the server
// (the sysinv faultstring), the HTTP status and the operation.  Every other
// error is logged as an error, exactly like logr.Logger would.
//
// Because Logger embeds logr.Logger, it is a drop-in replacement: call sites
// keep using log.Info/log.Error/log.V/... as before, and only need to wrap
// their logger once with New.
package logger

import (
	"github.com/go-logr/logr"
	"github.com/wind-river/cloud-platform-deployment-manager/common"
)

// Logger wraps a logr.Logger and classifies system API errors passed to Error.
type Logger struct {
	logr.Logger
}

// New wraps an existing logr.Logger.
func New(log logr.Logger) Logger {
	return Logger{log}
}

// Error logs err.  A request rejected by the system API is logged at info
// level with the server reason, status and operation (no stack trace); any
// other error is logged at error level like logr.Logger.Error.
func (l Logger) Error(err error, msg string, keysAndValues ...interface{}) {
	if code := common.APIErrorStatusCode(err); code != 0 {
		kv := append([]interface{}{}, keysAndValues...)
		if op := common.APIErrorOperation(err); op != "" {
			kv = append(kv, "operation", op)
		}
		kv = append(kv, "status", code, "reason", common.APIErrorReason(err))
		l.Info(msg, kv...)
		return
	}
	// Call the embedded logger explicitly to avoid recursing into this method.
	l.Logger.Error(err, msg, keysAndValues...)
}

// WithName returns a Logger with the given name element appended, preserving
// the API-error classification.
func (l Logger) WithName(name string) Logger {
	return Logger{l.Logger.WithName(name)}
}

// WithValues returns a Logger with the given key/value pairs added, preserving
// the API-error classification.
func (l Logger) WithValues(keysAndValues ...interface{}) Logger {
	return Logger{l.Logger.WithValues(keysAndValues...)}
}
