/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2024 Wind River Systems, Inc. */

package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// Webhook response reasons
const PlatformNetworkAllowedReason string = "allowed to be admitted"

// log is for logging in this package.
var platformnetworklog = logf.Log.WithName("platformnetwork-resource")

func (r *PlatformNetwork) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-starlingx-windriver-com-v1-platformnetwork,mutating=true,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=platformnetworks,verbs=create;update,versions=v1,name=mplatformnetwork.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &PlatformNetwork{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *PlatformNetwork) Default() {
	platformnetworklog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-starlingx-windriver-com-v1-platformnetwork,mutating=false,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=platformnetworks,versions=v1,name=vplatformnetwork.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &PlatformNetwork{}

// TODO(sriram-gn): Identify and update validations for creation of PlatformNetwork resources.
// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PlatformNetwork) ValidateCreate() error {
	platformnetworklog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// TODO(sriram-gn): Identify and update validations for update of PlatformNetwork resources.
// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PlatformNetwork) ValidateUpdate(old runtime.Object) error {
	platformnetworklog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *PlatformNetwork) ValidateDelete() error {
	platformnetworklog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
