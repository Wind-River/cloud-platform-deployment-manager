/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */
package osds

import (
	"github.com/alecthomas/units"
)

// Gibibytes returns the partition size in GiB units.
func (in *JournalInfo) Gibibytes() int {
	if in.Size != nil {
		return *in.Size / int(units.Kibibyte) // MiB -> GiB
	}
	return 0
}
