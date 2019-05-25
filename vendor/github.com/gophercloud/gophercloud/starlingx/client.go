/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package starlingx

import "github.com/gophercloud/gophercloud"

func initClientOpts(client *gophercloud.ProviderClient, eo gophercloud.EndpointOpts, clientType string) (*gophercloud.ServiceClient, error) {
	sc := new(gophercloud.ServiceClient)
	eo.ApplyDefaults(clientType)
	url, err := client.EndpointLocator(eo)
	if err != nil {
		return sc, err
	}
	sc.ProviderClient = client
	sc.Endpoint = url
	sc.Type = clientType
	return sc, nil
}

// NewInventoryV1 creates a ServiceClient that may be used with the v1 inventory
// package.
func NewInventoryV1(client *gophercloud.ProviderClient, eo gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error) {
	return initClientOpts(client, eo, "inventory")
}
