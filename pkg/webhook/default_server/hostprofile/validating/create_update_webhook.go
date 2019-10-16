/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package validating

import (
	starlingxv1beta1 "github.com/wind-river/cloud-platform-deployment-manager/pkg/apis/starlingx/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
)

func init() {
	builderName := "validating-create-update-hostprofile"
	Builders[builderName] = builder.
		NewWebhookBuilder().
		Name(builderName+".windriver.com").
		Path("/"+builderName).
		Validating().
		Operations(admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update).
		FailurePolicy(admissionregistrationv1beta1.Fail).
		ForType(&starlingxv1beta1.HostProfile{})
}
