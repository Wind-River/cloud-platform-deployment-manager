#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Copyright(c) 2019 Wind River Systems, Inc

# This is a utility script that launches the target program and then attaches
# the delve debugger to it.  If the "WAIT" environment variable is set to
# "true" then the program is suspended until a debugger client attaches to it;
# otherwise the program is allowed to run and then a debugger client can attach
# to it at a later time.
#
# The program is started independent of the debugger rather than within the
# debugger (as a subprogram thereof) because if the program crashes we want
# the pod to terminate and restart.  When the program is a subprogram of the
# debugger the debugger does not exit when the program terminates.
#

DLVPATH=${DLVPATH}
SERVER=${SERVER:-"127.0.0.1"}
PORT=${PORT:-40000}
WAIT=${WAIT:-"false"}
MULTICLIENT=${MULTICLIENT:-1}
APIVERSION=${APIVERSION:-2}
HEADLESS=${HEADLESS:-true}

trap cleanup EXIT INT QUIT TERM

if [[ "${DLVPATH}" == "" ]]; then
    # Use the automatic path if available.
    DLVPATH=$(which dlv)
    if [[ "${DLVPATH}" == "" ]]; then
        # Otherwise, default to the expected image path
        DLVPATH="/dlv"
    fi
fi

# Kills the inject background task if it exists.
cleanup () {
    kill %2 > /dev/null 2>1
    kill %1 > /dev/null 2>1
    return 0
}

# Injects a "continue" command into the local debug session so that it starts
# the process being debugged without needing to wait for a real debugger to
# connect.
inject_continue () {
    local RET=1
    while [[ ${RET} -ne 0 ]]; do
        ${DLVPATH} --init <(echo "exit -c") connect ${SERVER}:${PORT} >/dev/null 2>1
        RET=$?

        if [[ ${RET} -ne 0 ]]; then
           sleep 5
        fi
    done

    echo "debugger continued; exiting"

    return 0
}

# Attach the debugger to the target program.  This method is preferred over the
# "exec" method because when using "exec" the debugger will not terminate if the
# program terminates.  When running inside of a container the termination of
# the program needs to propagate to the death of the container so that it is
# restarted.
attach_debugger () {
    local TARGETPID=$1
    local MULTICLIENT=$2
    local PROGRAM=$3

    if [[ ${MULTICLIENT} -ne 0 ]]; then
        ACCEPTMULTI="--accept-multiclient"
    else
        ACCEPTMULTI=""
    fi

    ${DLVPATH} --listen=:${PORT} --headless=${HEADLESS} --api-version=${APIVERSION} ${ACCEPTMULTI} attach ${TARGETPID} ${PROGRAM} &
    local DLVPID=$!

    # Give it time to start
    sleep 0.5

    # Test that it is running
    kill -0 ${DLVPID} > /dev/null 2>1
    if [[ $? -ne 0 ]]; then
        RET=$?
        echo "Debugger did not start: ${RET}"
        return ${RET}
    fi

    return 0
}

# Start the target program in the background
echo "Starting: $@"
$@ &
TARGETPID=$!

# Attach the debugger to the target program.  There is a race condition here
# in that the program gets to run for some amount of time before the debugger
# attaches to it so it may not be possible to debug the entry point of the
# program, but in practice this seems to catch it early enough that except
# for debugging the lowest-level issues this should be fine for application
# startup debugging.
attach_debugger ${TARGETPID} ${MULTICLIENT} $1
if [[ $? -ne 0 ]]; then
    exit $?
fi

if [[ ${WAIT,,} != "true" ]]; then
    # Do not wait for a debugger client to connect.  Release the program now.
    inject_continue &
fi

# Test that the target program is running
kill -0 ${TARGETPID} > /dev/null 2>1
if [[ $? -ne 0 ]]; then
    RET=$?
    echo "Target program did not start: ${RET}"
    exit ${RET}
fi

wait ${TARGETPID}
exit $?
