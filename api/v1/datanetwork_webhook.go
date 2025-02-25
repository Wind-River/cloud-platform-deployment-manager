/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2022,2025 Wind River Systems, Inc. */

package v1

import (
	"context"
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/datanetworks"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var datanetworklog = logf.Log.WithName("datanetwork-resource")

func (r *DataNetwork) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(&DataNetworkCustomDefaulter{}).
		WithValidator(&DataNetworkCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-starlingx-windriver-com-v1-datanetwork,mutating=true,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=datanetworks,verbs=create;update,versions=v1,name=mdatanetwork.kb.io,admissionReviewVersions=v1,timeoutSeconds=30

type DataNetworkCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &DataNetworkCustomDefaulter{}

// Webhook response reasons
const AllowedReason string = "allowed to be admitted"

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (d *DataNetworkCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	dataNetwork, ok := obj.(*DataNetwork)
	if !ok {
		return fmt.Errorf("expected a DataNetwork object but got %T", obj)
	}
	datanetworklog.Info("default", "name", dataNetwork.Name)
	return nil
}

func (r *DataNetwork) validateDataNetwork() error {
	if r.Spec.Type != datanetworks.TypeVxLAN {
		if r.Spec.VxLAN != nil {
			return errors.New("VxLAN attributes are only allowed for VxLAN type data networks.")
		}
	}
	datanetworklog.Info(AllowedReason)
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-starlingx-windriver-com-v1-datanetwork,mutating=false,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=datanetworks,versions=v1,name=vdatanetwork.kb.io,admissionReviewVersions=v1,timeoutSeconds=30

type DataNetworkCustomValidator struct{}

var _ webhook.CustomValidator = &DataNetworkCustomValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *DataNetworkCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	dataNetwork, ok := obj.(*DataNetwork)
	if !ok {
		return nil, fmt.Errorf("expected a DataNetwork object but got %T", obj)
	}
	datanetworklog.Info("validate create", "name", dataNetwork.Name)
	return nil, dataNetwork.validateDataNetwork()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *DataNetworkCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	dataNetwork, ok := newObj.(*DataNetwork)
	if !ok {
		return nil, fmt.Errorf("expected a DataNetwork object but go %T", newObj)
	}
	datanetworklog.Info("validate update", "name", dataNetwork.Name)

	return nil, dataNetwork.validateDataNetwork()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *DataNetworkCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	dataNetwork, ok := obj.(*AddressPool)
	if !ok {
		return nil, fmt.Errorf("expected a AddressPool object but got %T", obj)
	}
	addresspoollog.Info("validate delete", "name", dataNetwork.Name)
	return nil, nil
}
