FROM ghcr.io/agoric/agoric-3-proposals:latest

# Update .bashrc
RUN grep -qF 'env_setup.sh' /root/.bashrc || echo "source /usr/src/upgrade-test-scripts/env_setup.sh" >> /root/.bashrc
RUN grep -qF 'printKeys' /root/.bashrc || echo "printKeys" >> /root/.bashrc


# TODO: Make agoric work in localnet.
# ## Install Go v1.23.0
# RUN wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
# RUN rm -rf /usr/local/go && tar -C /usr/local -xzf go*.tar.gz
# RUN echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc
# RUN echo 'export GOPATH=/usr/local/go' >> /root/.bashrc
#
# ## Install Debugging Tools
# RUN source /root/.bashrc && go install github.com/go-delve/delve/cmd/dlv@v1.23.0
# RUN apt update && apt install -y gdbserver gdb
#
# ## Clone and build Agoric SDK for local debugging source
# RUN git clone https://github.com/agoric/agoric-sdk.git --branch=v0.35.0-u19.2 /usr/src/agoric-sdk2
# WORKDIR /usr/src/agoric-sdk2/golang/cosmos
# RUN yarn add node-addon-api
# RUN . /root/.bashrc && go mod tidy
# RUN . /root/.bashrc && make all
# RUN cp ./build/agd /usr/local/bin/agd
#
# ## Reset the working directory
# WORKDIR /usr/src/upgrade-test-scripts
