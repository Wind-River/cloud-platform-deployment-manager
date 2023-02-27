/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2020 Wind River Systems, Inc. */

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version info variables are set in the Makefile
var GitLastTag string
var GitHead string
var GitBranch string
var GitPatch string

func VersionToString() string {
	if GitPatch == "" {
		return (fmt.Sprintf("%s (%s: %s)", GitLastTag, GitBranch, GitHead))
	} else {
		return (fmt.Sprintf("%s-%s (%s: %s)", GitLastTag, GitPatch, GitBranch, GitHead))
	}
}

func VersionCmdRun(cmd *cobra.Command, args []string) {
	fmt.Printf("Version: %s\n", VersionToString())
	os.Exit(0)
}

// versionCmd represents the build command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Run:   VersionCmdRun,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
