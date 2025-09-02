FROM ubuntu:24.04

ARG TARGETARCH

# Install system deps and Go
RUN apt-get update && apt-get install -y wget curl git build-essential ca-certificates

ENV PATH="/usr/local/go/bin:${PATH}"

# Install hermes
RUN mkdir -p /root/.hermes/bin
WORKDIR /usr/local
RUN case "${TARGETARCH}" in \
        "amd64") HERMES_ARCH="x86_64" ;; \
        "arm64") HERMES_ARCH="aarch64" ;; \
        *) echo "Unsupported architecture: ${TARGETARCH}" && exit 1 ;; \
    esac && \
    # Smart fallback: try pokt-network first, fallback to informalsystems
    if wget -q --spider "https://github.com/pokt-network/hermes/releases/download/v1.13.4/hermes-v1.13.4-${HERMES_ARCH}-unknown-linux-gnu.tar.gz" 2>/dev/null; then \
        echo "✓ Using pokt-network hermes v1.13.4" && \
        wget https://github.com/pokt-network/hermes/releases/download/v1.13.4/hermes-v1.13.4-${HERMES_ARCH}-unknown-linux-gnu.tar.gz && \
        tar -C /root/.hermes/bin/ -vxf hermes-v1.13.4-${HERMES_ARCH}-unknown-linux-gnu.tar.gz; \
    else \
        echo "⚠ Falling back to informalsystems hermes v1.13.1" && \
        wget https://github.com/informalsystems/hermes/releases/download/v1.13.1/hermes-v1.13.1-${HERMES_ARCH}-unknown-linux-gnu.tar.gz && \
        tar -C /root/.hermes/bin/ -vxf hermes-v1.13.1-${HERMES_ARCH}-unknown-linux-gnu.tar.gz; \
    fi && \
    # Clean up archives and verify installation
    rm -f *.tar.gz && \
    /root/.hermes/bin/hermes version

ENTRYPOINT ["/root/.hermes/bin/hermes"]
CMD ["start"]
