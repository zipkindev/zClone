ARG BASE_IMAGE=zclone/zclone:latest
FROM ${BASE_IMAGE} as binaries

# build plugin image
FROM alpine:latest

# put zclone in /usr/bin, reserve /usr/local/bin for plugin wrappers
COPY --from=binaries /usr/local/bin/zclone /usr/bin/zclone

RUN mkdir -p /data/config /data/cache /mnt \
 && apk --no-cache add ca-certificates fuse3 tzdata \
 && echo "user_allow_other" >> /etc/fuse.conf \
 && zclone version

ENV ZCLONE_CONFIG=/data/config/zclone.conf
ENV ZCLONE_CACHE_DIR=/data/cache
ENV ZCLONE_BASE_DIR=/mnt
ENV ZCLONE_VERBOSE=0

ENV HTTP_PROXY=
ENV HTTPS_PROXY=
ENV NO_PROXY=

WORKDIR /data
ENTRYPOINT ["zclone", "serve", "docker"]
