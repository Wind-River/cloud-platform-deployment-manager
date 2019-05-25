/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package validating

import (
	"context"
	"net/http"

	starlingxv1beta1 "github.com/wind-river/titanium-deployment-manager/pkg/apis/starlingx/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// Webhook response reasons
const AllowedReason string = "allowed to be admitted"

func init() {
	webhookName := "validating-create-update-host"
	if HandlerMap[webhookName] == nil {
		HandlerMap[webhookName] = []admission.Handler{}
	}
	HandlerMap[webhookName] = append(HandlerMap[webhookName], &HostCreateUpdateHandler{})
}

// HostCreateUpdateHandler handles Host
type HostCreateUpdateHandler struct {
	// To use the client, you need to do the following:
	// - uncomment it
	// - import sigs.k8s.io/controller-runtime/pkg/client
	// - uncomment the InjectClient method at the bottom of this file.
	// client  client.client

	// Decoder decodes objects
	Decoder types.Decoder
}

func (h *HostCreateUpdateHandler) validateMatchBMInfo(ctx context.Context, obj *starlingxv1beta1.Host) (bool, string, error) {
	if obj.Spec.Match.BoardManagement.Address == nil {
		return false, "board management address must be supplied in match criteria", nil
	}

	return true, AllowedReason, nil
}

func (h *HostCreateUpdateHandler) validateMatchDMIInfo(ctx context.Context, obj *starlingxv1beta1.Host) (bool, string, error) {
	if obj.Spec.Match.DMI.SerialNumber == nil || obj.Spec.Match.DMI.AssetTag == nil {
		return false, "DMI Serial Number or Asset Tag must be supplied in match criteria", nil
	}

	return true, AllowedReason, nil
}

func (h *HostCreateUpdateHandler) validateMatchInfo(ctx context.Context, obj *starlingxv1beta1.Host) (bool, string, error) {
	var allowed = true
	var reason = AllowedReason
	var err error

	match := obj.Spec.Match

	if match.BootMAC == nil && match.BoardManagement == nil && match.DMI == nil {
		return false, "host must be configured with at least 1 match criteria", nil
	}

	if match.BoardManagement != nil {
		allowed, reason, err = h.validateMatchBMInfo(ctx, obj)
		if !allowed || err != nil {
			return allowed, reason, err
		}
	}

	if match.DMI != nil {
		allowed, reason, err = h.validateMatchDMIInfo(ctx, obj)
		if !allowed || err != nil {
			return allowed, reason, err
		}
	}

	return true, AllowedReason, nil
}

func (h *HostCreateUpdateHandler) validatingHostFn(ctx context.Context, obj *starlingxv1beta1.Host) (bool, string, error) {
	if obj.Spec.Match != nil {
		allowed, reason, err := h.validateMatchInfo(ctx, obj)
		if !allowed || err != nil {
			return allowed, reason, err
		}
	}

	return true, AllowedReason, nil
}

var _ admission.Handler = &HostCreateUpdateHandler{}

// Handle handles admission requests.
func (h *HostCreateUpdateHandler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &starlingxv1beta1.Host{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	allowed, reason, err := h.validatingHostFn(ctx, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

//var _ inject.client = &HostCreateUpdateHandler{}
//
//// InjectClient injects the client into the HostCreateUpdateHandler
//func (h *HostCreateUpdateHandler) InjectClient(c client.client) error {
//	h.client = c
//	return nil
//}

var _ inject.Decoder = &HostCreateUpdateHandler{}

// InjectDecoder injects the decoder into the HostCreateUpdateHandler
func (h *HostCreateUpdateHandler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}
