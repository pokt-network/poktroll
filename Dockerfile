FROM debian:latest

ARG TARGETARCH

COPY release_linux_$TARGETARCH/poktrolld /usr/local/bin/poktrolld

RUN chmod +x /usr/local/bin/poktrolld

CMD ["/usr/local/bin/poktrolld"]