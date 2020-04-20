# Debugging Issues

This document provides information on the best approaches to debug DM issues on
 a running system.

## Checking deployment-manager status
The following command will display the deployment-manager status:
```
kubectl -n deployment get datanetworks,hostprofiles,hosts,platformnetworks,systems
```

Of primary interest is the INSYNC column, which shows whether the deployment-manager
has been able to synchronize the deployment configuration with the system:
```
controller-0:~$ kubectl -n deployment get
datanetworks,hostprofiles,hosts,platformnetworks,systems
NAME                                                TYPE   INSYNC
datanetwork.starlingx.windriver.com/group0-data0    vlan   true
datanetwork.starlingx.windriver.com/group0-data0b   vlan   true
datanetwork.starlingx.windriver.com/group0-data1    vlan   true
datanetwork.starlingx.windriver.com/group0-ext0     vlan   true

NAME                                                     BASE
hostprofile.starlingx.windriver.com/controller-profile

NAME                                        ADMINISTRATIVE   OPERATIONAL
AVAILABILITY   PROFILE              INSYNC
host.starlingx.windriver.com/controller-0   unlocked         enabled       available
controller-profile   true

NAME                                  MODE      TYPE         VERSION   INSYNC
system.starlingx.windriver.com/vbox   simplex   all-in-one   19.12     true
```

## Looking at logs of the currently running Pod
The logs from the currently running Pod can be queried using the following 
command.  The "-f" argument follows the log stream much like the Linux "tail" 
command.
```
kubectl -n platform-deployment-manager logs platform-deployment-manager-0 manager -f
```

## Looking at logs of the previously running Pod
If the Pod crashes, is deleted, or restarts the logs from the previous
instantiation of the Pod are kept and can be accessed with the "-p" argument.  
But, if the Pod restarts multiple time the logs from the preceding 
instantiations are lost unless the platform is configured with a more advance
data collection mechanism.
 
```
kubectl -n platform-deployment-manager logs platform-deployment-manager-0 manager -p
```

## Increasing the log level
The DM log level can be increased by specifying the desired log level with the 
"--v" parameter when running the "manager" binary.  The manager Container can be 
modified to change the "manager" launch arguments with a custom log level.  The 
"Args:" section would look like the following if the log level was changed from 
the default (0) to a new value (2).

```yaml
Args:
/manager
--metrics-addr=127.0.0.1:8080
--alsologtostderr=true
--v=2
```

Alternatively, if the log level needs to be set to a non-default value before
the first instantiation, it can be set as a Helm chart override.  The following
sample Helm override file demonstrates how to set the log level to 2 in addition
to other attributes that may already be present in the file.  The Helm chart
override method can also be used to update an already running system to increase
or decrease the log level.

```yaml
manager:
  logLevel: 2
  image:
    repository: wind-river/cloud-platform-deployment-manager
    tag: latest
    pullPolicy: IfNotPresent
```

To re-apply a new set of overrides to an existing installation the Helm upgrade 
command can simply be re-executed.

```
helm upgrade --install deployment-manager --values deployment-manager-overrides.yaml wind-river-cloud-platform-deployment-manager-2.0.5.tgz
```

## Enabling version API interaction logs
If the problem being debugged involves looking at details of exact REST API
interactions with the StarlingX System API then more verbose logging can be
enabled.  To enable verbose API logging you must set the OS_DEBUG attribute
in the system-endpoint Secret resource which defines the System API endpoint
through which the DM will interact with the system being configured.  For
example, to enable more detailed logs the system-endpoint Secret would look
similar to the following.

```yaml
apiVersion: v1
data:
  OS_PASSWORD: TGk2OW51eCo=
  OS_USERNAME: YWRtaW4=
kind: Secret
metadata:
  name: system-endpoint
  namespace: deployment
stringData:
  OS_AUTH_URL: http://192.168.204.102:5000/v3
  OS_INTERFACE: internal
  OS_PROJECT_DOMAIN_NAME: Default
  OS_PROJECT_NAME: admin
  OS_REGION_NAME: RegionOne
  OS_DEBUG: True
type: Opaque
``` 

