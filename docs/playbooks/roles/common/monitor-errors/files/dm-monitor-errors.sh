#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright(c) 2026 Wind River Systems, Inc.
#
# Prints the requests that failed on the system API (sysinv).
#
# The deployment manager records a Warning event for each request that failed
# on the system API, whatever the resource being reconciled. The events are on
# the deployment manager deployment, so they are all in its namespace, and
# survive the pod being replaced by the unlock. Their message starts with the
# kind and name of the resource. The reason is one of:
#   SystemAPIRejected     rejected (4xx), the configuration must be corrected
#   SystemAPITransient    no response, request timeout or rate limiting
#   SystemAPIServerError  failed on the server (5xx)
# Kubernetes folds identical events into one, so a request that fails on every
# retry is printed once, with the number of times it failed.
#
# A deployment manager that does not record these events (an older release)
# only logs its failures. If no event is found, its pod log is searched for the
# errors instead, as the playbook always did: the ERROR lines, except the ones
# about validation, waiting and reconciler errors, one per distinct (10th, last)
# field. They are printed as logged, with no reason or count.
#
# Usage:
#   dm-monitor-errors.sh <namespace> [<since>]
#
# <namespace> is the namespace of the deployment manager. If <since> is
# given, a UTC time formatted as "date -u +%Y-%m-%dT%H:%M:%SZ", only the
# failures seen since then are printed.
#
# Prints one line per failure, oldest first:
#   <last seen> <count> <reason> <resource>: <message>
# or, for the errors found in the pod log, the log line.
# Exits with a non-zero status if neither the events nor the pod log can be
# read.

set -u -o pipefail

namespace=${1}
since=${2:-}

export KUBECONFIG=${KUBECONFIG:-/etc/kubernetes/admin.conf}

readonly DEPLOYMENT="deployments/platform-deployment-manager"
readonly CONTAINER="manager"
readonly FIELDS="TIME:.lastTimestamp,COUNT:.count,REASON:.reason,MESSAGE:.message"

# Prints the failures recorded as events. kubectl cannot select several
# reasons at once, hence the filter on the REASON column. The timestamps have
# the format of <since>, so they compare as strings.
event_failures() {
    kubectl -n "${namespace}" get events \
        --field-selector type=Warning,involvedObject.kind=Deployment \
        --sort-by=.lastTimestamp --no-headers -o custom-columns="${FIELDS}" |
        awk -v since="${since}" '$3 ~ /^SystemAPI/ && $1 >= since'
}

# Prints the errors found in the pod log, as they were logged.
log_failures() {
    local logs
    # --since-time keeps the errors of a previous run out of the report.
    logs=$(kubectl -n "${namespace}" logs "${DEPLOYMENT}" "${CONTAINER}" \
        ${since:+--since-time="${since}"} 2>&1) || {
        echo "${logs}" >&2
        return 1
    }

    awk '/ERROR/ &&
        !/validation/ &&
        !/waiting/ &&
        !/Reconciler error/ &&
        !(seen[$10, $NF]++)' <<<"${logs}"
}

events=$(event_failures)
events_failed=$?
if [ "${events_failed}" -eq 0 ] && [ -n "${events}" ]; then
    echo "${events}"
    exit 0
fi
# Nothing was recorded as an event, or the events cannot be read: this is
# expected with an older deployment manager. Look in its log instead.

if log_failures; then
    exit 0
fi

# The log is not available when the pods are drained by the unlock. That is not
# an error if the events could be read, they were just empty.
[ "${events_failed}" -eq 0 ] && exit 0
echo "neither the events nor the log of the deployment manager can be read" >&2
exit 1
