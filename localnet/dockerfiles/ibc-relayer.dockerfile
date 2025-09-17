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
    # TODO_TECHDEBT: use informalsystems hermes image once they cut a new release.
    # Ref: https://github.com/informalsystems/hermes/issues/4370#issuecomment-3246757714
    # wget https://github.com/informalsystems/hermes/releases/download/v1.13.1/hermes-v1.13.1-${HERMES_ARCH}-unknown-linux-gnu.tar.gz && \
    # tar -C /root/.hermes/bin/ -vxf hermes-v1.13.1-${HERMES_ARCH}-unknown-linux-gnu.tar.gz
    wget https://github.com/pokt-network/hermes/releases/download/v1.13.6/hermes-v1.13.6-${HERMES_ARCH}-unknown-linux-gnu.tar.gz && \
    tar -C /root/.hermes/bin/ -vxf hermes-v1.13.6-${HERMES_ARCH}-unknown-linux-gnu.tar.gz

ENTRYPOINT ["/root/.hermes/bin/hermes"]
CMD ["start"]
