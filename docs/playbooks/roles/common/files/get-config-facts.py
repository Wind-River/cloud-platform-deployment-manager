#!/usr/bin/env python3
"""
SPDX-License-Identifier: Apache-2.0
# Copyright(c) 2025 Wind River Systems, Inc.

DM Configuration Parser

Parses YAML deployment configuration file to extract namespace and deployment scope.
Identifies:
- namespace from host resources (defaults to 'deployment')
- principal deployment scope flag from resource status
- hosts configured

Usage: python3 get-config-facts.py <deploy-config.yaml>
Output: json with namespace and principal scope boolean
"""

import json
import sys
import yaml


def safe_get_namespace(resource: dict) -> str:
    """Extract namespace from resource metadata, defaulting to 'deployment'.

    Returns:
       str: Namespace value or 'deployment' if not found
    """
    if not isinstance(resource, dict):
        return "deployment"

    metadata = resource.get("metadata")
    if not metadata:
        return "deployment"

    namespace = metadata.get("namespace")
    if not namespace:
        return "deployment"

    return namespace


def safe_get_principal_scope(resource: dict) -> bool:
    """Check if resource has deploymentScope set to 'principal'.

    Returns:
        bool: True if deploymentScope is 'principal', False otherwise
    """
    if not isinstance(resource, dict):
        return False

    status = resource.get("status", {})
    if not isinstance(status, dict):
        return False

    scope = status.get("deploymentScope", "")
    return scope.lower() == "principal"


def parse_config(file_path: str) -> dict:
    """Parse dm yaml deployment configuration to extract namespace and deployment scope.

    Extracts the namespace from any host resource and sets principal to True
    if any resource has deploymentScope set to 'principal'.

    Returns:
        dict: Configuration facts with 'namespace' (from host resource),
              'principal' (True if any resource has principal scope) and
              'hosts' with the number of Host resources
    """

    facts = {"namespace": "deployment", "principal": False, "hosts": 0}

    try:
        with open(file_path) as f:
            resources = yaml.safe_load_all(f)

            for r in resources:
                if not r:
                    continue

                facts["principal"] = facts["principal"] or safe_get_principal_scope(r)
                if r.get("kind").lower() == "host":
                    facts["namespace"] = safe_get_namespace(r)
                    facts["hosts"] += 1

    except Exception as e:
        pass

    return facts


try:
    config = parse_config(sys.argv[1])
except (IndexError, Exception):
    config = {"namespace": "deployment", "principal": False, "hosts": 0}

print(json.dumps(config))