***Note:*** The OS_DEBUG value is parsed using standard Go libraries therefore
the value used must be understood as a boolean by ```strconv.ParseBool``` which
at the time of writing is "1", "t", "T", "true", "TRUE", and "True".

## Disabling individual sub-reconcilers
For debugging and isolation purposes each of the reconcilers implemented in the
DM is sub-divided into smaller "sub-reconciler" entities that can be selectively
enabled/disabled for debugging purposes.  This functionality does not provide
much usefulness for customer deployments but has been useful on occasion to
isolate problematic parts of the system so that the DM does not try to reconcile
its data.  

The DM consumes a ConfigMap at runtime which can contain individual boolean
values that control the state of each reconciler.  Any changes to the ConfigMap
are immediately read into the DM's internal configuration and go into effect the
next time the DM queries its internal state.   The set of supported values can
be found in the ```pkg/config/config.go``` file within the DM repo.  The
following Helm chart override file provides an example of how to set ConfigMap
values as an override to disable the Host Memory sub-reconciler.

```yaml
manager:
  image:
    repository: wind-river/cloud-platform-deployment-manager
    tag: latest
    pullPolicy: IfNotPresent
  configmap:
    reconcilers:
      host:
        memory:
          enabled: false
```

## Attaching a remote debugger
The GoLang ecosystem supports remote debugging.  The best resource available for
remote debugging at the moment is the Delve debugger.
 
    https://github.com/go-delve/delve.   

Like programs written in C, debugging a Go program requires compilation with
special flags.  The DM is deployed as a Docker image; therefore, a new image
needs to be built which contains a debug-enabled binary.  A debug image can be
built by setting the ```DEBUG=yes``` environment variable when invoking
```make```.  

For example, the following command will build a debug version of the DM image
and will tag it "debug" rather than "latest".

```bash
DEBUG=yes make docker-build
```

Attaching a remote debugger to the running Pod from outside of the Cluster
requires exposing a TCP port on the Container.   It may also require other
networking or firewall changes depending on the policies in place in your
environment.  The DM Helm chart is structured to make exposing this port and
configuring the environment for debugging possible through a simple Helm chart
override.  The following Helm chart override file provides an example of how to
configure debugging.

```yaml
manager:
  logLevel: 2
  debugger:
    enabled: true
    wait: false
    port: 30000
  image:
    repository: wind-river/cloud-platform-deployment-manager
    tag: debug
    pullPolicy: Always
```

The ```manager.debugger.enabled``` value controls whether a debug port is
exposed to make the Pod accessible from outside of the Cluster.

The ```manager.debugger.wait``` value controls whether the "manager" binary
is forced to stop and wait for a debugger to attach before running.  This is
handy if debugging an issue on initial startup and there is not sufficient time
to manually attach a debugger before the problem occurs.  This option is not
recommended until it is necessary to debug a startup issue - because it causes
webhooks to fail if a configuration change is requested before the debugger
can be attached.  This value defaults to ```false``` therefore it does not need
to be specified unless it is being set explicitly.

The ```manager.debugger.port``` value controls which port is exposed for
accessibility from outside of the cluster.  This value defaults to ```30000```
and does not need to be specified unless different from the default value.

## Suspending the Deployment Manager

In some scenarios, it may be necessary to stop the DM from processing changes
altogether in order to debug an issue or to correct a problem with the
underlying system API state.  You can adjust the replica count of the DM
statefulset to 0 to force its termination.  Reversing the change by setting the
replica count back to the normal value of 1 should relaunch the DM.

```bash
kubectl scale --replicas=0 -n platform-deployment-manager statefulset platform-deployment-manager
```

## Restarting the Deployment Manager

As a last resort, if the DM does not appear to be making progress or you wish to
force the DM to re-evaluate the state of all hosts you can elect to restart the
DM Pod.

```
kubectl -n platform-deployment-manager delete pods platform-deployment-manager-0
```

## Deleting the Deployment Manager

For testing purposes, it is sometimes necessary to remove all resources related
to the Deployment Manager.  If the DM was installed using the recommended Helm
chart install method then it can be removed using a similar operation.

```bash
helm delete --purge deployment-manager
```
