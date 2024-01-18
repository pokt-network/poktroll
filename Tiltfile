load("ext://restart_process", "docker_build_with_restart")
load("ext://helm_resource", "helm_resource", "helm_repo")

# A list of directories where changes trigger a hot-reload of the sequencer
hot_reload_dirs = ["app", "cmd", "tools", "x", "pkg"]

# Create a localnet config file from defaults, and if a default configuration doesn't exist, populate it with default values
localnet_config_path = "localnet_config.yaml"
localnet_config_defaults = {
    "sequencer": {"cleanupBeforeEachStart": True},
    "relayminers": {"count": 1},
    "gateways": {"count": 1},
    "appgateservers": {"count": 1},
    # By default, we use the `helm_repo` function below to point to the remote repository
    # but can update it to the locally cloned repo for testing & development
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

# Configure helm chart reference. If using a local repo, set the path to the local repo; otherwise, use our own helm repo.
helm_repo("pokt-network", "https://pokt-network.github.io/helm-charts/")
chart_prefix = "pokt-network/"
if localnet_config["helm_chart_local_repo"]["enabled"]:
    helm_chart_local_repo = localnet_config["helm_chart_local_repo"]["path"]
    hot_reload_dirs.append(helm_chart_local_repo)
    print("Using local helm chart repo " + helm_chart_local_repo)
    chart_prefix = helm_chart_local_repo + "/charts/"


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
        "data": data,
    }
    return encode_yaml(config_map_object)


# Import keyring/keybase files into Kubernetes ConfigMap
k8s_yaml(
    generate_config_map_yaml(
        "poktrolld-keys", read_files_from_directory("localnet/poktrolld/keyring-test/")
    )
)  # poktrolld/keys
# Import configuration files into Kubernetes ConfigMap
k8s_yaml(
    generate_config_map_yaml(
        "poktrolld-configs", read_files_from_directory("localnet/poktrolld/config/")
    )
)  # poktrolld/configs

# Hot reload protobuf changes
local_resource(
    "hot-reload: generate protobufs",
    "ignite generate proto-go -y",
    deps=["proto"],
    labels=["hot-reloading"],
)
# Hot reload the poktrolld binary used by the k8s cluster
local_resource(
    "hot-reload: poktrolld",
    "GOOS=linux ignite chain build --skip-proto --output=./bin --debug -v",
    deps=hot_reload_dirs,
    labels=["hot-reloading"],
    resource_deps=["hot-reload: generate protobufs"],
)
# Hot reload the local poktrolld binary used by the CLI
local_resource(
    "hot-reload: poktrolld - local cli",
    "ignite chain build --skip-proto --debug -v -o $(go env GOPATH)/bin",
    deps=hot_reload_dirs,
    labels=["hot-reloading"],
    resource_deps=["hot-reload: generate protobufs"],
)

# Build an image with a poktrolld binary
docker_build_with_restart(
    "poktrolld",
    ".",
    dockerfile_contents="""FROM golang:1.20.8
RUN apt-get -q update && apt-get install -qyy curl jq less
RUN go install github.com/go-delve/delve/cmd/dlv@latest
COPY bin/poktrolld /usr/local/bin/poktrolld
WORKDIR /
""",
    only=["./bin/poktrolld"],
    entrypoint=["/bin/sh", "/scripts/pocket.sh"],
    live_update=[sync("bin/poktrolld", "/usr/local/bin/poktrolld")],
)

# Run celestia and anvil nodes
k8s_yaml(
    ["localnet/kubernetes/celestia-rollkit.yaml", "localnet/kubernetes/anvil.yaml", "localnet/kubernetes/sequencer-volume.yaml"]
)

# Run pocket-specific nodes (sequencer, relayminers, etc...)
helm_resource(
    "sequencer",
    chart_prefix + "poktroll-sequencer",
    flags=[
        "--values=./localnet/kubernetes/values-common.yaml",
        "--values=./localnet/kubernetes/values-sequencer.yaml",
        "--set=persistence.cleanupBeforeEachStart=" + str(localnet_config["sequencer"]["cleanupBeforeEachStart"]),
        ],
    image_deps=["poktrolld"],
    image_keys=[("image.repository", "image.tag")],
)
helm_resource(
    "relayminers",
    chart_prefix + "relayminer",
    flags=[
        "--values=./localnet/kubernetes/values-common.yaml",
        "--values=./localnet/kubernetes/values-relayminer.yaml",
        "--set=replicaCount=" + str(localnet_config["relayminers"]["count"]),
    ],
    image_deps=["poktrolld"],
    image_keys=[("image.repository", "image.tag")],
)
helm_resource(
    "appgateservers",
    chart_prefix + "appgate-server",
    flags=[
        "--values=./localnet/kubernetes/values-common.yaml",
        "--values=./localnet/kubernetes/values-appgateserver.yaml",
        "--set=replicaCount=" + str(localnet_config["appgateservers"]["count"]),
    ],
    image_deps=["poktrolld"],
    image_keys=[("image.repository", "image.tag")],
)

# Configure tilt resources (tilt labels and port forwards) for all of the nodes above
k8s_resource(
    "celestia-rollkit",
    labels=["blockchains"],
    port_forwards=["26657", "26658", "26659"],
)
k8s_resource(
    "sequencer",
    labels=["blockchains"],
    resource_deps=["celestia-rollkit"],
    port_forwards=["36657", "36658", "40004"],
)
k8s_resource(
    "relayminers",
    labels=["blockchains"],
    resource_deps=["sequencer"],
    port_forwards=["8545", "40005"],
)
k8s_resource(
    "appgateservers",
    labels=["blockchains"],
    resource_deps=["sequencer"],
    port_forwards=["42069", "40006", "9093:9090"],
)
k8s_resource("anvil", labels=["blockchains"], port_forwards=["8547"])
