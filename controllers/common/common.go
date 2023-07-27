/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2023 Wind River Systems, Inc. */

package common

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/gophercloud/gophercloud"
	perrors "github.com/pkg/errors"
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/api/v1"
	"github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	// RetryImmediate should be used whenever a known transient error is
	// detected and there is a very likely that retrying immediately will
	// succeed.  For example,
	RetryImmediate = reconcile.Result{Requeue: true, RequeueAfter: time.Second}

	// RetrySystemNotReady should be used whenever a controller needs to wait
	// for the system controller to finish its reconcile task.  The system
	// controller kicks the other controllers when it has finish so there
	// is no need to automatically requeue these events.
	RetrySystemNotReady = reconcile.Result{Requeue: false}

	// RetryCephPrimaryGroupNotReady should be used whenever a storage node needs to wait
	// for the ceph primary storage group to finish its reconcile task.
	RetryCephPrimaryGroupNotReady = reconcile.Result{Requeue: true}

	// RetryMissingClient should be used for any object reconciliation that
	// fails because of the platform client is missing or was reset.  The system
	// controller is responsible for re-creating the client and it will kick
	// the other controllers once it has re-established a connection to the
	// target system.
	RetryMissingClient = reconcile.Result{Requeue: false}

	// RetryTransientError should be used for any object reconciliation that
	// fails because of a transient error and needs to be re-attempted at a
	// future time.
	RetryTransientError = reconcile.Result{Requeue: true, RequeueAfter: 20 * time.Second}

	// RetryUserError should be used for any errors caught after an API request
	// that is likely due to data validation errors.  These could theoretically
	// not retry and just sit and wait for the user to correct the error, but
	// to mitigate against dependency errors or transient errors we will retry.
	RetryUserError = reconcile.Result{Requeue: true, RequeueAfter: time.Minute}

	// RetryValidationError should be used for any errors resulting from an
	// upfront validation error.  There is no point in trying again since the
	// data is invalid.  Just wait for the user to correct the issue.
	RetryValidationError = reconcile.Result{Requeue: false}

	// RetryServerError should be used for any errors caught after an API
	// request that is likely due to internal server errors.  These could
	// theoretically not retry and just sit and wait for the user to correct the
	// error, but to mitigate against dependency errors or transient errors we
	// will retry.
	RetryServerError = reconcile.Result{Requeue: true, RequeueAfter: time.Minute}

	// RetryNetworkError should be used for any DNS resolution errors.  There
	// is a good chance that these errors will persist for a while until the
	// user intervenes so slow down retry attempts.
	RetryResolutionError = reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Minute}

	// RetryNetworkError should be used for any errors caught after a API
	// request that is likely due to network errors.  This could happen
	// because of a misconfiguration of the endpoint URL or whenever the system
	// becomes temporarily unreachable.  We need to retry until the system
	// becomes reachable.  Since the most likely explanation is that the
	// active controller was rebooted then it makes sense to keep retrying
	// frequently because it will come back relatively quickly.
	// TODO(alegacy): consider backing off using a rate limiter queue.
	RetryNetworkError = reconcile.Result{Requeue: true, RequeueAfter: 15 * time.Second}

	// RetryNever is used when the reconciler will be triggered by a separate
	// mechanism and no retry is necessary.
	RetryNever = reconcile.Result{Requeue: false}

	// Properties contained on System resource
	SystemProperties = map[string]interface{}{
		"certificates":      nil,
		"contact":           nil,
		"description":       nil,
		"dnsServers":        nil,
		"latitude":          nil,
		"license":           nil,
		"location":          nil,
		"longitude":         nil,
		"ntpServers":        nil,
		"ptp":               []string{"mode", "transport", "mechanism"},
		"serviceParameters": nil,
		"storage":           []string{"filesystems", "drbd", "backends"},
		"vswitchType":       nil,
	}

	// Properties contained on Host resource
	HostProperties = map[string]interface{}{
		"addresses":            []string{"address", "interface", "prefix"},
		"administrativeState":  nil,
		"appArmor":             nil,
		"base":                 nil,
		"boardManagement":      []string{"address", "credentials"},
		"bootDevice":           nil,
		"bootMAC":              nil,
		"clockSynchronization": nil,
		"console":              nil,
		"hwSettle":             nil,
		"installOutput":        nil,
		"interfaces":           []string{"bond", "ethernet", "vf", "vlan"},
		"labels":               nil,
		"location":             nil,
		"maxCPUMHzConfigured":  nil,
		"memory":               nil,
		"personality":          nil,
		"powerOn":              nil,
		"processors":           nil,
		"provisioningMode":     nil,
		"ptpInstances":         nil,
		"rootDevice":           nil,
		"routes":               []string{"gateway", "interface", "metric", "prefix", "subnet"},
		"storage":              []string{"filesystems", "monitor", "osds", "volumeGroups"},
		"subfunctions":         nil,
	}
)

