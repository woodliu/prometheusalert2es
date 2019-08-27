FROM docker.io/alpine:latest
COPY prometheusalert2es /usr/local/bin/
ENTRYPOINT ["prometheusalert2es"]