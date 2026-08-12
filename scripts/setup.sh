#!/usr/bin/env bash
#
# Build the "evil node" images and export the app image's filesystem into
# ./rootfs so the box runtime can run it. Two stages: the base image
# (Dockerfile -> cyc26node) swaps the real node for the escape probe, and the
# app image (Dockerfile.app -> cyc26node-app) copies the webapp on top. We
# export the app image because that is the one that actually contains /app.
#
# The first argument picks the branding of the web app (webapp/server.<theme>.js):
#
#     bash scripts/setup.sh cyc       # Commit Your Code   (default)
#     bash scripts/setup.sh nagios    # Nagios
#
set -euo pipefail

C_GRN=$'\033[32m'
C_CYN=$'\033[36m'
C_YEL=$'\033[33m'
C_RED=$'\033[31m'
C_RST=$'\033[0m'

THEME=${1:-cyc}
case "${THEME}" in
cyc | nagios) ;;
*)
  echo -e "${C_RED}> unknown theme '${THEME}' -- expected 'cyc' or 'nagios'${C_RST}" >&2
  echo -e "  usage: bash scripts/setup.sh [cyc|nagios]" >&2
  exit 2
  ;;
esac

BASE_IMAGE=${BASE_IMAGE:-cyc26node}
APP_IMAGE=${APP_IMAGE:-cyc26node-app:${THEME}}
ROOTFS=${ROOTFS:-rootfs}
BASE=${BASE:-node:latest}

# Build context is the project root (Dockerfile needs scripts/, Dockerfile.app
# needs webapp/). .dockerignore keeps the exported rootfs out of the context.
echo -e "${C_CYN}> building ${BASE_IMAGE} base image (base: ${BASE})${C_RST}"
docker build --build-arg BASE="${BASE}" -t "${BASE_IMAGE}" .

echo -e "${C_CYN}> building ${APP_IMAGE} app image (theme: ${THEME})${C_RST}"
docker build --build-arg THEME="${THEME}" -t "${APP_IMAGE}" -f Dockerfile.app .

echo -e "${C_CYN}> exporting image filesystem -> ./${ROOTFS}/${C_RST}"
rm -rf "${ROOTFS}"
mkdir -p "${ROOTFS}"

cid="$(docker create "${APP_IMAGE}")"
trap 'docker rm -f "${cid}" >/dev/null 2>&1 || true' EXIT
docker export "${cid}" | tar -C "${ROOTFS}" -xf -

echo -e "${C_GRN}> done: ./${ROOTFS} ready (theme: ${THEME})${C_RST}"
echo -e "${C_YEL}  webapp     -> server.${THEME}.js"
echo -e "  node       -> escape probe (harmless, read-only)"
echo -e "  node.real  -> the genuine interpreter"
echo -e "  probe log  -> ${ROOTFS}/var/log/node.log (after it runs)${C_RST}"
