#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright(c) 2026 Wind River Systems, Inc.
#
# Collects the reasons why a host has not been unlocked yet.
#
# The deployment manager unlocks the host once the configuration has been
# applied. The host is not unlocked if sysinv rejects part of the
# configuration, and sysinv may reject the unlock itself while puppet is still
# applying parts of the configuration (SR-IOV interfaces, host cpus, ...). The
# deployment manager retries in both cases, and records each failed request as
# a Warning event, see dm-monitor-errors.sh.
#
# This script is started detached by roles/common/lock-unlock before the
# playbook starts waiting for the unlock reboot, because that wait blocks and
# cannot report anything while it runs. The playbook reports what was
# collected here once the wait is over. Only the events are read, so the
# collector never interferes with the unlock itself.
#
# The collector is killed by the unlock reboot, by the playbook, or by the
# ansible async timeout, whichever comes first.
#
# Usage:
#   dm-unlock-monitor.sh <hostname> <namespace> <delay> <interval> <logfile> \
#       [<since>]
#
# <namespace> and <since> are passed to dm-monitor-errors.sh, which must
# be in the directory of this script. No failure is reported during the first
# <delay> seconds: sysinv rejecting the unlock while the configuration is being
# applied is expected. The failures are then checked every <interval> seconds,
# and a line is appended to <logfile> whenever the reported status changes.

set -u

hostname=${1}
namespace=${2}
delay=${3}
interval=${4}
logfile=${5}
since=${6:-}

failures_cmd="$(dirname "${0}")/dm-monitor-errors.sh"

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

    if ! failures=$("${failures_cmd}" "${namespace}" "${since}" 2>&1); then
        # Expected whenever the kube-apiserver is unreachable. Keep polling.
        status="the system API failures are not available: $(head -1 <<<"${failures}")"
    elif [ -z "${failures}" ]; then
        status="${hostname} is not unlocked yet and the deployment manager"
        status="${status} reported no system API failure"
    else
        status="${hostname} is not unlocked yet, requests that failed on the"
        status="${status} system API (reason, resource, message):"
        # Without the time and count of each failure, which change on every
        # retry: only a new failure is worth reporting.
        while read -r _ _ failure; do
            status="${status}"$'\n'"    ${failure}"
        done <<<"${failures}"
    fi

    # Only report changes: the status is otherwise repeated every interval
    # until the host is unlocked.
    if [ "${status}" != "${previous}" ]; then
        log "after ${elapsed}s: ${status}"
        previous="${status}"
    fi
done
