FROM ghcr.io/agoric/agoric-3-proposals:latest

# Update .bashrc
RUN grep -qF 'env_setup.sh' /root/.bashrc || echo "source /usr/src/upgrade-test-scripts/env_setup.sh" >> /root/.bashrc
RUN grep -qF 'printKeys' /root/.bashrc || echo "printKeys" >> /root/.bashrc
