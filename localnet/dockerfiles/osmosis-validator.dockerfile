# Extract the osmosisd binary from the official image
FROM osmolabs/osmosis:29.0.1 AS source

# Final stage
FROM debian:bullseye-slim

RUN apt update
RUN apt install -y jq vim

COPY --from=source /bin/osmosisd /bin/osmosisd
RUN chmod +x /bin/osmosisd

ENV HOME=/osmosis
WORKDIR /osmosis

EXPOSE 1317 26656 26657
ENTRYPOINT ["/bin/osmosisd"]
