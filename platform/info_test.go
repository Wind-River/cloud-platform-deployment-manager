/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2026 Wind River Systems, Inc. */

package platform

import (
	"testing"

	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/routes"
)

func TestFindRouteUUID_AmbiguousGateway(t *testing.T) {
	host := HostInfo{
		Routes: []routes.Route{
			{
				ID:            "uuid-route-1",
				InterfaceName: "eth0",
				Network:       "10.10.10.0",
				Prefix:        24,
				Gateway:       "10.10.10.1",
				Metric:        1,
			},
			{
				ID:            "uuid-route-2",
				InterfaceName: "eth0",
				Network:       "10.10.10.0",
				Prefix:        24,
				Gateway:       "10.10.10.2",
				Metric:        1,
			},
		},
	}

	// Look up the first route by gateway
	route1, found1 := host.FindRouteUUID("eth0", "10.10.10.0", 24, "10.10.10.1")
	if !found1 {
		t.Fatal("expected to find route for eth0/10.10.10.0/24 via 10.10.10.1 but got not found")
	}
	if route1.Gateway != "10.10.10.1" {
		t.Errorf("expected first route gateway to be 10.10.10.1, got %s", route1.Gateway)
	}
	if route1.ID != "uuid-route-1" {
		t.Errorf("expected first route ID to be uuid-route-1, got %s", route1.ID)
	}

	// Look up the second route by gateway — should now correctly find it
	route2, found2 := host.FindRouteUUID("eth0", "10.10.10.0", 24, "10.10.10.2")
	if !found2 {
		t.Fatal("expected to find route for eth0/10.10.10.0/24 via 10.10.10.2 but got not found")
	}
	if route2.Gateway != "10.10.10.2" {
		t.Errorf("expected second route gateway to be 10.10.10.2, got %s", route2.Gateway)
	}
	if route2.ID != "uuid-route-2" {
		t.Errorf("expected second route ID to be uuid-route-2, got %s", route2.ID)
	}
}
