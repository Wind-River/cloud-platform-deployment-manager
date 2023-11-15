/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019-2023 Wind River Systems, Inc. */

package manager

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/acceptance/clients"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/starlingx/inventory/v1/system"
	"github.com/gophercloud/gophercloud/starlingx/nfv/v1/systemconfigupdate"
	perrors "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// Expected Secret data and string data keys
	AuthUrlKey                     = "OS_AUTH_URL"
	UsernameKey                    = "OS_USERNAME"
	UserIDKey                      = "OS_USERID"
	PasswordKey                    = "OS_PASSWORD"
	TenantIDKey                    = "OS_TENANT_ID"
	TenantNameKey                  = "OS_TENANT_NAME"
	DomainIDKey                    = "OS_PROJECT_DOMAIN_ID"
	DomainNameKey                  = "OS_PROJECT_DOMAIN_NAME"
	RegionNameKey                  = "OS_REGION_NAME"
	KeystoneRegionNameKey          = "OS_KEYSTONE_REGION_NAME"
	ApplicationCredentialIDKey     = "OS_APPLICATION_CREDENTIAL_ID"
	ApplicationCredentialNameKey   = "OS_APPLICATION_CREDENTIAL_NAME"
	ApplicationCredentialSecretKey = "OS_APPLICATION_CREDENTIAL_SECRET"
	ProjectIDKey                   = "OS_PROJECT_ID"
	ProjectNameKey                 = "OS_PROJECT_NAME"
	InterfaceKey                   = "OS_INTERFACE"
	DebugKey                       = "OS_DEBUG"
)

const (
	// Well-known openstack API attribute values for the system API
	SystemEndpointName = "sysinv"
	SystemEndpointType = "platform"
	VimEndpointName    = "vim"
	VimEndpointType    = "nfv"
)

// Builds the client authentication options from a given secret which should
// contain environment variable like values.  For example, OS_AUTH_URL,
// OS_USERNAME, etc...
func GetAuthOptionsFromSecret(endpointSecret *v1.Secret) ([]gophercloud.AuthOptions, error) {
	username := string(endpointSecret.Data[UsernameKey])
	password := string(endpointSecret.Data[PasswordKey])
	authURL := string(endpointSecret.Data[AuthUrlKey])
	userID := string(endpointSecret.Data[UserIDKey])
	tenantID := string(endpointSecret.Data[TenantIDKey])
	tenantName := string(endpointSecret.Data[TenantNameKey])
	domainID := string(endpointSecret.Data[DomainIDKey])
	domainName := string(endpointSecret.Data[DomainNameKey])
	applicationCredentialID := string(endpointSecret.Data[ApplicationCredentialIDKey])
	applicationCredentialName := string(endpointSecret.Data[ApplicationCredentialNameKey])
	applicationCredentialSecret := string(endpointSecret.Data[ApplicationCredentialSecretKey])
	projectID := string(endpointSecret.Data[ProjectIDKey])
	projectName := string(endpointSecret.Data[ProjectNameKey])

	if projectID != "" {
		// If OS_PROJECT_ID is set, overwrite tenantID with the value.
		tenantID = projectID
	}

	if projectName != "" {
		// If OS_PROJECT_Name is set, overwrite tenantName with the value.
		tenantName = projectName
	}

	if authURL == "" {
		return nil, NewClientError("OS_AUTH_URL must be provided")
	}
	if userID == "" && username == "" {
		return nil, NewClientError("OS_USERID or OS_USERNAME must be provided")
	}

	if password == "" && applicationCredentialID == "" && applicationCredentialName == "" {
		return nil, NewClientError("OS_PASSWORD must be provided")
	}

	if (applicationCredentialID != "" || applicationCredentialName != "") && applicationCredentialSecret == "" {
		return nil, NewClientError("OS_APPLICATION_CREDENTIAL_SECRET must be provided")
	}

	result := make([]gophercloud.AuthOptions, 0)
	for _, entry := range strings.Split(authURL, ",") {
		// The authURL may be specified as a comma separated list of URL therefore
		// return a list of auth options to the caller.
		ao := gophercloud.AuthOptions{
			IdentityEndpoint:            entry,
			UserID:                      userID,
			Username:                    username,
			Password:                    password,
			TenantID:                    tenantID,
			TenantName:                  tenantName,
			DomainID:                    domainID,
			DomainName:                  domainName,
			ApplicationCredentialID:     applicationCredentialID,
			ApplicationCredentialName:   applicationCredentialName,
			ApplicationCredentialSecret: applicationCredentialSecret,
		}

		result = append(result, ao)
	}

	return result, nil
}

