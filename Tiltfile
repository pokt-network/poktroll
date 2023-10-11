load('ext://restart_process', 'docker_build_with_restart')

# A list of directories where changes trigger a hot-reload of the sequencer
hot_reload_dirs = ['app', 'cmd', 'tools', 'x']

# Create localnet config file from defaults, and if default configuration doesn't exist in it - populate with default values
localnet_config_path = "localnet_config.yaml"
localnet_config_defaults = {
    "sequencers": {"count": 1},
    "relayers": {"count": 1},
    "gateways": {"count": 0},
    "helm_chart_local_repo": {"enabled": False, "path": "../helm-charts"},
}
localnet_config_file = read_yaml(localnet_config_path, default=localnet_config_defaults)
localnet_config = {}
localnet_config.update(localnet_config_defaults)
localnet_config.update(localnet_config_file)
if (localnet_config_file != localnet_config) or (
    not os.path.exists(localnet_config_path)
):
    print("Updating " + localnet_config_path + " with defaults")
    local("cat - > " + localnet_config_path, stdin=encode_yaml(localnet_config))

# Import files into Kubernetes ConfigMap
def read_files_from_directory(directory):
    files = listdir(directory)
    config_map_data = {}
    for filepath in files:
        content = str(read_file(filepath)).strip()
        filename = os.path.basename(filepath)
        config_map_data[filename] = content
    return config_map_data

def generate_config_map_yaml(name, data):
    config_map_object = {
        "apiVersion": "v1",
        "kind": "ConfigMap",
        "metadata": {"name": name},
        "data": data
    }
    return encode_yaml(config_map_object)

# Import keyring/keybase files into Kubernetes ConfigMap
k8s_yaml(generate_config_map_yaml("pocketd-keys", read_files_from_directory("localnet/pocketd/keyring-test/"))) # pocketd/keys
# Import configuration files into Kubernetes ConfigMap
k8s_yaml(generate_config_map_yaml("pocketd-configs", read_files_from_directory("localnet/pocketd/config/"))) # pocketd/configs

# Hot reload protobuf changes
local_resource('hot-reload: generate protobufs', 'ignite generate proto-go -y', deps=['proto'], labels=["hot-reloading"])
# Hot reload the pocketd binary used by the k8s cluster
local_resource('hot-reload: pocketd', 'GOOS=linux ignite chain build --skip-proto --output=./bin --debug -v', deps=hot_reload_dirs, labels=["hot-reloading"], resource_deps=['hot-reload: generate protobufs'])
# Hot reload the local pocketd binary used by the CLI
local_resource('hot-reload: pocketd - local cli', 'ignite chain build --skip-proto --debug -v -o $(go env GOPATH)/bin', deps=hot_reload_dirs, labels=["hot-reloading"], resource_deps=['hot-reload: generate protobufs'])

# Build an image with a pocketd binary
docker_build_with_restart(
    "pocketd",
    '.',
    dockerfile_contents="""FROM golang:1.20.8
RUN apt-get -q update && apt-get install -qyy curl jq
RUN go install github.com/go-delve/delve/cmd/dlv@latest
COPY bin/pocketd /usr/local/bin/pocketd
WORKDIR /
""",
    only=["./bin/pocketd"],
    entrypoint=[
        "/bin/sh", "/scripts/pocket.sh"
    ],
    live_update=[sync("bin/pocketd", "/usr/local/bin/pocketd")],
)

# Run pocketd, relayer, celestia and anvil nodes
k8s_yaml(['localnet/kubernetes/celestia-rollkit.yaml',
    'localnet/kubernetes/pocketd.yaml',
    'localnet/kubernetes/pocketd-relayer.yaml',
    'localnet/kubernetes/anvil.yaml'])

# Submit poktrolld sequencer manifests to k8s cluster


# Configure tilt resources (tilt labels and port forawards) for all of the nodes above
k8s_resource('celestia-rollkit', labels=["blockchains"], port_forwards=['26657', '26658', '26659'])
k8s_resource('pocketd', labels=["blockchains"], resource_deps=['celestia-rollkit'], port_forwards=['36657', '40004'])
k8s_resource('pocketd-relayer', labels=["blockchains"], resource_deps=['pocketd'], port_forwards=['8545', '8546', '40005'])
k8s_resource('anvil', labels=["blockchains"], port_forwards=['8547'])
