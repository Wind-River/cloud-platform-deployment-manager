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

package v1

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apitypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var systemlog = logf.Log.WithName("system-resource")
var cl client.Client

// Webhook response reasons
const SystemAllowedReason string = "allowed to be admitted"

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

func (r *System) SetupWebhookWithManager(mgr ctrl.Manager) error {
	cl = mgr.GetClient()

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-starlingx-windriver-com-v1-system,mutating=true,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=systems,verbs=create;update,versions=v1,name=msystem.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &System{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *System) Default() {
	systemlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

func validateBackendServices(backendType string, services []string) error {
	for _, s := range services {
		if !validBackendServices[backendType][s] {
			msg := fmt.Sprintf("%s service not allowed with %s backend.", s, backendType)
			return errors.New(msg)
		}
	}

	return nil
}

func validateBackendAttributes(backend StorageBackend) error {
	if backend.PartitionSize != nil || backend.ReplicationFactor != nil {
		if backend.Type != ceph {
			msg := fmt.Sprintf("partitionSize and ReplicationFactor only permitted with %s backend", ceph)
			return errors.New(msg)
		}
	}

	return nil
}

func validateStorageBackends(obj *System) error {
	var present = make(map[string]bool)

	for _, b := range *obj.Spec.Storage.Backends {
		if present[b.Type] {
			return errors.New("backend services may only be specified once.")
		}

		if b.Services != nil {
			err := validateBackendServices(b.Type, b.Services)
			if err != nil {
				return err
			}
		}

		err := validateBackendAttributes(b)
		if err != nil {
			return err
		}

		present[b.Type] = true
	}

	return nil
}

func validateStorage(obj *System) error {
	if obj.Spec.Storage != nil && obj.Spec.Storage.Backends != nil {
		err := validateStorageBackends(obj)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateCertificates(obj *System) error {
	if obj.Spec.Certificates != nil {
		for _, c := range *obj.Spec.Certificates {
			secret := &corev1.Secret{}
			secretName := apitypes.NamespacedName{Name: c.Secret, Namespace: obj.ObjectMeta.Namespace}

			ctx := context.Background()
			err := cl.Get(ctx, secretName, secret)
			if err != nil {
				msg := fmt.Sprintf("unable to retrieve %s secret %s", c.Type, secretName)
				return errors.New(msg)
			}
		}
	}

	return nil
}

func (r *System) validatingSystem() error {
	err := validateStorage(r)
	if err != nil {
		return err
	}

	err = validateCertificates(r)
	if err != nil {
		return err
	}

	systemlog.Info(SystemAllowedReason)
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-starlingx-windriver-com-v1-system,mutating=false,failurePolicy=fail,sideEffects=None,groups=starlingx.windriver.com,resources=systems,versions=v1,name=vsystem.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &System{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *System) ValidateCreate() error {
	systemlog.Info("validate create", "name", r.Name)

	r.validatingSystem()

	return r.validatingSystem()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *System) ValidateUpdate(old runtime.Object) error {
	systemlog.Info("validate update", "name", r.Name)

	r.validatingSystem()

	return r.validatingSystem()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *System) ValidateDelete() error {
	systemlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
