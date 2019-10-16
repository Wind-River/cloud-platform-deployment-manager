/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package validating

import (
	"context"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/datanetworks"
	"net/http"

	starlingxv1beta1 "github.com/wind-river/cloud-platform-deployment-manager/pkg/apis/starlingx/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// Webhook response reasons
const AllowedReason string = "allowed to be admitted"

func init() {
	webhookName := "validating-create-update-datanetwork"
	if HandlerMap[webhookName] == nil {
		HandlerMap[webhookName] = []admission.Handler{}
	}
	HandlerMap[webhookName] = append(HandlerMap[webhookName], &DataNetworkCreateUpdateHandler{})
}

// DataNetworkCreateUpdateHandler handles DataNetwork
type DataNetworkCreateUpdateHandler struct {
	// To use the client, you need to do the following:
	// - uncomment it
	// - import sigs.k8s.io/controller-runtime/pkg/client
	// - uncomment the InjectClient method at the bottom of this file.
	// client  client.client

	// Decoder decodes objects
	Decoder types.Decoder
}

func (h *DataNetworkCreateUpdateHandler) validateOptionalFields(obj *starlingxv1beta1.DataNetwork) (bool, string, error) {
	if obj.Spec.Type != datanetworks.TypeVxLAN {
		if obj.Spec.VxLAN != nil {
			return false, "VxLAN attributes are only allowed for VxLAN type data networks.", nil
		}
	}

	return true, AllowedReason, nil
}

func (h *DataNetworkCreateUpdateHandler) validatingDataNetworkFn(ctx context.Context, obj *starlingxv1beta1.DataNetwork) (bool, string, error) {
	allowed, reason, err := h.validateOptionalFields(obj)
	if !allowed || err != nil {
		return allowed, reason, err
	}
	return allowed, reason, err
}

var _ admission.Handler = &DataNetworkCreateUpdateHandler{}

// Handle handles admission requests.
func (h *DataNetworkCreateUpdateHandler) Handle(ctx context.Context, req types.Request) types.Response {
	obj := &starlingxv1beta1.DataNetwork{}

	err := h.Decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	allowed, reason, err := h.validatingDataNetworkFn(ctx, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

//var _ inject.client = &DataNetworkCreateUpdateHandler{}
//
//// InjectClient injects the client into the DataNetworkCreateUpdateHandler
//func (h *DataNetworkCreateUpdateHandler) InjectClient(c client.client) error {
//	h.client = c
//	return nil
//}

var _ inject.Decoder = &DataNetworkCreateUpdateHandler{}

// InjectDecoder injects the decoder into the DataNetworkCreateUpdateHandler
func (h *DataNetworkCreateUpdateHandler) InjectDecoder(d types.Decoder) error {
	h.Decoder = d
	return nil
}