func GetAuthOptionsFromEnv() (gophercloud.AuthOptions, error) {
	password := os.Getenv(PasswordKey)
	username := os.Getenv(UsernameKey)
	authURL := os.Getenv(AuthUrlKey)
	userID := os.Getenv(UserIDKey)
	tenantID := os.Getenv(TenantIDKey)
	tenantName := os.Getenv(TenantNameKey)
	domainID := os.Getenv(DomainIDKey)
	domainName := os.Getenv(DomainNameKey)
	applicationCredentialID := os.Getenv(ApplicationCredentialIDKey)
	applicationCredentialName := os.Getenv(ApplicationCredentialNameKey)
	applicationCredentialSecret := os.Getenv(ApplicationCredentialSecretKey)
	projectID := os.Getenv(ProjectIDKey)
	projectName := os.Getenv(ProjectNameKey)

	if projectID != "" {
		// If OS_PROJECT_ID is set, overwrite tenantID with the value.
		tenantID = projectID
	}

	if projectName != "" {
		// If OS_PROJECT_Name is set, overwrite tenantName with the value.
		tenantName = projectName
	}

	if authURL == "" {
		return gophercloud.AuthOptions{}, NewClientError("OS_AUTH_URL must be provided")
	}

	if userID == "" && username == "" {
		return gophercloud.AuthOptions{}, NewClientError("OS_USERID or OS_USERNAME must be provided")
	}

	if password == "" && applicationCredentialID == "" && applicationCredentialName == "" {
		return gophercloud.AuthOptions{}, NewClientError("OS_PASSWORD must be provided")
	}

	if (applicationCredentialID != "" || applicationCredentialName != "") && applicationCredentialSecret == "" {
		return gophercloud.AuthOptions{}, NewClientError("OS_APPLICATION_CREDENTIAL_SECRET must be provided")
	}

	ao := gophercloud.AuthOptions{
		IdentityEndpoint:            authURL,
		UserID:                      userID,
		Username:                    username,
		Password:                    password,
		ApplicationCredentialID:     applicationCredentialID,
		ApplicationCredentialName:   applicationCredentialName,
		ApplicationCredentialSecret: applicationCredentialSecret,
	}

	if tenantID != "" {
		ao.TenantID = tenantID
	} else {
		ao.TenantName = tenantName
	}

	if domainID != "" {
		ao.DomainID = domainID
	} else {
		ao.DomainName = domainName
	}

	return ao, nil
}

func (m *PlatformManager) BuildPlatformClient(namespace string, endpointName string, endpointType string) (*gophercloud.ServiceClient, error) {
	var provider *gophercloud.ProviderClient

	secret := &v1.Secret{}
	secretName := types.NamespacedName{Namespace: namespace, Name: SystemEndpointSecretName}

	// Lookup the system endpoint secret for this namespace
	err := m.GetClient().Get(context.TODO(), secretName, secret)
	if err != nil {
		err = perrors.Wrap(err, "failed to find system endpoint secret")
		return nil, err
	}

	options, err := GetAuthOptionsFromSecret(secret)
	if err != nil {
		return nil, err
	}

	for _, authOptions := range options {
		// Force re-authentication on failures.
		authOptions.AllowReauth = true

	retry:
		// Authenticate against the openstack API
		provider, err = openstack.AuthenticatedClient(authOptions)
		if err != nil {
			if urlError, ok := err.(*url.Error); ok {
				if urlError.Err.Error() == "EOF" && strings.Contains(authOptions.IdentityEndpoint, HTTPPrefix) {
					// The endpoint has been switched to HTTPS mode so automatically
					// update our endpoint to HTTPS so that we can continue.
					authOptions.IdentityEndpoint = strings.Replace(authOptions.IdentityEndpoint, HTTPPrefix, HTTPSPrefix, 1)
					log.Info("retrying authentication request with HTTPS enabled")
					goto retry

				} else if strings.Contains(err.Error(), HTTPSNotEnabled) && strings.Contains(authOptions.IdentityEndpoint, HTTPSPrefix) {
					// The endpoint has been switched to HTTP mode so automatically
					// update our endpoint to HTTP so that we can continue.
					authOptions.IdentityEndpoint = strings.Replace(authOptions.IdentityEndpoint, HTTPSPrefix, HTTPPrefix, 1)
					log.Info("retrying authentication request with HTTPS disabled")
					goto retry
				}
			}

			authOptions.Password = "***REDACTED***" // redact for logging
			log.Error(err, "failed to authenticate client", "url", authOptions.IdentityEndpoint, "options", authOptions)

		} else {
			// Use the first successful client
			break
		}
	}

	if provider == nil {
		return nil, perrors.Wrap(err, "failed to authenticate against all available auth URL options")
	}

	availability := gophercloud.Availability(secret.Data[InterfaceKey])
	if availability == "" {
		availability = gophercloud.AvailabilityPublic
	}

	// Set the destination endpoint options to point to the system API
	endpointOpts := gophercloud.EndpointOpts{
		Name:         endpointName,
		Type:         endpointType,
		Availability: availability,
		Region:       string(secret.Data[RegionNameKey]),
	}

	// Get the system API URL
	urlEndpoint, err := provider.EndpointLocator(endpointOpts)
	if err != nil {
		err = perrors.Wrapf(err, "failed to find endpoint location, options: %+v", endpointOpts)
		return nil, err
	}

	c := &gophercloud.ServiceClient{
		ProviderClient: provider,
		Endpoint:       urlEndpoint,
		ResourceBase:   urlEndpoint}

	debug, err := strconv.ParseBool(string(secret.Data[DebugKey]))
	if err == nil && debug {
		// Debug is enabled so log all API requests/responses
		t := c.HTTPClient.Transport
		if t == nil {
			t = http.DefaultTransport
		}
		c.HTTPClient.Transport = &clients.LogRoundTripper{Rt: t}
	}

	if endpointName == SystemEndpointName {

		// Test the client because the authentication endpoint is different from
		// the resource endpoint therefore there is no guarantee that it works.
		_, err = system.GetDefaultSystem(c)
		if err != nil {
			err = perrors.Wrap(err, "failed to test system client connection")
			return nil, err
		}

		m.lock.Lock()
		defer func() { m.lock.Unlock() }()

		if obj, ok := m.systems[namespace]; !ok {
			m.systems[namespace] = &SystemNamespace{client: c}
			m.strategyStatus.Namespace = namespace
		} else {
			obj.client = c
		}
	} else if endpointName == VimEndpointName {
		// Test the client because the authentication endpoint is different from
		// the resource endpoint therefore there is no guarantee that it works.
		res, err := systemconfigupdate.Show(c)
		if err != nil || res == nil {
			err = perrors.Wrap(err, "failed to test vim client connection")
			return nil, err
		}
	}

	return c, nil
}
