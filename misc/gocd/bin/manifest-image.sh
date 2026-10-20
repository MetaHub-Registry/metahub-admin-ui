#!/bin/bash
# Assemble the multi-arch metahub-admin-ui manifest from the per-arch images
# the parallel Image jobs (Amd64/Arm64) pushed, and push it under the plain
# tag.
#
# Consumes the two result.yaml artifacts (fetched to amd64_result/ and
# arm64_result/) and publishes its own result.yaml with the manifest-list ref
# in metahub.image.full — that is what the Deploy (GitSync) and AwaitRollout
# stages consume.
#
# `docker manifest` talks to the registry from the CLI itself (not dockerd),
# so it needs --insecure for plain-HTTP harbor regardless of the dockerd
# INSECURE_REGISTRIES config.
set -e

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

for f in amd64_result/result.yaml arm64_result/result.yaml; do
   if [[ ! -f "${f}" ]]; then
      echo "ERROR: ${f} missing — fetch the per-arch artifacts first"
      exit 1
   fi
done

FULL_AMD64=$(yq -r .metahub.image.full amd64_result/result.yaml)
FULL_ARM64=$(yq -r .metahub.image.full arm64_result/result.yaml)
NAME=$(yq -r .metahub.image.name amd64_result/result.yaml)
TAG=$(yq -r .metahub.image.tag amd64_result/result.yaml)
REG=$(yq -r .metahub.image.reg amd64_result/result.yaml)
# guard: both legs must be of the same pipeline run
NAME_ARM=$(yq -r .metahub.image.name arm64_result/result.yaml)
TAG_ARM=$(yq -r .metahub.image.tag arm64_result/result.yaml)
if [[ "${NAME}" != "${NAME_ARM}" || "${TAG}" != "${TAG_ARM}" ]]; then
   echo "ERROR: per-arch artifacts disagree (name/tag): ${NAME}:${TAG} vs ${NAME_ARM}:${TAG_ARM}"
   exit 1
fi

NEW_IMAGE=$(echo ${NAME}:${TAG} | tr '[:upper:]' '[:lower:]')
echo "> Manifest: ${NEW_IMAGE} <- ${FULL_AMD64} + ${FULL_ARM64}"

if [[ -z "${HARBOR_REG}" || -z "${HARBOR_TOKEN}" ]]; then
   echo "ERROR: HARBOR_REG/HARBOR_TOKEN required to push the manifest"
   exit 1
fi
echo "> Login to ${HARBOR_REG} as 'robot\$good'"
docker login ${HARBOR_REG} -u 'robot$good' -p "${HARBOR_TOKEN}"

export DOCKER_CLI_EXPERIMENTAL=enabled
set -x
# clean any stale local manifest list from a retried job
docker manifest rm "${NEW_IMAGE}" >/dev/null 2>&1 || true
docker manifest create --insecure "${NEW_IMAGE}" "${FULL_AMD64}" "${FULL_ARM64}"
docker manifest push --insecure "${NEW_IMAGE}"

# result.yaml does not exist in this (clean) workspace — gdh patches an
# existing file.
touch result.yaml
gdh patch -f result.yaml \
   --add=metahub.image.reg=${REG} \
   --add=metahub.image.name=${NAME} \
   --add=metahub.image.tag=${TAG} \
   --add=metahub.image.full=${NEW_IMAGE} \
   --add=metahub.image.arch=multi
