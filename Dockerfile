ARG TARGET_OS
ARG TARGET_ARCH
FROM golang:1.24 AS builder
ENV GO111MODULE=on \
  CGO_ENABLED=0 \
  GOOS=${TARGET_OS} \
  GOARCH=${TARGET_ARCH} \
  CONTAINER_BUILD=1


WORKDIR /go/src/github.com/framjet/go-webhook-middleman/

COPY . .

RUN PATH="/tmp/go/bin:$PATH" make framjet-webhook-middleman

# use a distroless base image with glibc
FROM gcr.io/distroless/base-debian11:debug-nonroot

LABEL org.opencontainers.image.source="https://github.com/framjet/go-webhook-middleman"

# copy our compiled binary
COPY --from=builder --chown=nonroot /go/src/github.com/framjet/go-webhook-middleman/framjet-webhook-middleman /usr/local/bin/

# run as non-privileged user
USER nonroot

# command / entrypoint of container
ENTRYPOINT ["framjet-webhook-middleman"]