// Constant for processLines and searchParameters when gathering the Delta config
const ParentFound = "parent_found"

// Common event record reasons
const (
	ResourceCreated    = "Created"
	ResourceUpdated    = "Updated"
	ResourceDeleted    = "Deleted"
	ResourceWait       = "Wait"
	ResourceDependency = "Dependency"
)

var logCommon = log.Log.WithName("commmon")

func FormatStruct(obj interface{}) string {
	buf, _ := json.Marshal(obj)
	return string(buf)
}

func CompareStructs(a, b interface{}) bool {
	bufferA, _ := json.Marshal(a)
	bufferB, _ := json.Marshal(b)
	return string(bufferA) == string(bufferB)
}

// ReconcilerErrorHandler defines the interface type associated to any
// reconciler error handler.
type ReconcilerErrorHandler interface {
	HandleReconcilerError(request reconcile.Request, in error) (reconcile.Result, error)
}

// ErrorHandler is the common implementation of the ReconcilerErrorHandler
// interface.
type ErrorHandler struct {
	logr.Logger
	manager.CloudManager
}

// HandleReconcilerError is the common error handler implementation for all
// controllers.  It is responsible for looking at the type of error that was
// caught and determine what the best resolution might be.
func (h *ErrorHandler) HandleReconcilerError(request reconcile.Request, in error) (result reconcile.Result, err error) {
	resetClient := true

	// We use wrapped errors throughout the system so make sure we are looking
	// at the initial error before determining what actually went wrong.
	cause := perrors.Cause(in)

	switch cause.(type) {
	case gophercloud.ErrDefault400, gophercloud.ErrDefault403,
		gophercloud.ErrDefault404, gophercloud.ErrDefault405:
		// These errors are resource based errors.  This means we successfully
		// submitted the request but the server rejected it therefore the client
		// is still valid.  There is likely a problem with the data provided
		// by the user so wait for the user to correct the data.  Retrying is
		// pointless
		resetClient = false
		result = RetryUserError
		err = nil

		h.Error(in, "user error", "request", request)

	case gophercloud.ErrDefault500, gophercloud.ErrDefault503:
		// These errors are server based errors.  This means we successfully
		// submitted the request but the server encountered an unexpected or
		// unhandled exception
		resetClient = false
		result = RetryServerError
		err = nil

		h.Error(in, "server error", "request", request)

	case *errors.StatusError:
		// These errors are rest client errors from client-go.
		resetClient = false
		err = nil

		if strings.Contains(cause.Error(), "object has been modified") {
			// This is likely a status update conflict so immediately retry.
			result = RetryImmediate
			h.Info("status update conflict", "request", request)
		} else {
			result = RetryTransientError
			h.Error(in, "status error", "request", request)
		}

	case *url.Error:
		// These errors are networking type errors.  We failed to reach or
		// connect to the server.  Reset the client in all cases
		urlError := cause.(*url.Error)

		result = RetryNetworkError
		err = nil

		if opError, ok := urlError.Err.(*net.OpError); ok {
			if _, ok := opError.Err.(*net.DNSError); ok {
				// For this specific error we know that more time will be
				// needed for the user to intervene so use a longer delay.
				result = RetryResolutionError
				h.Error(in, "resolution error", "request")
				break
			}

		} else if strings.Contains(urlError.Error(), manager.HTTPSNotEnabled) {
			h.Info("HTTPS request was sent to an non HTTPS system")

			// The system controller will need to deal with this error when
			// it attempts to rebuild the client.
		}

		h.Error(in, "URL error", "request", request)

	case HTTPSClientRequired:
		// These errors are generated when the system controllers discovers
		// that a requires that HTTPS be enabled first.
		result = RetryTransientError
		err = nil

		h.Error(in, "HTTPS client required", "request", request)

	case ValidationError, ChangeAfterReconciled:
		// These errors are data validation errors.  There is likely a problem
		// with the data provided by the user so wait for the user to correct
		// the data.  Retrying is pointless.
		resetClient = false
		result = RetryValidationError
		err = nil

		h.Error(in, "validation error", "request", request)

	case ErrSystemDependency, ErrResourceStatusDependency:
		// These errors are transient errors.  Resources must be in stable
		// states before reconciling changes therefore we need to wait until
		// they settle before continuing.
		resetClient = false
		result = RetryTransientError
		err = nil

		h.Error(in, "resource status error", "request", request)

	case manager.ClientError, ErrUserDataError,
		starlingxv1.ErrMissingSystemResource, ErrMissingKubernetesResource:
		// These errors are user data errors.  Usually a reference to a
		// non-existent resource.
		resetClient = false
		result = RetryUserError
		err = nil

		h.Error(in, "user data error", "request", request)

	case manager.WaitForMonitor:
		// These errors are explicit wait states within a reconciler.  If such
		// an error is used then the reconciler wants to stop and wait for its
		// monitor to force a new reconcilable event.
		resetClient = false
		result = RetryNever
		err = nil

		h.Error(in, "waiting for host monitor", "request", request)

	default:
		resetClient = false

		if !errors.IsNotFound(cause) {
			h.Error(in, "an unhandled error occurred", "type", reflect.TypeOf(cause))
			result = RetryTransientError
			err = in
		} else {
			// A request to the kubernetes client failed because of a missing
			// resource.  Assume that a user resource is not installed or
			// visible yet and try again.
			result = RetryUserError
			err = nil

			h.Error(in, "missing dependency", "request", request)
		}
	}

	if resetClient {
		if h.CloudManager.GetPlatformClient(request.Namespace) != nil {
			h.Info("resetting platform client")
			err2 := h.CloudManager.ResetPlatformClient(request.Namespace)
			if err2 != nil {
				h.Error(err2, "failed to reset platform client")
			}
		}
	}

	return result, err
}

