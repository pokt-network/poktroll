FROM golang:1.23.0

RUN apt-get update && apt-get install -y wget

# Install hermes IBC relayer CLI.
RUN mkdir -p /root/.hermes/bin
WORKDIR /usr/local
RUN wget https://github.com/informalsystems/hermes/releases/download/v1.10.3/hermes-v1.10.3-x86_64-unknown-linux-gnu.tar.gz
RUN tar -C /root/.hermes/bin/ -vxf /usr/local/hermes-v1.10.3-x86_64-unknown-linux-gnu.tar.gz

ENTRYPOINT ["/root/.hermes/bin/hermes"]
CMD ["start"]
