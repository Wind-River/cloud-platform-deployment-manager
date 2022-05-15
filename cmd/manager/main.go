/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wind-river/cloud-platform-deployment-manager/api"
	config2 "github.com/wind-river/cloud-platform-deployment-manager/common"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	var metricsAddr string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.Parse()

	// Some vendor modules still use glog which has the same command line
	// arguments defined as klog.  To get the klog arguments to parse we need to
	// do this workaround which is recommended by klog (see coexist_glog.go in
	// the klog module).
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	// Sync the glog and klog flags.
	flag.CommandLine.VisitAll(func(f1 *flag.Flag) {
		f2 := klogFlags.Lookup(f1.Name)
		if f2 != nil {
			value := f1.Value.String()
			err := f2.Value.Set(value)
			if err != nil {
				fmt.Println("Failed to set flag ", f2.Name, " err=", err)
				// continue anyway
			}
		}
	})

	// FIXME: cannot use klogger{...} (type klogger) as type logr.Logger in return argument
	//	logf.SetLogger(klogr.New())
	log := logf.Log.WithName("entrypoint")

	// Get a config to talk to the apiserver
	log.Info("setting up client for manager")
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "unable to set up client config")
		os.Exit(1)
	}

	// Load the manager config
	err = config2.ReadConfig()
	if err != nil {
		log.Error(err, "unable to read manager configuration")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	log.Info("setting up manager")
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: metricsAddr})
	if err != nil {
		log.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	log.Info("registering Components.")

	// Setup Scheme for all resources
	log.Info("setting up scheme")
	if err := api.AddToSchemeApi(mgr.GetScheme()); err != nil {
		log.Error(err, "unable add APIs to scheme")
		os.Exit(1)
	}

	log.Info("setting up webhooks")
	if err := api.AddToManagerWebhook(mgr); err != nil {
		log.Error(err, "unable to register webhooks to the manager")
		os.Exit(1)
	}

	// Setup all Controllers
	log.Info("setting up controller")
	if err := api.AddToManagerControllers(mgr); err != nil {
		log.Error(err, "unable to register controllers to the manager")
		os.Exit(1)
	}

	// Start the Cmd
	log.Info("starting the Cmd.")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "unable to run the manager")
		os.Exit(1)
	}
}