// ReconcilerEventLogger is an interface that is intended to allow specialized
// behavior when generating an event.
type ReconcilerEventLogger interface {
	NormalEvent(object runtime.Object, reason string, messageFmt string, args ...interface{})
	WarningEvent(object runtime.Object, reason string, messageFmt string, args ...interface{})
}

// EventLogger is an implementation of a ReconcilerEventLogger.  Its purpose is
// to simultaneously generate a log with every event and to prefix each event
// message with the object name.
type EventLogger struct {
	record.EventRecorder
	logr.Logger
}

// event is a method used to generate a log and an event for a given set of
// message, event type, and reason.
func (in *EventLogger) event(object runtime.Object, eventtype string, logLevel int, reason string, messageFmt string, args ...interface{}) {
	accessor := meta.NewAccessor()
	name, err := accessor.Name(object)
	if err != nil {
		name = "unknown"
	}
	msg := fmt.Sprintf("%s: %s", name, fmt.Sprintf(messageFmt, args...))
	in.Logger.V(logLevel).Info(msg)
	in.EventRecorder.Eventf(object, eventtype, reason, msg)
}

// NormalEvent generates a log and event for a "normal" event.
func (in *EventLogger) NormalEvent(object runtime.Object, reason string, messageFmt string, args ...interface{}) {
	// logLevel is set to the normal level (0) so that we can see these
	// in the log stream rather than having to look at the events.
	in.event(object, v1.EventTypeNormal, 0, reason, messageFmt, args...)
}

// WarningEvent generates a log and event for a "warning" event.  The intent is
// that this should only be used when declaring a reconciler error... all other
// events should use "NormalEvent".
func (in *EventLogger) WarningEvent(object runtime.Object, reason string, messageFmt string, args ...interface{}) {
	// logLevel is set to the debug level (1) because WarningEvent should be
	// accompanied by a reconciler error which has its own log generated.
	in.event(object, v1.EventTypeWarning, 1, reason, messageFmt, args...)
}

func GetDeltaString(spec interface{}, current interface{}, parameters map[string]interface{}) (string, error) {

	specBytes, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}

	currentBytes, err := json.Marshal(current)
	if err != nil {
		return "", err
	}

	var specData map[string]interface{}
	var currentData map[string]interface{}

	err = json.Unmarshal([]byte(specBytes), &specData)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal([]byte(currentBytes), &currentData)
	if err != nil {
		return "", err
	}

	diff := cmp.Diff(specData, currentData)
	deltaString := collectDiffValues(diff, parameters)
	deltaString = strings.TrimSuffix(deltaString, "\n")
	return deltaString, nil
}

/* CollectDiffValues collects and returns the diff values from the given diff string.
 The function returns lines starting with '+' or '-' that represent the differences,
 and will provide the parent hierarchy for that line based on the given parameters.

 Output example:

storage:
	"filesystems":
-		"size":		5,
+		"size":		10,

*/
func collectDiffValues(diff string, parameters map[string]interface{}) string {
	var diffLines []string
	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		line = strings.Join(strings.Fields(trimmedLine), "\t\t")
		processedLine := removeDataTypes(line)
		diffLines = append(diffLines, processedLine)
	}

	delta := processLines(diffLines, parameters)
	return delta.String()
}

