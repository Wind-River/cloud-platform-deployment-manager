/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2022 Wind River Systems, Inc. */

package ptpinterfaces

import "github.com/gophercloud/gophercloud"

func resourceURL(c *gophercloud.ServiceClient, id string) string {
	return c.ServiceURL("ptp_interfaces", id)
}

func rootURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("ptp_interfaces")
}

func getURL(c *gophercloud.ServiceClient, id string) string {
	return resourceURL(c, id)
}

func listURL(c *gophercloud.ServiceClient) string {
	return rootURL(c)
}

func createURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("ptp_interfaces")
}

func updateURL(c *gophercloud.ServiceClient, id string) string {
	return resourceURL(c, id)
}

func deleteURL(c *gophercloud.ServiceClient, id string) string {
	return resourceURL(c, id)
}

func hostListURL(c *gophercloud.ServiceClient, hostid string) string {
	return c.ServiceURL("ihosts", hostid, "ptp_interfaces")
}

func interfaceListURL(c *gophercloud.ServiceClient, interfaceid string) string {
	return c.ServiceURL("iinterfaces", interfaceid, "ptp_interfaces")
}

func interfaceUpdateURL(c *gophercloud.ServiceClient, interfaceid string) string {
	return c.ServiceURL("iinterfaces", interfaceid)
}
