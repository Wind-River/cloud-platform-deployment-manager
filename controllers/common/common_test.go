/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2023 Wind River Systems, Inc. */

package common

import (
	errpkg "errors"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		Context("when error is errors.StatusError", func() {
			It("should log error and return RetryTransientError", func() {
				testError := &errors.StatusError{
					ErrStatus: metav1.Status{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Status",
						},
						Status:  "Failure",
						Code:    404,
						Reason:  "NotFound",
						Message: "Resource not found",
					},
				}
				result, _ := testHandler.HandleReconcilerError(request, testError)

				Expect(result).To(Equal(RetryTransientError))
			})
		})
		Context("when error is ValidationError", func() {
			It("should log error and return RetryValidationError", func() {
				testError := ValidationError{BaseError{"error msg"}}
				result, _ := testHandler.HandleReconcilerError(request, testError)

				Expect(result).To(Equal(RetryValidationError))
			})
		})
		Context("when error is ErrMissingSystemResource", func() {
			It("should log error and return RetryUserError", func() {
				testError := starlingxv1.ErrMissingSystemResource{}
				result, _ := testHandler.HandleReconcilerError(request, testError)

				Expect(result).To(Equal(RetryUserError))
			})
		})
		Context("when error is some text error", func() {
			It("should log error and return RetryTransientError", func() {
				testError := errpkg.New("error msg")
				result, _ := testHandler.HandleReconcilerError(request, testError)

				Expect(result).To(Equal(RetryTransientError))
			})
		})
	})

	Describe("Test removeDataTypes function", func() {
		Context("When the constant of dataType float64 is given", func() {
			It("It returns the constant without dataType", func() {
				input1 := "float64(1500)"
				expected1 := "1500"
				out1 := removeDataTypes(input1)
				Expect(out1).To(Equal(expected1))
			})
		})
	})

	Describe("Test searchParameters", func() {
		Context("When the non empty lines are given", func() {
			It("Should expect result as follows", func() {

				// Define test input data
				lines := []string{
					"line1",
					"line2",
					"line3",
					"subparam1",
					"param1",
					"subparam2",
					"line5",
					"line6",
				}

				parameters := map[string]interface{}{
					"param1": []string{"subparam1", "subparam2"},
					"param2": nil,
				}

				lineNumber := 4 // Index of "param1" in the lines slice
				// Test case: Parent found
				expected := "parent_found"
				result := searchParameters(lines, lineNumber, parameters)
				Expect(expected).To(Equal(result))

				// Test case: Sub-parameter found
				lineNumber = 3 // Index of line containing "subparam1"
				expected = "param1:\n"
				result = searchParameters(lines, lineNumber, parameters)
				Expect(expected).To(Equal(result))

				// Test case: Sub-parameter found in above lines
				lineNumber = 6 // Index of line containing "subparam1"
				expected = "param1:\n\t\"subparam2\":\n"
				result = searchParameters(lines, lineNumber, parameters)
				Expect(expected).To(Equal(result))

				// Test case: Parameter not found
				lineNumber = 0 // Index of first line
				expected = ""
				result = searchParameters(lines, lineNumber, parameters)
				Expect(expected).To(Equal(result))
			})
		})
	})

	Describe("Test SyncIFNameByUuid", func() {
		Context("When uuid is the same", func() {
			It("Should copy interface name from current to profile", func() {
				profile := &starlingxv1.HostProfileSpec{
					Interfaces: &starlingxv1.InterfaceInfo{
						Ethernet: starlingxv1.EthernetList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: "ProfEthName",
									UUID: "EthUUID",
								},
								Port: starlingxv1.EthernetPortInfo{},
							},
						},
					},
				}
				ethPortName := "EthPortName"
				currEthName := "CurrEthName"
				current := &starlingxv1.HostProfileSpec{
					Interfaces: &starlingxv1.InterfaceInfo{
						Ethernet: starlingxv1.EthernetList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: currEthName,
									UUID: "EthUUID",
								},
								Port: starlingxv1.EthernetPortInfo{
									Name: ethPortName,
								},
							},
						},
					},
				}
				SyncIFNameByUuid(profile, current)
				Expect(profile.Interfaces.Ethernet[0].Port.Name).To(Equal(ethPortName))
				Expect(profile.Interfaces.Ethernet[0].CommonInterfaceInfo.Name).To(Equal(currEthName))

			})
		})
		Context("if uuid is not same", func() {
			It("Shouldnt copy interface name from current to profile", func() {
				profPortName := "profPortName"
				profEthName := "profEthName"
				profile := &starlingxv1.HostProfileSpec{
					Interfaces: &starlingxv1.InterfaceInfo{
						Ethernet: starlingxv1.EthernetList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: profEthName,
									UUID: "EthUUID1",
								},
								Port: starlingxv1.EthernetPortInfo{
									Name: profPortName,
								},
							},
						},
					},
				}
				currPortName := "EthPortName"
				currEthName := "CurrEthName"
				current := &starlingxv1.HostProfileSpec{
					Interfaces: &starlingxv1.InterfaceInfo{
						Ethernet: starlingxv1.EthernetList{
							{
								CommonInterfaceInfo: starlingxv1.CommonInterfaceInfo{
									Name: currEthName,
									UUID: "EthUUID",
								},
								Port: starlingxv1.EthernetPortInfo{
									Name: currPortName,
								},
							},
						},
					},
				}
				SyncIFNameByUuid(profile, current)
				Expect(profile.Interfaces.Ethernet[0].Port.Name).To(Equal(profPortName))
				Expect(profile.Interfaces.Ethernet[0].CommonInterfaceInfo.Name).To(Equal(profEthName))

			})
		})
	})
	Describe("Test processLines", func() {
		Context("When there given non empty lines", func() {
			It("Should gather delta config", func() {
				lines := []string{"line1", "line2", "line3"}
				parameters := map[string]interface{}{}
				expectedResult := ""

				result := processLines(lines, parameters)
				Expect(expectedResult).To(Equal(result.String()))

				// Define test input data
				lines = []string{
					"+ line1",
					"- line2",
					"param1",
					"+ line3",
					"- line4",
				}
				parameters = map[string]interface{}{
					"param1": nil,
					"param2": []string{"subparam1", "subparam2"},
				}

				expectedResult = "\n+ line1\n- line2\n\nparam1:\n+ line3\n- line4\n"
				result = processLines(lines, parameters)
				Expect(expectedResult).To(Equal(result.String()))

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
