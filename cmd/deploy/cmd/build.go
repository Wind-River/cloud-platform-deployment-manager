/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package cmd

import (
	"fmt"
	neturl "net/url"
	"os"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/spf13/cobra"
	"github.com/wind-river/cloud-platform-deployment-manager/build"
	"github.com/wind-river/cloud-platform-deployment-manager/manager"
)

const (
	OutputFileNameArg                = "output-file"
	SystemNameArg                    = "system-name"
	NamespaceNameArg                 = "namespace-name"
	NoCACertificatesFilterArg        = "no-ca-certificates"
	NoDefaultsFilterArg              = "no-defaults"
	NoMemoryFilterArg                = "no-memory"
	NoProcessorFilterArg             = "no-processors"
	NoInterfaceDefaultsFilterArg     = "no-interface-defaults"
	NoServiceParametersFilterArg     = "no-service-parameters"
	NoSysVgFilterArg                 = "no-sys-vg"
	NormalizeInterfaceNamesFilterArg = "normalize-interfaces"
	NormalizeInterfaceMTUFilterArg   = "normalize-mtu"
	NormalizeConsoleFilterArg        = "normalize-console"
	MinimalConfigFilterArg           = "minimal-config"
)

func CollectCmdRun(cmd *cobra.Command, args []string) {
	var normalizeInterfaces bool
	var noInterfaceDefaults bool
	var normalizeConsole bool
	var noCACertificates bool
	var noServiceParams bool
	var outputFile *os.File
	var minimalConfig bool
	var noProcessors bool
	var normalizeMTU bool
	var namespace string
	var noDefaults bool
	var noMemory bool
	var noSysVg bool
	var name string
	var err error

	if outputFilename, err := cmd.Flags().GetString(OutputFileNameArg); err == nil {
		outputFile, err = os.Create(outputFilename)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to open output file: %s\n",
				err.Error())
			os.Exit(1)
		}
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get %q argument\n",
			OutputFileNameArg)
		os.Exit(2)
	}

	if namespace, err = cmd.Flags().GetString(NamespaceNameArg); err == nil {
		if namespace == "" {
			_, _ = fmt.Fprintf(os.Stderr, "namespace name must not be blank\n")
			os.Exit(14)
		} else if strings.Contains(namespace, " ") || strings.Contains(namespace, "\t") {
			_, _ = fmt.Fprintf(os.Stderr, "namespace name must not contain whitespace characters\n")
			os.Exit(15)
		}

		// Kubernetes does not allow underscores in resource names.
		namespace = strings.Replace(namespace, "_", "-", -1)

	} else {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get %q argument\n",
			NamespaceNameArg)
		os.Exit(16)
	}

	if name, err = cmd.Flags().GetString(SystemNameArg); err == nil {
		if name == "" {
			_, _ = fmt.Fprintf(os.Stderr, "system name must not be blank\n")
			os.Exit(3)
		} else if strings.Contains(name, " ") || strings.Contains(name, "\t") {
			_, _ = fmt.Fprintf(os.Stderr, "system name must not contain whitespace characters\n")
			os.Exit(4)
		}

		// Kubernetes does not allow underscores in resource names.
		name = strings.Replace(name, "_", "-", -1)

	} else {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get %q argument\n",
			SystemNameArg)
		os.Exit(5)
	}

	if noMemory, err = cmd.Flags().GetBool(NoMemoryFilterArg); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get %q argument\n",
			NoMemoryFilterArg)
		os.Exit(6)
	}

	if noProcessors, err = cmd.Flags().GetBool(NoProcessorFilterArg); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get %q argument\n",
			NoProcessorFilterArg)
		os.Exit(7)
	}

	if noInterfaceDefaults, err = cmd.Flags().GetBool(NoInterfaceDefaultsFilterArg); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get %q argument\n",
			NoInterfaceDefaultsFilterArg)
		os.Exit(8)
	}

	if noDefaults, err = cmd.Flags().GetBool(NoDefaultsFilterArg); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get %q argument\n",
			NoDefaultsFilterArg)
		os.Exit(9)
	}

	if normalizeInterfaces, err = cmd.Flags().GetBool(NormalizeInterfaceNamesFilterArg); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get %q argument\n",
			NormalizeInterfaceNamesFilterArg)
		os.Exit(10)
	}

	if normalizeMTU, err = cmd.Flags().GetBool(NormalizeInterfaceMTUFilterArg); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get %q argument\n",
			NormalizeInterfaceMTUFilterArg)
		os.Exit(11)
	}

	if normalizeConsole, err = cmd.Flags().GetBool(NormalizeConsoleFilterArg); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get %q argument\n",
			NormalizeConsoleFilterArg)
		os.Exit(12)
	}

	if minimalConfig, err = cmd.Flags().GetBool(MinimalConfigFilterArg); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get %q argument\n",
			NormalizeConsoleFilterArg)
		os.Exit(13)
	}

	if noCACertificates, err = cmd.Flags().GetBool(NoCACertificatesFilterArg); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get %q argument\n",
			NoCACertificatesFilterArg)
		os.Exit(14)
	}

	if noSysVg, err = cmd.Flags().GetBool(NoSysVgFilterArg); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get %q argument\n",
			NoSysVgFilterArg)
		os.Exit(15)
	}

	if noServiceParams, err = cmd.Flags().GetBool(NoServiceParametersFilterArg); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get %q argument\n",
			NoServiceParametersFilterArg)
		os.Exit(16)
	}

	if minimalConfig {
		noCACertificates = true
		noDefaults = true
		noInterfaceDefaults = true
		normalizeInterfaces = true
		normalizeMTU = true
		normalizeConsole = true
		noServiceParams = true
	}

	ao, err := manager.GetAuthOptionsFromEnv()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "failed to build authentication options", err)
		os.Exit(30)
	}

	ao.AllowReauth = true

	// Authenticate with keystone to make a new client.
	provider, err := openstack.AuthenticatedClient(ao)
	if err != nil {
		if urlError, ok := err.(*neturl.Error); ok {
			if urlError.Err.Error() == "EOF" && strings.Contains(ao.IdentityEndpoint, manager.HTTPPrefix) {
				_, _ = fmt.Fprintf(os.Stderr, "URL contains an HTTP scheme but the server might be HTTPS enabled; %s\n",
					ao.IdentityEndpoint)
			} else if strings.Contains(err.Error(), manager.HTTPSNotEnabled) && strings.Contains(ao.IdentityEndpoint, manager.HTTPSPrefix) {
				_, _ = fmt.Fprintf(os.Stderr, "URL contains an HTTPS scheme but the server is not HTTPS enabled; %s\n",
					ao.IdentityEndpoint)
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "unknown URL error: %s\n",
					urlError.Error())
			}

		} else {
			ao.Password = "" // redact for logging
			_, _ = fmt.Fprintf(os.Stderr, "failed to authenticate client with options %+v, Error: %s\n", ao,
				err.Error())
		}

		os.Exit(31)
	}

	availability := gophercloud.Availability(os.Getenv(manager.InterfaceKey))
	if availability == "" {
		availability = gophercloud.AvailabilityPublic
	}

	endpointOpts := gophercloud.EndpointOpts{
		Name:         manager.SystemEndpointName,
		Type:         manager.SystemEndpointType,
		Availability: availability,
		Region:       os.Getenv(manager.RegionNameKey),
	}

	// Get the system API URL
	url, err := provider.EndpointLocator(endpointOpts)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to find endpoint location for opts %+v: %s\n",
			endpointOpts, err.Error())
		os.Exit(32)
	}

	// Combine the target endpoint information with the keystone client to form
	// a final client which will be used to communicate with the system API.
	client := &gophercloud.ServiceClient{
		ProviderClient: provider,
		Endpoint:       url,
		ResourceBase:   url}

	builder := build.NewDeploymentBuilder(client, namespace, name, os.Stdout)

	profileFilters := make([]build.ProfileFilter, 0)

	if noDefaults {
		profileFilters = append(profileFilters,
			build.NewInterfaceUnusedFilter())
	}

	if noDefaults && !noMemory {
		profileFilters = append(profileFilters, build.NewMemoryDefaultsFilter())
	} else if noMemory {
		profileFilters = append(profileFilters, build.NewMemoryClearAllFilter())
	}

	if noDefaults && !noProcessors {
		profileFilters = append(profileFilters, build.NewProcessorDefaultsFilter())
	} else if noProcessors {
		profileFilters = append(profileFilters, build.NewProcessorClearAllFilter())
	}

	if noInterfaceDefaults {
		profileFilters = append(profileFilters, build.NewInterfaceDefaultsFilter())
	}

	if normalizeInterfaces {
		profileFilters = append(profileFilters, build.NewInterfaceNamingFilter())
	}

	if normalizeMTU {
		profileFilters = append(profileFilters, build.NewInterfaceMTUFilter())
	}

	if normalizeConsole {
		profileFilters = append(profileFilters, build.NewConsoleNameFilter())
	}

	if noSysVg {
		profileFilters = append(profileFilters, build.NewVolumeGroupSystemFilter())
	}

	if len(profileFilters) > 0 {
		builder.AddProfileFilters(profileFilters)
	}

	systemFilters := make([]build.SystemFilter, 0)

	if noCACertificates {
		systemFilters = append(systemFilters, build.NewCACertificateFilter())
	}

	if noServiceParams {
		systemFilters = append(systemFilters, build.NewServiceParametersSystemFilter())
	}

	if len(systemFilters) > 0 {
		builder.AddSystemFilters(systemFilters)
	}

	deployment, err := builder.Build()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to build deployment details: %s\n", err.Error())
		os.Exit(40)
	}

	yamlBuf, err := deployment.ToYAML()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to convert deployment struct to YAML: %s\n", err.Error())
		os.Exit(41)
	}

	_, err = fmt.Fprintf(outputFile, "# Generated: %s\n# Tool version: %s\n",
		time.Now().Format(time.UnixDate),
		VersionToString())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to write to output file: %s\n", err.Error())
		os.Exit(42)
	}

	_, err = fmt.Fprintf(outputFile, "%s", yamlBuf)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to write to output file: %s\n", err.Error())
		os.Exit(42)
	}

	err = outputFile.Close()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to close output file: %s\n", err.Error())
		os.Exit(43)
	}

	fmt.Printf("done.\n")

	if len(deployment.IncompleteSecrets) != 0 {
		fmt.Printf("\nWarning: The generated deployment configuration contains kubernetes\n")
		fmt.Printf("  Secrets that must be manually edited to add information that is not\n")
		fmt.Printf("  retrievable from the system.  For example, any BMC Secrets must be\n")
		fmt.Printf("  edited to add the password and any SSL Secrets must be edited to add\n")
		fmt.Printf("  the certificate and key information.  Any such information must be\n")
		fmt.Printf("  added in base64 encoded format.\n")
		os.Exit(44)
	}
}

