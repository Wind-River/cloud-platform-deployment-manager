/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2025 Wind River Systems, Inc. */

package v1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Webhook response reasons
const PlatformNetworkAllowedReason string = "allowed to be admitted"

// log is for logging in this package.
var platformnetworklog = logf.Log.WithName("platformnetwork-resource")

func (r *PlatformNetwork) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(&PlatformNetworkCustomDefaulter{}).
		WithValidator(&PlatformNetworkCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-starlingx-windriver-com-v1-platformnetwork,mutating=true,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=platformnetworks,verbs=create;update,versions=v1,name=mplatformnetwork.kb.io,admissionReviewVersions=v1,timeoutSeconds=30

type PlatformNetworkCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &PlatformNetworkCustomDefaulter{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (d *PlatformNetworkCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	platformNetwork, ok := obj.(*PlatformNetwork)
	if !ok {
		return fmt.Errorf("expected a PlatformNetwork object but got %T", obj)
	}
	platformnetworklog.Info("default", "name", platformNetwork.Name)
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-starlingx-windriver-com-v1-platformnetwork,mutating=false,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=platformnetworks,versions=v1,name=vplatformnetwork.kb.io,admissionReviewVersions=v1,timeoutSeconds=30

type PlatformNetworkCustomValidator struct{}

var _ webhook.CustomValidator = &PlatformNetworkCustomValidator{}

// TODO(sriram-gn): Identify and update validations for creation of PlatformNetwork resources.
// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *PlatformNetworkCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	platformNetwork, ok := obj.(*PlatformNetwork)
	if !ok {
		return nil, fmt.Errorf("expected a PlatformNetwork object but got %T", obj)
	}
	hostlog.Info("validate create", "name", platformNetwork.Name)

	platformnetworklog.Info("validate create", "name", platformNetwork.Name)
	return nil, nil
}

// TODO(sriram-gn): Identify and update validations for update of PlatformNetwork resources.
// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *PlatformNetworkCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	platformNetwork, ok := newObj.(*PlatformNetwork)
	if !ok {
		return nil, fmt.Errorf("expected a PlatformNetwork object but got %T", newObj)
	}
	platformnetworklog.Info("validate update", "name", platformNetwork.Name)
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *PlatformNetworkCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	platformNetwork, ok := obj.(*PlatformNetwork)
	if !ok {
		return nil, fmt.Errorf("expected a PlatformNetwork object but got %T", obj)
	}
	platformnetworklog.Info("validate delete", "name", platformNetwork.Name)
	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
