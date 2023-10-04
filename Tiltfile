load('ext://restart_process', 'docker_build_with_restart')

# A list of directories where changes trigger a hot-reload of the sequencer
hot_reload_dirs = ['app', 'cmd', 'tools', 'x']

# Run celestia node
k8s_yaml('localnet/kubernetes/celestia-rollkit.yaml')

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

k8s_yaml(generate_config_map_yaml("poktrolld-keys", read_files_from_directory("localnet/poktrolld/keyring-test/"))) # poktrolld/keys
k8s_yaml(generate_config_map_yaml("poktrolld-configs", read_files_from_directory("localnet/poktrolld/config/"))) # poktrolld/configs

# Build sequencer
local_resource('hot-reload: generate protobufs', 'ignite generate proto-go -y', deps=['proto'], labels=["hot-reloading"])
local_resource('hot-reload: poktrolld', 'GOOS=linux ignite chain build --skip-proto --output=./bin --debug -v', deps=hot_reload_dirs, labels=["hot-reloading"], resource_deps=['hot-reload: generate protobufs'])
local_resource('hot-reload: poktrolld - local cli', 'ignite chain build --skip-proto --debug -v -o $(go env GOPATH)/bin', deps=hot_reload_dirs, labels=["hot-reloading"], resource_deps=['hot-reload: generate protobufs'])

# Build an image with a sequencer
docker_build_with_restart(
    "poktrolld",
    '.',
    dockerfile_contents="""FROM golang:1.20.8
RUN apt-get -q update && apt-get install -qyy curl jq
RUN go install github.com/go-delve/delve/cmd/dlv@latest
COPY bin/poktrolld /usr/local/bin/poktrolld
WORKDIR /
""",
    only=["./bin/poktrolld"],
    entrypoint=[
        "/bin/sh", "/scripts/poktroll.sh"
    ],
    live_update=[sync("bin/poktrolld", "/usr/local/bin/poktrolld")],
)

# Run poktrolld
k8s_yaml(['localnet/kubernetes/poktrolld.yaml', 'localnet/kubernetes/poktrolld-relayer.yaml', 'localnet/kubernetes/anvil.yaml'])

# Configure tilt resources for nodes
# TODO(@okdas): add port forwarding to be able to query the endpoints on localhost
k8s_resource('celestia-rollkit', labels=["blockchains"], port_forwards=['26657', '26658', '26659'])
k8s_resource('poktrolld', labels=["blockchains"], resource_deps=['celestia-rollkit'], port_forwards=['36657', '40004'])
k8s_resource('poktrolld-relayer', labels=["blockchains"], resource_deps=['poktrolld'], port_forwards=['8545', '8546', '40005'])
k8s_resource('anvil', labels=["blockchains"], port_forwards=['8547'])
