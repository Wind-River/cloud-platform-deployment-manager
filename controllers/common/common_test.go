/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2023 Wind River Systems, Inc. */

package common

import (
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Common utils", func() {
	Describe("Check return for HandleReconcilerError", func() {
		var testHandler *ErrorHandler
		var request reconcile.Request
		var sink *DummyLogSink
		var mockLogger logr.Logger
		BeforeEach(func() {
			sink = &DummyLogSink{infoCalled: false, errorCalled: false, message: ""}
			mockLogger = logr.New(sink)
			testHandler = &ErrorHandler{Logger: mockLogger}
			request = reconcile.Request{}
		})
		Context("when error is ErrResourceStatusDependency", func() {
			It("should log info and return RetryValidationError", func() {
				testError := ErrResourceStatusDependency{BaseError{"Test for status dependency"}}
				result, _ := testHandler.HandleReconcilerError(request, testError)

				Expect(result).To(Equal(RetryTransientError))
				Expect(sink.infoCalled).To(BeTrue())
				Expect(sink.errorCalled).To(BeFalse())
				Expect(sink.message).To(Equal("waiting for dependency status"))
			})
		})
		Context("when error is ErrResourceConfigurationDependency", func() {
			It("should log error and return RetryValidationError", func() {
				testError := ErrResourceConfigurationDependency{BaseError{"Test for config dependency"}}
				result, _ := testHandler.HandleReconcilerError(request, testError)

				Expect(result).To(Equal(RetryTransientError))
				Expect(sink.infoCalled).To(BeFalse())
				Expect(sink.errorCalled).To(BeTrue())
				Expect(sink.message).To(Equal("resource configuration error"))
			})
		})
		Context("when error is ErrResourceConfigurationDependency", func() {
			It("should log error and return RetryValidationError", func() {
				testError := manager.NewWaitForMonitor("Test for waiting monitor")
				result, _ := testHandler.HandleReconcilerError(request, testError)

				Expect(result).To(Equal(RetryNever))
				Expect(sink.infoCalled).To(BeTrue())
				Expect(sink.errorCalled).To(BeFalse())
				Expect(sink.message).To(Equal("waiting for host monitor to trigger another reconciliation"))
			})
		})
	})
})

type DummyLogSink struct {
	infoCalled  bool
	errorCalled bool
	message     string
}

func (l *DummyLogSink) Init(info logr.RuntimeInfo) {
}

func (l *DummyLogSink) Enabled(level int) bool {
	return true
}
func (l *DummyLogSink) Info(level int, msg string, keysAndValues ...interface{}) {
	l.infoCalled = true
	l.message = msg
}
func (l *DummyLogSink) Error(err error, msg string, keysAndValues ...interface{}) {
	l.errorCalled = true
	l.message = msg
}
func (l *DummyLogSink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return nil
}
func (l *DummyLogSink) WithName(name string) logr.LogSink {
	return nil
}
