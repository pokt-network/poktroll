FROM ubuntu:24.04

# Install system deps and Go
RUN apt-get update && apt-get install -y wget curl git build-essential ca-certificates

ENV PATH="/usr/local/go/bin:${PATH}"

# Install hermes
RUN mkdir -p /root/.hermes/bin
WORKDIR /usr/local
RUN wget https://github.com/informalsystems/hermes/releases/download/v1.13.1/hermes-v1.13.1-x86_64-unknown-linux-gnu.tar.gz
RUN tar -C /root/.hermes/bin/ -vxf hermes-v1.13.1-x86_64-unknown-linux-gnu.tar.gz

ENTRYPOINT ["/root/.hermes/bin/hermes"]
CMD ["start"]
