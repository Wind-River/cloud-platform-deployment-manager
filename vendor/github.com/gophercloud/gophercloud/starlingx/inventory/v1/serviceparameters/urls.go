/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2020 Wind River Systems, Inc. */
/* https://docs.starlingx.io/api-ref/config/api-ref-sysinv-v1-config.html#service-parameter */

package serviceparameters

import "github.com/gophercloud/gophercloud"

func resourceURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL("service_parameter", id)
}

func rootURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("service_parameter")
}

func getURL(c *gophercloud.ServiceClient, id string) string {
	return resourceURL(c, id)
}

func listURL(c *gophercloud.ServiceClient) string {
	return rootURL(c)
}

func createURL(c *gophercloud.ServiceClient) string {
	return rootURL(c)
}

func updateURL(c *gophercloud.ServiceClient, id string) string {
	return resourceURL(c, id)
}

func deleteURL(c *gophercloud.ServiceClient, id string) string {
	return resourceURL(c, id)
}

func applyURL(c *gophercloud.ServiceClient) string {
	return resourceURL(c, "apply")
}
