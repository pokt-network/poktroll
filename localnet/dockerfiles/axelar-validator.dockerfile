# syntax=docker/dockerfile:experimental

# Derived from https://github.com/axelarnetwork/axelar-core/blob/v1.2.1/Dockerfile

FROM alpine:3.18 as build

ARG GO_VERSION=1.23.1
ARG TARGETARCH
ARG ARCH=${TARGETARCH}
ARG WASM=true
ARG IBC_WASM_HOOKS=false

# Install necessary packages
RUN apk add --no-cache --update \
    curl \
    git \
    make \
    tar \
    build-base \
    ca-certificates \
    linux-headers

# Download and install Go
RUN case "${TARGETARCH}" in \
        "amd64") GO_ARCH="amd64" ;; \
        "arm64") GO_ARCH="arm64" ;; \
        *) echo "Unsupported architecture: ${TARGETARCH}" && exit 1 ;; \
    esac && \
    curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz -o golang.tar.gz \
    && tar -C /usr/local -xzf golang.tar.gz \
    && rm golang.tar.gz

# Set Go paths
ENV GOROOT=/usr/local/go
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:$GOROOT/bin:$PATH

RUN git clone --branch=v1.2.1 --depth=1 https://github.com/axelarnetwork/axelar-core.git /axelar
WORKDIR /axelar
RUN go mod download

# Use a compatible libwasmvm
# Alpine Linux requires static linking against muslc: https://github.com/CosmWasm/wasmd/blob/v0.34.1/INTEGRATION.md#prerequisites
RUN if [[ "${WASM}" == "true" ]]; then \
    case "${TARGETARCH}" in \
        "amd64") WASMVM_ARCH="x86_64" ;; \
        "arm64") WASMVM_ARCH="aarch64" ;; \
        *) echo "Unsupported architecture: ${TARGETARCH}" && exit 1 ;; \
    esac && \
    WASMVM_VERSION=v1.5.8 && \
    wget https://github.com/CosmWasm/wasmvm/releases/download/${WASMVM_VERSION}/libwasmvm_muslc.${WASMVM_ARCH}.a \
        -O /lib/libwasmvm_muslc.a && \
    wget https://github.com/CosmWasm/wasmvm/releases/download/${WASMVM_VERSION}/checksums.txt -O /tmp/checksums.txt && \
    sha256sum /lib/libwasmvm_muslc.a | grep $(cat /tmp/checksums.txt | grep libwasmvm_muslc.${WASMVM_ARCH}.a | cut -d ' ' -f 1); \
    fi

RUN case "${TARGETARCH}" in \
        "amd64") MAKE_ARCH="x86_64" ;; \
        "arm64") MAKE_ARCH="aarch64" ;; \
        *) echo "Unsupported architecture: ${TARGETARCH}" && exit 1 ;; \
    esac && \
    make VERSION="1.2.1" ARCH="${MAKE_ARCH}" MUSLC="${WASM}" WASM="${WASM}" IBC_WASM_HOOKS="${IBC_WASM_HOOKS}" build

FROM alpine:3.18

ARG USER_ID=1000
ARG GROUP_ID=1001
RUN apk add --no-cache jq bash
COPY --from=build /axelar/bin/* /usr/local/bin/
RUN addgroup -S -g ${GROUP_ID} axelard && adduser -S -u ${USER_ID} axelard -G axelard
USER axelard
COPY --from=build /axelar/entrypoint.sh /entrypoint.sh

# The home directory of axelar-core where configuration/genesis/data are stored
ENV HOME_DIR /home/axelard
# Host name for tss daemon (only necessary for validator nodes)
ENV TOFND_HOST ""
# The keyring backend type https://docs.cosmos.network/master/run-node/keyring.html
ENV AXELARD_KEYRING_BACKEND file
# The chain ID
ENV AXELARD_CHAIN_ID axelar-testnet-lisbon-3
# The file with the peer list to connect to the network
ENV PEERS_FILE ""
# Path of an existing configuration file to use (optional)
ENV CONFIG_PATH ""
# A script that runs before launching the container's process (optional)
ENV PRESTART_SCRIPT ""
# The Axelar node's moniker
ENV NODE_MONIKER ""

# Create these folders so that when they are mounted the permissions flow down
RUN mkdir /home/axelard/.axelar && chown axelard /home/axelard/.axelar
RUN mkdir /home/axelard/shared && chown axelard /home/axelard/shared
RUN mkdir /home/axelard/genesis && chown axelard /home/axelard/genesis
RUN mkdir /home/axelard/scripts && chown axelard /home/axelard/scripts
RUN mkdir /home/axelard/conf && chown axelard /home/axelard/conf

ENTRYPOINT ["/entrypoint.sh"]
