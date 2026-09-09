# Supply approved, locally preloaded images explicitly. Omitting either value
# fails before a builder can contact a registry.
ARG GO_IMAGE
ARG RUNTIME_IMAGE
FROM ${GO_IMAGE} AS builder

ARG CGO_ENABLED=0
ARG INSTALL_BUILD_DEPS=0

WORKDIR /go/src/zclone/

RUN echo "**** Set Go Environment Variables ****" && \
    go env -w GOCACHE=/root/.cache/go-build

RUN if [ "$INSTALL_BUILD_DEPS" = 1 ]; then \
		apk add --no-cache make bash gawk git; \
	fi

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    echo "**** Build Binary ****" && \
    make zclone

RUN echo "**** Print Version Binary ****" && \
    ./build/zclone version

# Begin final image
FROM ${RUNTIME_IMAGE}
ARG INSTALL_RUNTIME_DEPS=0

RUN if [ "$INSTALL_RUNTIME_DEPS" = 1 ]; then \
		apk add --no-cache ca-certificates fuse3 tzdata; \
	fi && \
	if [ -f /etc/fuse.conf ]; then echo "user_allow_other" >> /etc/fuse.conf; fi

COPY --from=builder /go/src/zclone/build/zclone /usr/local/bin/zclone

RUN addgroup -g 1009 zclone && adduser -u 1009 -Ds /bin/sh -G zclone zclone

ENTRYPOINT [ "zclone" ]

WORKDIR /data
ENV XDG_CONFIG_HOME=/config
