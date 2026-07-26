FROM alpine:3.21

ARG PLUGIN_VERSION

LABEL org.opencontainers.image.source="https://github.com/SlothCroissant/traefik-queryparam-plugin"
LABEL org.opencontainers.image.version="${PLUGIN_VERSION}"

COPY . /plugin
COPY docker-entrypoint.sh /usr/local/bin/copy-plugin

RUN rm /plugin/docker-entrypoint.sh && chmod 0555 /usr/local/bin/copy-plugin

ENTRYPOINT ["/usr/local/bin/copy-plugin"]
