#!/bin/bash
#
# Copyright (c) 2023 Wind River Systems, Inc.
#
# SPDX-License-Identifier: Apache-2.0
#

set -x #echo on

#
# This script automates DM patching to 
# 1. pull image from configured source
# 2. push image to local registry 
# 3. run anisible playbook to refresh DM 

#
# Step-1: Set variables for the image and tags, including the reference for the Wind River repo on the Wind River Registry.
#
echo -e "\nStep-1: Set variables for the image and tags, including the reference for the Wind River repo on the Wind River Registry."
export MY_MIRROR=admin-2.cumulus.wrs.com:30093
export REGISTRY_LOCAL=registry.local:9001

#
# Step-2: Set tags for the image being updated, defining the new image tags.
#
echo -e "\nStep-2: Set tags for the image being updated, defining the new image tags."
export MY_IMAGE=wind-river/cloud-platform-deployment-manager
export MY_IMAGE_TAG=WRCP_22.12-wrs.5

#
# Step-3: For Standard controllers, you may need to perform a docker login to prevent an authentication error on the docker push command below.
# When prompted, use admin as the username and your keystone admin password.
#
echo -e "\nStep-3: For Standard controllers, you may need to perform a docker login to prevent an authentication error on the docker push command below."
sudo docker login registry.local:9001

#
# Step-4: Pull the new image to docker, tag it, and then push it to the local registry.
#
echo -e "\nStep-4: Pull the new image to docker, tag it, and then push it to the local registry."
sudo docker pull ${MY_MIRROR}/${MY_IMAGE}:${MY_IMAGE_TAG}
export MY_LOCAL_IMAGE=docker.io/wind-river/cloud-platform-deployment-manager
sudo docker tag ${MY_MIRROR}/${MY_IMAGE}:${MY_IMAGE_TAG} ${REGISTRY_LOCAL}/${MY_LOCAL_IMAGE}:${MY_IMAGE_TAG}
sudo docker push ${REGISTRY_LOCAL}/${MY_LOCAL_IMAGE}:${MY_IMAGE_TAG}
sudo crictl pull --creds "admin:Li69nux*" ${REGISTRY_LOCAL}/${MY_LOCAL_IMAGE}:${MY_IMAGE_TAG}

#
# Step-5: Clean up docker.
#
echo -e "\nStep-5: Clean up docker."
sudo docker rmi ${MY_MIRROR}/${MY_IMAGE}:${MY_IMAGE_TAG} ${REGISTRY_LOCAL}/${MY_LOCAL_IMAGE}:${MY_IMAGE_TAG}

#
# Step-6: Check the running pod location and state.
#
echo -e "\nStep-6: Check the running pod location and state."
kubectl get pods -n platform-deployment-manager -o wide

#
# Step-7: Check the image tags.
#
echo -e "\nStep-7: Check the image tags."
dm_pod=`kubectl -n platform-deployment-manager get pods | grep platform-deployment-manager- | awk 'NR == 1 { print $1 }'`
kubectl describe pod -n platform-deployment-manager $dm_pod | grep Image

#
# Step-8: Set environment variables for overrides and charts.
#
echo -e  "\nStep-8: Set environment variables for overrides and charts."
export MY_IMG_OVERRIDES=helm-chart-overrides.yaml
export MY_NEW_TARBALL=/usr/local/share/applications/helm/wind-river-cloud-platform-deployment-manager-2.0.10.tgz

#
# Step-9: Create/Copy an helm-chart-overrides.yaml overrides file for the new image.
#
echo -e "\nStep-9: Create/Copy an helm-chart-overrides.yaml overrides file for the new image."
cp /usr/local/share/applications/overrides/wind-river-cloud-platform-deployment-manager-overrides.yaml helm-chart-overrides.yaml
sed -i -e "s/tag: latest/tag: ${MY_IMAGE_TAG}/g" helm-chart-overrides.yaml

#
# Step-10: Transfer the charts tarball to your controller.
# use from the load: /usr/local/share/applications/helm/wind-river-cloud-platform-deployment-manager-2.0.10.tgz
#
echo -e "\nStep-10: Transfer the charts tarball to your controller."

#
# Step-11: Update the running image and charts.
#
echo -e "\nStep-11: Update the running image and charts."
helm upgrade --install deployment-manager --values ${MY_IMG_OVERRIDES} ${MY_NEW_TARBALL}

#
# Step-12: Check the location and state of the updated pod.
#
kubectl get pods -n platform-deployment-manager -o wide

#
# Step-13: Check the image of the updated pod.
#
dm_pod_after_refresh=`kubectl -n platform-deployment-manager get pods | grep platform-deployment-manager- | awk 'NR == 1 { print $1 }'`
kubectl describe pod -n platform-deployment-manager $dm_pod_after_refresh| grep Image
exit 0

