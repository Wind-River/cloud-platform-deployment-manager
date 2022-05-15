/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
/* Copyright(c) 2019-2022 Wind River Systems, Inc. */

package v1

import (
	"errors"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/datanetworks"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var datanetworklog = logf.Log.WithName("datanetwork-resource")

func (r *DataNetwork) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-starlingx-windriver-com-v1-datanetwork,mutating=true,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=datanetworks,verbs=create;update,versions=v1,name=mdatanetwork.kb.io,admissionReviewVersions=v1
var _ webhook.Defaulter = &DataNetwork{}

// Webhook response reasons
const AllowedReason string = "allowed to be admitted"

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *DataNetwork) Default() {
	datanetworklog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
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
// +kubebuilder:webhook:verbs=create;update,path=/validate-starlingx-windriver-com-v1-datanetwork,mutating=false,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=datanetworks,versions=v1,name=vdatanetwork.kb.io,admissionReviewVersions=v1
var _ webhook.Validator = &DataNetwork{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *DataNetwork) ValidateCreate() error {
	datanetworklog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return r.validateDataNetwork()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *DataNetwork) ValidateUpdate(old runtime.Object) error {
	datanetworklog.Info("validate update", "name", r.Name)

	return r.validateDataNetwork()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *DataNetwork) ValidateDelete() error {
	datanetworklog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
