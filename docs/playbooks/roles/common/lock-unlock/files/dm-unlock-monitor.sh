#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright(c) 2026 Wind River Systems, Inc.
#
# Collects the reasons why a host has not been unlocked yet.
#
# The deployment manager unlocks the host once the configuration has been
# applied, and sysinv may reject that unlock while puppet is still applying
# parts of the configuration (SR-IOV interfaces, host cpus, ...). The
# deployment manager retries, so those rejections are only visible in its pod
# log.
#
# This script is started detached by roles/common/lock-unlock before the
# playbook starts waiting for the unlock reboot, because that wait blocks and
# cannot report anything while it runs. The playbook reports what was
# collected here once the wait is over. Only the pod log is read, so the
# collector never interferes with the unlock itself.
#
# The collector is killed by the unlock reboot, by the playbook, or by the
# ansible async timeout, whichever comes first.
#
# Usage:
#   dm-unlock-monitor.sh <hostname> <pod-namespace> <delay> <interval> <logfile>
#
# No rejection is reported during the first <delay> seconds: sysinv rejecting
# the unlock while the configuration is being applied is expected. The pod log
# is then checked every <interval> seconds, and a line is appended to
# <logfile> whenever the reported status changes.

set -u

hostname=${1}
pod_namespace=${2}
delay=${3}
interval=${4}
logfile=${5}

export KUBECONFIG=/etc/kubernetes/admin.conf

readonly DEPLOYMENT="deployments/platform-deployment-manager"
readonly CONTAINER="manager"

# Logged by the deployment manager for every unlock request it sends.
readonly ATTEMPT_PATTERN="unlocking host"

# A rejected unlock is logged either as
#   request rejected by system API {..., "operation": "unlock host <name>",
#   "status": 400, "reason": "<reason>"}
# or, by a deployment manager built before commit 07fd868c, as
#   Failed to unlock <name>: <reason>
# Both are matched: the build running on the target is not known here, and the
# pod log may hold neither (no rejection logged, or the log was rotated).
readonly REJECTION_PATTERN="unlock host ${hostname}\""
readonly LEGACY_REJECTION_PATTERN="Failed to unlock ${hostname}: "

# Start fresh: a log left behind by a previous (failed) run must not be
# reported as the result of this one.
: >"${logfile}" || exit 1

# Used by the playbook to stop the collector. Matching the command line
# instead (pkill -f) would also match any other process that happens to
# mention this script, including the playbook invoking it.
pidfile="${logfile}.pid"
echo $$ >"${pidfile}"
trap 'rm -f "${pidfile}"' EXIT

log() {
    printf '%s  %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "${1}" >>"${logfile}"
}

# Prints the reason reported by sysinv for each rejected unlock of this host
# found in "${logs}", in the order they were logged, one per line.
rejections() {
    {
        grep -F "${REJECTION_PATTERN}" <<<"${logs}" |
            sed -nE 's/.*"reason": ?"(.*)"\}[[:space:]]*$/\1/p'
        grep -F "${LEGACY_REJECTION_PATTERN}" <<<"${logs}" |
            sed -E "s/.*${LEGACY_REJECTION_PATTERN}//"
    } |
        # The reasons are embedded in a JSON object, and a reason reported by
        # sysinv can itself quote a value.
        sed -e 's/\\"/"/g' -e 's/[[:space:]]*$//' |
        grep -v '^$'
}

start=$(date +%s)
previous=""

# Nothing is written until there is something to report, so that a host that
# is unlocked right away leaves an empty log and nothing to show.
while true; do
    sleep "${interval}"

    elapsed=$(($(date +%s) - start))
    if [ "${elapsed}" -lt "${delay}" ]; then
        continue
    fi

    # Read a bit more than the monitored window so that the first report also
    # covers the unlock attempts made before the collector was started.
    if ! logs=$(kubectl -n "${pod_namespace}" logs "${DEPLOYMENT}" "${CONTAINER}" \
        --since="$((elapsed + 60))s" 2>&1); then
        # Expected while the unlock drains the pods, and whenever the
        # kube-apiserver is unreachable. Keep polling.
        status="the deployment manager pod log is not available (the pods may"
        status="${status} have been drained by the unlock): $(head -1 <<<"${logs}")"
    else
        attempts=$(grep -c -F "${ATTEMPT_PATTERN}" <<<"${logs}")
        reasons=$(rejections)

        if [ -n "${reasons}" ]; then
            status="unlock of ${hostname} rejected $(grep -c . <<<"${reasons}")"
            status="${status} time(s) out of ${attempts} attempt(s), reason(s)"
            status="${status} reported by sysinv:"
            # Distinct reasons only: the same one is reported on every retry.
            while read -r reason; do
                status="${status}"$'\n'"    ${reason}"
            done <<<"$(awk '!seen[$0]++' <<<"${reasons}")"
        else
            status="${hostname} is not unlocked yet and the deployment manager"
            status="${status} logged no unlock rejection (${attempts} unlock"
            status="${status} attempt(s) so far)"
            # Without a rejection to report, the most recent errors logged by
            # the deployment manager are the only hint available.
            errors=$(grep -E '(^|[[:space:]])ERROR([[:space:]]|$)' <<<"${logs}" |
                awk '!seen[$0]++' | tail -5)
            if [ -n "${errors}" ]; then
                status="${status}, last error(s) logged:"
                while read -r line; do
                    status="${status}"$'\n'"    ${line}"
                done <<<"${errors}"
            fi
        fi
    fi

    # Only report changes: the status is otherwise repeated every interval
    # until the host is unlocked.
    if [ "${status}" != "${previous}" ]; then
        log "after ${elapsed}s: ${status}"
        previous="${status}"
    fi
done
