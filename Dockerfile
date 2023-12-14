# Copyright 2021-present Open Networking Foundation
#
# SPDX-License-Identifier: Apache-2.0
#

FROM golang:1.21.3-bookworm AS builder

LABEL maintainer="ONF <omec-dev@opennetworking.org>"

#RUN apt remove cmdtest yarn
RUN apt-get update && apt-get -y install apt-transport-https ca-certificates
RUN apt-get update
RUN apt-get -y install gcc cmake autoconf libtool pkg-config libmnl-dev libyaml-dev
RUN apt-get clean
# Copy into path

RUN cd $GOPATH/src && mkdir -p nssf
COPY . $GOPATH/src/nssf

RUN cd $GOPATH/src/nssf \
    && make all

FROM alpine:3.16 as nssf

LABEL description="ONF open source 5G Core Network" \
    version="Stage 3"

ARG DEBUG_TOOLS

# Install debug tools ~ 100MB (if DEBUG_TOOLS is set to true)
RUN apk update && apk add -U vim strace net-tools curl netcat-openbsd bind-tools

# Set working dir
WORKDIR /free5gc
RUN mkdir -p nssf/

# Copy executable and default certs
COPY --from=builder /go/src/nssf/bin/* ./nssf
WORKDIR /free5gc/nssf
