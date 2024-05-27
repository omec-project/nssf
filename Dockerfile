# Copyright 2021-present Open Networking Foundation
# Copyright 2024-present Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0
#

FROM golang:1.22.3-bookworm AS builder

LABEL maintainer="Aether SD-Core <dev@aetherproject.org>"

RUN apt-get update && \
    apt-get -y install --no-install-recommends \
    apt-transport-https \
    ca-certificates \
    gcc \
    cmake \
    autoconf \
    libtool \
    pkg-config \
    libmnl-dev \
    libyaml-dev && \
    apt-get clean

WORKDIR $GOPATH/src/nssf
COPY . .
RUN make all

FROM alpine:3.20 as nssf

LABEL description="ONF open source 5G Core Network" \
    version="Stage 3"

ARG DEBUG_TOOLS

# Install debug tools ~ 50MB (if DEBUG_TOOLS is set to true)
RUN if [ "$DEBUG_TOOLS" = "true" ]; then \
        apk update && apk add --no-cache -U vim strace net-tools curl netcat-openbsd bind-tools; \
        fi

# Set working dir
WORKDIR /free5gc/nssf

# Copy executable and default certs
COPY --from=builder /go/src/nssf/bin/* .
