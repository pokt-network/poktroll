# This Dockerfile is used to build container image for development purposes.
# It intentionally contains no security features, ships with code and troubleshooting tools.

FROM mcr.microsoft.com/devcontainers/go:1-1.23-bookworm

RUN curl https://get.ignite.com/cli | bash
RUN mv ignite /usr/local/bin/

# enable faster module downloading.
ENV GOPROXY https://proxy.golang.org
