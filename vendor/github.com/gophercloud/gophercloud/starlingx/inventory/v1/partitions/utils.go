/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */
package partitions

import (
	"github.com/alecthomas/units"
)

// Gibibytes returns the partition size in GiB units.
// TODO(alegacy): remove once system API is converted to GiB units.
func (in *DiskPartition) Gibibytes() int {
	return in.Size / int(units.Kibibyte) // MiB -> GiB
}
