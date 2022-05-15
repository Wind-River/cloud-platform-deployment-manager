/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2019 Wind River Systems, Inc. */

package common

import (
	"fmt"
	"os"
	"reflect"

	"github.com/fsnotify/fsnotify"
	perrors "github.com/pkg/errors"
	"github.com/spf13/viper"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("controller").WithName("host")

// ReconcilerPrefix defines the viper configuration prefix for all reconcilers
// and sub-reconcilers.
const ReconcilerPrefix = "reconcilers"

// ReconcilerName is the type alias that represents the path for a reconciler
// or sub-reconciler.
type ReconcilerName string

// Defines the current list of supported reconcilers and sub-reconcilers.
const (
	DataNetwork       ReconcilerName = "dataNetwork"
	Host              ReconcilerName = "host"
	BMC               ReconcilerName = "host.bmc"
	Memory            ReconcilerName = "host.memory"
	Processor         ReconcilerName = "host.processor"
	Storage           ReconcilerName = "host.storage"
	FileSystems       ReconcilerName = "host.storage.filesystems"
	StorageMonitor    ReconcilerName = "host.storage.monitor"
	OSD               ReconcilerName = "host.storage.osd"
	Partition         ReconcilerName = "host.storage.partition"
	PhysicalVolume    ReconcilerName = "host.storage.physicalVolume"
	VolumeGroup       ReconcilerName = "host.storage.volumeGroup"
	Networking        ReconcilerName = "host.networking"
	Address           ReconcilerName = "host.networking.address"
	Interface         ReconcilerName = "host.networking.interface"
	Route             ReconcilerName = "host.networking.route"
	HostProfile       ReconcilerName = "hostProfile"
	PlatformNetwork   ReconcilerName = "platformNetwork"
	System            ReconcilerName = "system"
	Certificate       ReconcilerName = "system.certificate"
	DNS               ReconcilerName = "system.dns"
	DRBD              ReconcilerName = "system.drbd"
	SystemFileSystems ReconcilerName = "system.filesystems"
	License           ReconcilerName = "system.license"
	NTP               ReconcilerName = "system.ntp"
	PTP               ReconcilerName = "system.ptp"
	Backends          ReconcilerName = "system.storage.backend"
	ServiceParameters ReconcilerName = "system.serviceParameters"
	PTPInstance       ReconcilerName = "ptpInstance"
	PTPInterface      ReconcilerName = "ptpInterface"
)

// reconcilerDefaultStates is the default state of each reconciler.
var reconcilerDefaultStates = map[ReconcilerName]bool{
	DataNetwork:       true,
	Host:              true,
	BMC:               true,
	Memory:            true,
	Processor:         true,
	Storage:           true,
	FileSystems:       true,
	StorageMonitor:    true,
	OSD:               true,
	Partition:         true,
	PhysicalVolume:    true,
	VolumeGroup:       true,
	Networking:        true,
	Address:           true,
	Interface:         true,
	Route:             true,
	HostProfile:       true,
	PlatformNetwork:   true,
	System:            true,
	Certificate:       true,
	DNS:               true,
	DRBD:              true,
	SystemFileSystems: true,
	License:           true,
	NTP:               true,
	PTP:               true,
	Backends:          true,
	ServiceParameters: true,
	PTPInstance:       true,
	PTPInterface:      true,
}

// OptionName is the type alias that represents the path for a reconciler
// or sub-reconciler.
type OptionName string

// Defines the current list of supported reconciler options.
const (
	HTTPSRequired   OptionName = "httpsRequired"
	StopAfterInSync OptionName = "stopAfterInSync"
)

// reconcilerOptionDefaults is the default value for each reconciler option.
var reconcilerOptionDefaults = map[ReconcilerName]map[OptionName]interface{}{
	Certificate: {
		HTTPSRequired: true,
	},
	BMC: {
		HTTPSRequired: true,
	},
	DataNetwork: {
		StopAfterInSync: true,
	},
	PTPInterface: {
		StopAfterInSync: true,
	},
	PTPInstance: {
		StopAfterInSync: true,
	},
	Host: {
		StopAfterInSync: true,
	},
	PlatformNetwork: {
		StopAfterInSync: true,
	},
	System: {
		StopAfterInSync: true,
	},
}

// configFilepath is the absolute path of the manager config file.
const configFilepath = "/etc/manager/config.yaml"

var cfg *viper.Viper

// ReconcilerConfigPath returns the config attribute path which represents the
// top-level path for the specified reconciler.
func ReconcilerConfigPath(name ReconcilerName) string {
	return fmt.Sprintf("%s.%s", ReconcilerPrefix, name)
}

// ReconcilerStatePath returns the config attribute path which represents the
// current configured state of the reconciler.
func ReconcilerStatePath(name ReconcilerName) string {
	return fmt.Sprintf("%s.enabled", ReconcilerConfigPath(name))
}

// ReconcilerOptionPath returns the config attribute path which represents the
// option value of the specified reconciler option.
func ReconcilerOptionPath(name ReconcilerName, option OptionName) string {
	return fmt.Sprintf("%s.%s", ReconcilerConfigPath(name), option)
}

// ReadConfig is a utility which loads the current manager configuration into
// memory.
func ReadConfig() (err error) {
	if _, err := os.Stat(configFilepath); os.IsNotExist(err) {
		// The file is not present so use the defaults, and monitoring will
		// not be possible.
		return nil
	}

	err = cfg.ReadInConfig()
	if err == nil {
		cfg.WatchConfig()
		cfg.OnConfigChange(func(e fsnotify.Event) {
			log.Info("config file changed", "path", cfg.ConfigFileUsed())
		})

		log.Info("manager config has been loaded from file.")
	} else {
		err = perrors.Wrap(err, "failed to read config file")
	}

	return err
}

// IsReconcilerEnabled returns whether a specific reconciler is enabled or
// not.
func IsReconcilerEnabled(name ReconcilerName) bool {
	value := cfg.GetBool(ReconcilerStatePath(name))
	if !value {
		log.Info("reconciler is disabled", "name", string(name))
	}

	return value
}

// GetReconcilerOption returns the value of the specified option as an Interface
// value; otherwise nil is returned if the option does not exist in the config.
func GetReconcilerOption(name ReconcilerName, option OptionName) interface{} {
	return cfg.Get(ReconcilerOptionPath(name, option))
}

// GetReconcilerOptionBool returns the value of the specified option as a Bool
// value; otherwise the specified default value is returned if the option does
// not exist.
func GetReconcilerOptionBool(name ReconcilerName, option OptionName, defaultValue bool) bool {
	value := GetReconcilerOption(name, option)
	if value != nil {
		if required, ok := value.(bool); ok {
			return required
		} else {
			log.Info("unexpected option type",
				"option", option, "type", reflect.TypeOf(value))
		}
	}

	// Return the caller's default if not found.
	return defaultValue
}

func init() {
	cfg = viper.New()

	// Setup default values for all reconciler states
	for key, value := range reconcilerDefaultStates {
		path := ReconcilerStatePath(key)
		cfg.SetDefault(path, value)
	}

	// Setup default values for all reconciler options.
	for key, options := range reconcilerOptionDefaults {
		for option, value := range options {
			path := ReconcilerOptionPath(key, option)
			cfg.SetDefault(path, value)
		}
	}

	cfg.SetConfigFile(configFilepath)
	cfg.AutomaticEnv()
}
