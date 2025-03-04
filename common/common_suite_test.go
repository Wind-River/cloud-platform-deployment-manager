/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022,2025 Wind River Systems, Inc. */
package common

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCommon(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Common Suite")
}
