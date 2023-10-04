/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2023 Wind River Systems, Inc. */

package common

// Service Parameter Service Types
const ServiceTypeIdentity = "identity"
const ServiceTypePlatform = "platform"
const ServiceTypeRadosgw = "radosgw"
const ServiceTypeHttp = "http"

// Service Parameter Section
const ServiceParamSectionHttpConfig = "config"
const ServiceParamSectionPlatformConfig = "config"
const ServiceParamSectionPlatformMaintenance = "maintenance"
const ServiceParamSectionPlatformKernel = "kernel"
const ServiceParamSectionIdentityConfig = "config"
const ServiceParamSectionRadosgwConfig = "config"
const ServiceParamSectionSecurityCompliance = "security_compliance"

// Service Parameter Name
const ServiceParamHttpPortHttp = "http_port"
const ServiceParamHttpPortHttps = "https_port"
const ServiceParamIdentityConfigTokenExpiration = "token_expiration"
const ServiceParamNamePlatformAuditD = "audit"
const ServiceParamNamePlatformMaxCpuPercentage = "cpu_max_freq_min_percentage"

const ServiceParamNamePlatConfigIntelNicDriverVersion = "intel_nic_driver_version"
const ServiceParamNamePlatConfigIntelPstate = "intel_pstate"
const ServiceParamNameRadosgwFsSizeMB = "fs_size_mb"
const ServiceParamNameRadosgwServiceEnabled = "service_enabled"
const ServiceParamNameSecurityComplianceLockoutDuration = "lockout_seconds"

const ServiceParamNameSecurityComplianceLockoutFailureAttempts = "lockout_retries"
const ServiceParamPlatMtceControllerBootTimeout = "controller_boot_timeout"
const ServiceParamPlatMtceHbsPERIOD = "heartbeat_period"
const ServiceParamPlatMtceHbsDegradeThreshold = "heartbeat_degrade_threshold"
const ServiceParamPlatMtceHbsFailureAction = "heartbeat_failure_action"
const ServiceParamPlatMtceHbsFailureThreshold = "heartbeat_failure_threshold"
const ServiceParamPlatMtceMnfaThreshold = "mnfa_threshold"
const ServiceParamPlatMtceMnfaTimeout = "mnfa_timeout"
const ServiceParamPlatMtceWorkerBootTimeout = "worker_boot_timeout"

type ServiceParam struct {
	Service   string
	Section   string
	ParamName string
}

var DefaultParameters = [...]ServiceParam{
	ServiceParam{Service: ServiceTypeIdentity,
		Section:   ServiceParamSectionIdentityConfig,
		ParamName: ServiceParamIdentityConfigTokenExpiration,
	},
	ServiceParam{Service: ServiceTypeIdentity,
		Section:   ServiceParamSectionSecurityCompliance,
		ParamName: ServiceParamNameSecurityComplianceLockoutDuration,
	},
	ServiceParam{Service: ServiceTypeIdentity,
		Section:   ServiceParamSectionSecurityCompliance,
		ParamName: ServiceParamNameSecurityComplianceLockoutFailureAttempts,
	},
	ServiceParam{Service: ServiceTypePlatform,
		Section:   ServiceParamSectionPlatformMaintenance,
		ParamName: ServiceParamPlatMtceWorkerBootTimeout,
	},
	ServiceParam{Service: ServiceTypePlatform,
		Section:   ServiceParamSectionPlatformMaintenance,
		ParamName: ServiceParamPlatMtceControllerBootTimeout,
	},
	ServiceParam{Service: ServiceTypePlatform,
		Section:   ServiceParamSectionPlatformMaintenance,
		ParamName: ServiceParamPlatMtceHbsPERIOD,
	},
	ServiceParam{Service: ServiceTypePlatform,
		Section:   ServiceParamSectionPlatformMaintenance,
		ParamName: ServiceParamPlatMtceHbsFailureAction,
	},
	ServiceParam{Service: ServiceTypePlatform,
		Section:   ServiceParamSectionPlatformMaintenance,
		ParamName: ServiceParamPlatMtceHbsFailureThreshold,
	},
	ServiceParam{Service: ServiceTypePlatform,
		Section:   ServiceParamSectionPlatformMaintenance,
		ParamName: ServiceParamPlatMtceHbsDegradeThreshold,
	},
	ServiceParam{Service: ServiceTypePlatform,
		Section:   ServiceParamSectionPlatformMaintenance,
		ParamName: ServiceParamPlatMtceMnfaThreshold,
	},
	ServiceParam{Service: ServiceTypePlatform,
		Section:   ServiceParamSectionPlatformMaintenance,
		ParamName: ServiceParamPlatMtceMnfaTimeout,
	},
	ServiceParam{Service: ServiceTypePlatform,
		Section:   ServiceParamSectionPlatformKernel,
		ParamName: ServiceParamNamePlatformAuditD,
	},
	ServiceParam{Service: ServiceTypePlatform,
		Section:   ServiceParamSectionPlatformConfig,
		ParamName: ServiceParamNamePlatConfigIntelNicDriverVersion,
	},
	ServiceParam{Service: ServiceTypePlatform,
		Section:   ServiceParamSectionPlatformConfig,
		ParamName: ServiceParamNamePlatConfigIntelPstate,
	},
	ServiceParam{Service: ServiceTypeRadosgw,
		Section:   ServiceParamSectionRadosgwConfig,
		ParamName: ServiceParamNameRadosgwServiceEnabled,
	},
	ServiceParam{Service: ServiceTypeRadosgw,
		Section:   ServiceParamSectionRadosgwConfig,
		ParamName: ServiceParamNameRadosgwFsSizeMB,
	},
	ServiceParam{Service: ServiceTypeHttp,
		Section:   ServiceParamSectionHttpConfig,
		ParamName: ServiceParamHttpPortHttp,
	},
	ServiceParam{Service: ServiceTypeHttp,
		Section:   ServiceParamSectionHttpConfig,
		ParamName: ServiceParamHttpPortHttps,
	},
	ServiceParam{Service: ServiceTypePlatform,
		Section:   ServiceParamSectionPlatformConfig,
		ParamName: ServiceParamNamePlatformMaxCpuPercentage,
	},
}