// collectCmd represents the build command
var collectCmd = &cobra.Command{
	Use:   "build",
	Short: "The build subcommand extracts the configuration from a running system",
	Long: `The build subcommand extracts the configuration from a running system.
The yaml output from this tool must be manually verified and updated to fill-in
fields that are otherwise not automatically settable (i.e., secrets,
certificates).  This command requires that the Openstack credentials be sourced
to the current environment variables.`,
	Run: CollectCmdRun,
}

func init() {
	rootCmd.AddCommand(collectCmd)

	// Here you will define your flags and configuration settings.
	collectCmd.Flags().StringP(OutputFileNameArg, "o", "deployment-config.yaml", "A destination path used for output.")
	collectCmd.Flags().StringP(SystemNameArg, "s", "", "The name of the system to be created")
	collectCmd.Flags().StringP(NamespaceNameArg, "n", "deployment", "The name of the namespace used to contain the system")
	collectCmd.Flags().BoolP(NoDefaultsFilterArg, "f", false, "Enable/disable use of filters to remove unwanted fields")
	collectCmd.Flags().Bool(NoCACertificatesFilterArg, false, "Exclude all trusted CA certificates from system instances")
	collectCmd.Flags().Bool(NoMemoryFilterArg, false, "Exclude all memory configurations from profiles")
	collectCmd.Flags().Bool(NoProcessorFilterArg, false, "Exclude all processor configurations from profiles")
	collectCmd.Flags().Bool(NoInterfaceDefaultsFilterArg, false, "Exclude all interface default values from profiles")
	collectCmd.Flags().Bool(NoSysVgFilterArg, false, "Exclude system volume groups")
	collectCmd.Flags().Bool(NoServiceParametersFilterArg, false, "Exclude service parameters")
	collectCmd.Flags().Bool(NormalizeInterfaceNamesFilterArg, false, "Normalize interface names")
	collectCmd.Flags().Bool(NormalizeInterfaceMTUFilterArg, false, "Normalize interface MTU values")
	collectCmd.Flags().Bool(NormalizeConsoleFilterArg, false, "Normalize serial console attributes")
	collectCmd.Flags().Bool(MinimalConfigFilterArg, false, "Shorthand notation for adding all available filters")
}
