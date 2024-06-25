/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2024 Wind River Systems, Inc. */

package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wind-river/cloud-platform-deployment-manager/controllers/manager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StarlingxInstance interface {
	client.Object
	GetObjectMeta() metav1.Object
	GetStrategyRequired() string
	SetStrategyRequired(strategy string)
	GetDeploymentScope() string
	SetDeploymentScope(scope string)
}

func extractScope(config string) (string, error) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(config), &data)
	if err != nil {
		return "", err
	}

	status, ok := data["status"].(map[string]interface{})
	if !ok {
		return "", nil
	}

	scope, ok := status["deploymentScope"].(string)
	if !ok {
		return "", nil
	}

	return scope, nil
}

// GetDeploymentScope is to get the deploymentScope from the last applied configuration
// and set expected value based on the data.
func GetDeploymentScope(instance StarlingxInstance) (string, error) {
	// Set default value for deployment scope
	scope := manager.ScopeBootstrap
	// Set DeploymentScope from configuration
	annotation := instance.GetObjectMeta().GetAnnotations()
	if annotation == nil {
		return scope, nil
	}

	config, ok := annotation["kubectl.kubernetes.io/last-applied-configuration"]
	if !ok {
		return scope, nil
	}

	scope2, err := extractScope(config)
	if err != nil {
		return scope, err
	}

	if scope2 != "" {
		scope = scope2
	}

	scope, err = validateDeploymentScope(scope)
	return scope, err
}

func validateDeploymentScope(scope string) (string, error) {
	lowerCaseScope := strings.ToLower(scope)
	switch lowerCaseScope {
	case manager.ScopeBootstrap:
		return manager.ScopeBootstrap, nil
	case manager.ScopePrincipal:
		return manager.ScopePrincipal, nil
	default:
		return manager.ScopeBootstrap, fmt.Errorf("unsupported DeploymentScope: %s", scope)
	}
}

func UpdateDeploymentScope(client client.Client, instance StarlingxInstance) (bool, error) {
	scope, err := GetDeploymentScope(instance)
	if err != nil {
		return false, err
	}

	strategyRequired := instance.GetStrategyRequired()
	if strategyRequired == "" {
		instance.SetStrategyRequired(manager.StrategyNotRequired)
	}

	if instance.GetDeploymentScope() != scope {
		instance.SetDeploymentScope(scope)
		err := client.Status().Update(context.TODO(), instance)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}
