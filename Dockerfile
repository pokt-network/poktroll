FROM debian:latest

COPY release/poktrolld /usr/local/bin/poktrolld

RUN chmod +x /usr/local/bin/poktrolld

CMD ["/usr/local/bin/poktrolld"]