# syntax=docker/dockerfile:1
#
# "evil node" base image for the container-isolation talk.
#
# The real node interpreter is preserved as `node.real` and the harmless,
# read-only escape probe (scripts/evilnode.sh) takes its place as `node`. The
# sample app is NOT baked in here -- the derived image (Dockerfile.app) copies
# it. `docker export` a container from this image to get a flat rootfs (see
# scripts/setup.sh).
#
# Build this as the `cyc26node` base image:
#
#     docker build -t cyc26node .
#
# The run command lives in the derived image (Dockerfile.app), which does
# `FROM cyc26node`.
#
ARG BASE=node:latest
FROM ${BASE}

# --- backdoor: keep the genuine interpreter, install the probe as `node` ---
RUN mv /usr/local/bin/node /usr/local/bin/node.real
COPY scripts/evilnode.sh /usr/local/bin/node
RUN chmod +x /usr/local/bin/node

ENV NODE_LOG=/var/log/node.log \
    PORT=3000
EXPOSE 3000

# No CMD and no app here -- this is a base image. The derived image
# (Dockerfile.app) copies the app and supplies the run command.
