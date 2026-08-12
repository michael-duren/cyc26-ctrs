# syntax=docker/dockerfile:1
#
# App image for the container-isolation talk. Built on the `cyc26node` base
# image (see Dockerfile), which already has the evilnode probe installed as
# `node`. This image adds the sample app and the run command.
#
# Requires the base image to exist locally first:
#
#     docker build -t cyc26node .
#     docker build -t cyc26node-app:cyc -f Dockerfile.app .
#
# THEME picks which branded server to run (webapp/server.<theme>.js):
#
#     docker build --build-arg THEME=nagios -t cyc26node-app:nagios -f Dockerfile.app .
#
FROM cyc26node

# cyc = Commit Your Code theme, nagios = Nagios theme
ARG THEME=cyc

# --- sample application (zero-dependency http server) ---
WORKDIR /app
COPY webapp/ /app/

# Pick the themed server; `cp` fails the build on an unknown THEME.
RUN cp "/app/server.${THEME}.js" /app/server.js

# Because `node` is the probe, this runs the escape probe first (writing the
# log), then hands off to the real interpreter to start the server -- which
# then displays that very log.
CMD ["node", "/app/server.js"]
