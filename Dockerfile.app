FROM cyc26node

ARG THEME=cyc

WORKDIR /app
COPY webapp/ /app/

RUN cp "/app/server.${THEME}.js" /app/server.js

CMD ["node", "/app/server.js"]
