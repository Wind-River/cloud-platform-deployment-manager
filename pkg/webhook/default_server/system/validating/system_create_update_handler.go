/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package validating

import (
	"context"
	"fmt"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"

	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/pkg/apis/starlingx/v1"
	corev1 "k8s.io/api/core/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

var log = logf.Log.WithName("webhook")

// Webhook response reasons
const AllowedReason string = "allowed to be admitted"

const (
	// Backend types
	file = "file"
	lvm  = "lvm"
	ceph = "ceph"
)

const (
	// Backend services
	glance         = "glance"
	cinder         = "cinder"
	nova           = "nova"
	swift          = "swift"
	rbdProvisioner = "rbd-provisioner"
)

var validBackendServices = map[string]map[string]bool{
	file: {
		glance: true,
	},

	lvm: {
		cinder: true,
	},

	ceph: {
		glance:         true,
		cinder:         true,
		nova:           true,
		swift:          true,
		rbdProvisioner: true,
	},
}

func init() {
	webhookName := "validating-create-update-system"
	if HandlerMap[webhookName] == nil {
		HandlerMap[webhookName] = []admission.Handler{}
	}
	HandlerMap[webhookName] = append(HandlerMap[webhookName], &SystemCreateUpdateHandler{})
}

// SystemCreateUpdateHandler handles System
type SystemCreateUpdateHandler struct {
	// API client reference
	Client client.Client

	// Decoder decodes objects
	Decoder types.Decoder
}

func (h *SystemCreateUpdateHandler) validateBackendServices(backendType string, services []string) (bool, string, error) {
	for _, s := range services {
		if !validBackendServices[backendType][s] {
			return false, fmt.Sprintf("%s service not allowed with %s backend.", s, backendType), nil
		}
	}

	return true, AllowedReason, nil
}

func (h *SystemCreateUpdateHandler) validateBackendAttributes(backend starlingxv1.StorageBackend) (bool, string, error) {
	if backend.PartitionSize != nil || backend.ReplicationFactor != nil {
		if backend.Type != ceph {
			return false, fmt.Sprintf("partitionSize and ReplicationFactor only permitted with %s backend", ceph), nil
		}
	}

	return true, AllowedReason, nil
}

func (h *SystemCreateUpdateHandler) validateStorageBackends(obj *starlingxv1.System) (bool, string, error) {
	var present = make(map[string]bool)

	for _, b := range *obj.Spec.Storage.Backends {
		if present[b.Type] {
			return false, fmt.Sprintf("backend services may only be specified once."), nil
		}

		if b.Services != nil {
			allowed, reason, err := h.validateBackendServices(b.Type, b.Services)
			if !allowed || err != nil {
				return allowed, reason, err
			}
		}

		allowed, reason, err := h.validateBackendAttributes(b)
		if !allowed || err != nil {
			return allowed, reason, err
		}

		present[b.Type] = true
	}

	return true, AllowedReason, nil
}

func (h *SystemCreateUpdateHandler) validateStorage(ctx context.Context, obj *starlingxv1.System) (bool, string, error) {
	if obj.Spec.Storage != nil && obj.Spec.Storage.Backends != nil {
		allowed, reason, err := h.validateStorageBackends(obj)
		if !allowed || err != nil {
			return allowed, reason, err
		}
	}

	return true, AllowedReason, nil
}

func (h *SystemCreateUpdateHandler) validateCertificates(ctx context.Context, obj *starlingxv1.System) (bool, string, error) {
	if obj.Spec.Certificates != nil {
		for _, c := range *obj.Spec.Certificates {
			secret := &corev1.Secret{}
			secretName := apitypes.NamespacedName{Name: c.Secret, Namespace: obj.ObjectMeta.Namespace}

			err := h.Client.Get(ctx, secretName, secret)
			if err != nil {
				return false, fmt.Sprintf("unable to retrieve %s secret %s", c.Type, secretName), err
			}
		}
	}

	return true, AllowedReason, nil
}

func (h *SystemCreateUpdateHandler) validatingSystemFn(ctx context.Context, obj *starlingxv1.System) (bool, string, error) {
	allowed, reason, err := h.validateStorage(ctx, obj)
	if !allowed || err != nil {
		return allowed, reason, err
	}

	allowed, reason, err = h.validateCertificates(ctx, obj)
	if !allowed || err != nil {
		return allowed, reason, err
	}

	return allowed, reason, err
}

var _ admission.Handler = &SystemCreateUpdateHandler{}

// Handle handles admission requests.
func (h *SystemCreateUpdateHandler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &starlingxv1.System{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	allowed, reason, err := h.validatingSystemFn(ctx, obj)

	log.Info("system webhook returning with", "allowed", allowed, "reason", reason, "error", err)

	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

var _ inject.Client = &SystemCreateUpdateHandler{}

// InjectClient injects the client into the SystemCreateUpdateHandler
func (h *SystemCreateUpdateHandler) InjectClient(c client.Client) error {
	h.Client = c
	return nil
}

var _ inject.Decoder = &SystemCreateUpdateHandler{}

// InjectDecoder injects the decoder into the SystemCreateUpdateHandler
func (h *SystemCreateUpdateHandler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}
