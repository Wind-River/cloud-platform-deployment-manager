/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package validating

import (
	starlingxv1 "github.com/wind-river/cloud-platform-deployment-manager/pkg/apis/starlingx/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
)

func init() {
	builderName := "validating-create-update-ptpinterface"
	Builders[builderName] = builder.
		NewWebhookBuilder().
		Name(builderName+".windriver.com").
		Path("/"+builderName).
		Validating().
		Operations(admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update).
		FailurePolicy(admissionregistrationv1beta1.Fail).
		ForType(&starlingxv1.PtpInterface{})
}