/*
removeDataTypes removes data types and specific interfaces from the given string.
The modified string with data types and specific interfaces removed.
Example:

  input: "float64(1500)"
  output: "1500"
*/
func removeDataTypes(line string) string {
	// Define the regular expression to match and capture data types
	re := regexp.MustCompile(`\b(string|float64|bool|int|)\(([^)]*)\)`)
	noDataTypes := re.ReplaceAllString(line, "$2")

	// Define the regular expression to match and remove specific interfaces
	re = regexp.MustCompile(`(map\[string\]interface\{\}|\[\]interface\{\})`)
	noInterfaces := re.ReplaceAllString(noDataTypes, "$2")
	return noInterfaces
}

// processLines processes the diff lines and generates the delta configuration.
func processLines(lines []string, parameters map[string]interface{}) strings.Builder {
	var delta strings.Builder
	lastParent := "-"
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		// Check if the line starts with a "+" or "-" indicating an added or removed configuration.
		if strings.HasPrefix(trimmedLine, "+") || strings.HasPrefix(trimmedLine, "-") {
			// Search for the parent and sub-parameters of the line in the given properties.
			parent := searchParameters(lines, i, parameters)
			if parent == ParentFound {
				// If the line represents a parameter itself, add it directly to the delta.
				trimmedLine := strings.TrimSpace(trimmedLine)
				line = strings.Join(strings.Fields(trimmedLine), "  ")
				delta.WriteString(line)
				delta.WriteString("\n")
				continue
			}
			// If the parent has changed, add a newline and the parent to the delta.
			if parent != lastParent {
				delta.WriteString("\n")
				delta.WriteString(parent)
				lastParent = parent
			}
			// Add the line to the delta.
			delta.WriteString(trimmedLine)
			delta.WriteString("\n")
		}
	}

	return delta
}

// searchParameters searches for the parent and sub-parameters of a given line in the given resource properties.
// It iterates over the lines starting from the given line number and goes upwards to find the relevant information.
// Parameters:
//   - lines: A slice of strings representing the lines of the diff.
//   - lineNumber: The index of the line being processed.
//   - parameters: A map of resource properties with their corresponding values.
// Returns:
//   - A string representing the hierarchy of the parent and sub-parameters, or "param_found" if the line represents a parameter itself.
//     The hierarchy is constructed in the format: "parent:\n\t"sub-parameter":\n".
func searchParameters(lines []string, lineNumber int, parameters map[string]interface{}) string {
	var result string

	for i := lineNumber; i >= 0; i-- {
		line := lines[i]

		// Check if the line matches any parameter or sub-parameter in the system properties.
		for param, subParams := range parameters {
			if subParams != nil {
				subParamsList, ok := subParams.([]string)
				if ok {
					for _, subParam := range subParamsList {
						// If a sub-parameter is found in the line, construct the corresponding hierarchy.
						if strings.Contains(line, subParam) {
							if i == lineNumber {
								// If the line represents the sub-parameter itself, return the parent parameter.
								return fmt.Sprintf("%s:\n", param)
							}
							// Append the sub-parameter to the hierarchy.
							result += fmt.Sprintf("%s:\n\t\"%s\":\n", param, subParam)
							return result
						}
					}
				}
			}
		}

		// Check if the line matches any parameter in the system properties.
		for param := range parameters {
			if strings.Contains(line, param) {
				if i == lineNumber {
					// If the line represents the parameter itself, indicate it with "param_found".
					return ParentFound
				}
				// Return the parent parameter.
				return fmt.Sprintf("%s:\n", param)
			}
		}
	}

	return result
}

// Sync interface name for ethernet
// When N3000 interface is used, it is possible to change name after unlocked.
// This function is to copy interface name from current to profile if uuid is the same
// to avoid configuration difference because of names.
func SyncIFNameByUuid(profile *starlingxv1.HostProfileSpec, current *starlingxv1.HostProfileSpec) {

	if_current := current.Interfaces.Ethernet
	if_profile := profile.Interfaces.Ethernet
	for idx_current := range if_current {
		info_current := if_current[idx_current].CommonInterfaceInfo
		port_current := if_current[idx_current].Port.Name
		for idx_profile := range if_profile {
			info_profile := if_profile[idx_profile].CommonInterfaceInfo
			if info_profile.UUID == info_current.UUID &&
				info_profile.Name != info_current.Name {
				logCommon.Info("Ethernet name sync", "profile", info_profile.Name,
					"current", info_current.Name, "uuid", info_profile.UUID)
				if_profile[idx_profile].CommonInterfaceInfo.Name = info_current.Name
				if_profile[idx_profile].Port.Name = port_current
			}
		}
	}
}
