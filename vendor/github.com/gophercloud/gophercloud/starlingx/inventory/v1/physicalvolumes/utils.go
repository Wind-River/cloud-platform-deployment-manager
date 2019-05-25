/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */
package physicalvolumes

import (
	"github.com/alecthomas/units"
)

// Gibibytes returns the physical volume size in GiB units.
// TODO(alegacy): remove once system API is converted to GiB units.
func (in *PhysicalVolume) Gibibytes() int {
	return in.Size / int(units.Gibibyte) // B -> GiB
}
