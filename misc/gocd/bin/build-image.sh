#!/bin/bash
# Build + push ONE ARCH of the metahub-admin-ui PRODUCT image to Harbor.
#
# The pipeline fans out: two parallel Image jobs (ARCH=amd64 / ARCH=arm64),
# each pinned to an arch-matched agent (k8s-dind-amd64 / k8s-dind-arm64), push
# ${NAME}:${TAG}-${ARCH}; a downstream Manifest stage assembles them into a
# multi-arch manifest list via `docker manifest create` (manifest-image.sh).
# This isolates per-arch failures, parallelizes the legs, and runs everything
# natively — no QEMU/binfmt, no buildx sidecar.
#
# The image carries the embedded SPA (dist/) and the Go service binary; the
# multi-stage Dockerfile's ui/go stages are pinned to $BUILDPLATFORM and the
# go stage cross-compiles via TARGETOS/TARGETARCH, so a plain native
# `docker build --platform linux/${ARCH}` on the arch-matched agent yields the
# correct binary for that arch.
set -e

if [[ -z "${ARCH}" ]]; then
   echo "ERROR: ARCH required (amd64|arm64) — the pipeline sets it per job"
   exit 1
fi
if [[ "${ARCH}" != "amd64" && "${ARCH}" != "arm64" ]]; then
   echo "ERROR: unsupported ARCH '${ARCH}' (amd64|arm64)"
   exit 1
fi

if [[ -f ~/gocd_func.sh ]]; then
   FUNC_SRC=~/gocd_func.sh
elif [[ -f /opt/qnib/gocd/src/gocd_func.sh ]]; then
   FUNC_SRC=/opt/qnib/gocd/src/gocd_func.sh
else
   echo "ERROR: No gocd_func.sh found"
   exit 1
fi
source ${FUNC_SRC}
dumpEnv

if [[ -z "${HARBOR_REG}" || -z "${HARBOR_TOKEN}" ]]; then
   echo "ERROR: HARBOR_REG/HARBOR_TOKEN required to push the admin-ui image"
   exit 1
fi
if [[ -z "${GOCD_GROUP}" || -z "${ADMINUI_BRANCH}" ]]; then
   echo "ERROR: GOCD_GROUP/ADMINUI_BRANCH required (pipeline environment_variables)"
   exit 1
fi

# Branch-based image path, matching the flux deployment and the metahub-ui
# convention: metahub/admin-ui/<group>/<branch>:<count>[-<arch>]
NEW_IMAGE_NAME=$(echo ${HARBOR_REG}/metahub/admin-ui/${GOCD_GROUP}/${ADMINUI_BRANCH} | tr '[:upper:]' '[:lower:]')
TAG="$(echo ${GO_PIPELINE_LABEL} | tr '[:upper:]' '[:lower:]')"
# Per-arch tag: the multi-arch manifest is assembled downstream (manifest-image.sh).
NEW_IMAGE=$(echo ${NEW_IMAGE_NAME}:${TAG}-${ARCH} | tr '[:upper:]' '[:lower:]')
echo "> NEW_IMAGE: ${NEW_IMAGE}"

echo "> Login to ${HARBOR_REG} as 'robot\$good'"
echo "HARBOR_TOKEN length: $(echo ${HARBOR_TOKEN}|wc -c)"
docker login ${HARBOR_REG} -u 'robot$good' -p "${HARBOR_TOKEN}"

set -x
# --- single-arch build on the arch-pinned agent ------------------------------
# Plain `docker build` is enough here: the job runs on an arch-matched agent
# (k8s-dind-amd64 / k8s-dind-arm64), so build AND final-stage COPYs execute
# natively — no QEMU/binfmt, no buildx sidecar. The default (docker) buildkit
# driver emits a plain single-platform manifest — no attestation index — which
# is exactly what `docker manifest create` needs as a source downstream.
# The push goes through dockerd, which has harbor in INSECURE_REGISTRIES
# (plain HTTP), so no buildkitd registry config is needed either.
export DOCKER_BUILDKIT=1
docker build --platform linux/${ARCH} -t ${NEW_IMAGE} .
docker push ${NEW_IMAGE}
# .full is the per-arch image; the Manifest stage publishes the manifest-list
# ref (no suffix) that the Deploy/AwaitRollout stages consume.
gdh patch -f result.yaml \
   --add=metahub.image.reg=${HARBOR_REG} \
   --add=metahub.image.name=${NEW_IMAGE_NAME} \
   --add=metahub.image.tag=${TAG} \
   --add=metahub.image.arch=${ARCH} \
   --add=metahub.image.full=${NEW_IMAGE}
