load("ext://restart_process", "docker_build_with_restart")
load("ext://helm_resource", "helm_resource", "helm_repo")
load("ext://configmap", "configmap_create")

# A list of directories where changes trigger a hot-reload of the validator
hot_reload_dirs = ["app", "cmd", "tools", "x", "pkg"]

# Create a localnet config file from defaults, and if a default configuration doesn't exist, populate it with default values
localnet_config_path = "localnet_config.yaml"
localnet_config_defaults = {
    "validator": {"cleanupBeforeEachStart": True},
    "relayminers": {"count": 1},
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

# Import keyring/keybase files into Kubernetes ConfigMap
configmap_create(
    "poktrolld-keys", from_file=listdir("localnet/poktrolld/keyring-test/")
)
# Import configuration files into Kubernetes ConfigMap
configmap_create(
    "poktrolld-configs", from_file=listdir("localnet/poktrolld/config/"), watch=True
)
# TODO(@okdas): Import validator keys when we switch to `poktrolld` helm chart
# by uncommenting the following lines:
# load("ext://secret", "secret_create_generic")
# secret_create_generic(
#     "poktrolld-validator-keys",
#     from_file=[
#         "localnet/poktrolld/config/node_key.json",
#         "localnet/poktrolld/config/priv_validator_key.json",
#     ],
# )

# Hot reload protobuf changes
local_resource(
    "hot-reload: generate protobufs",
    "make proto_regen",
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
    dockerfile_contents="""FROM golang:1.21.6
RUN apt-get -q update && apt-get install -qyy curl jq less
RUN go install github.com/go-delve/delve/cmd/dlv@latest
COPY bin/poktrolld /usr/local/bin/poktrolld
WORKDIR /
""",
    only=["./bin/poktrolld"],
    entrypoint=["/bin/sh", "/scripts/pocket.sh"],
    live_update=[sync("bin/poktrolld", "/usr/local/bin/poktrolld")],
)

# Run data nodes & validators
k8s_yaml(
    ["localnet/kubernetes/anvil.yaml", "localnet/kubernetes/validator-volume.yaml"]
)

# Provision validator
helm_resource(
    "validator",
    chart_prefix + "poktroll-validator",
    flags=[
        "--values=./localnet/kubernetes/values-common.yaml",
        "--values=./localnet/kubernetes/values-validator.yaml",
        "--set=persistence.cleanupBeforeEachStart="
        + str(localnet_config["validator"]["cleanupBeforeEachStart"]),
    ],
    image_deps=["poktrolld"],
    image_keys=[("image.repository", "image.tag")],
)

# Provision RelayMiners
actor_number = 0
for x in range(localnet_config["relayminers"]["count"]):
    actor_number = actor_number + 1
    helm_resource(
        "relayminer" + str(actor_number),
        chart_prefix + "relayminer",
        flags=[
            "--values=./localnet/kubernetes/values-common.yaml",
            "--values=./localnet/kubernetes/values-relayminer-common.yaml",
            "--values=./localnet/kubernetes/values-relayminer-"+str(actor_number)+".yaml",
        ],
        image_deps=["poktrolld"],
        image_keys=[("image.repository", "image.tag")],
    )
    k8s_resource(
        "relayminer"  + str(actor_number),
        labels=["supplier_nodes"],
        resource_deps=["validator"],
        port_forwards=[
            str(8084+actor_number)+":8545", # relayminer1 - exposes 8545, relayminer2 exposes 8546, etc.
            str(40044+actor_number)+":40005", # DLV port. relayminer1 - exposes 40045, relayminer2 exposes 40046, etc.
            # Run `curl localhost:PORT` to see the current snapshot of relayminer metrics.
            str(9069+actor_number)+":9090", # Relayminer metrics port. relayminer1 - exposes 9070, relayminer2 exposes 9071, etc.
        ],
    )

# Provision AppGate Servers
actor_number = 0
for x in range(localnet_config["appgateservers"]["count"]):
    actor_number = actor_number + 1
    helm_resource(
        "appgateserver" + str(actor_number),
        chart_prefix + "appgate-server",
        flags=[
            "--values=./localnet/kubernetes/values-common.yaml",
            "--values=./localnet/kubernetes/values-appgateserver.yaml",
            "--set=config.signing_key=app" + str(actor_number),
        ],
        image_deps=["poktrolld"],
        image_keys=[("image.repository", "image.tag")],
    )
    k8s_resource(
        "appgateserver" + str(actor_number),
        labels=["supplier_nodes"],
        resource_deps=["validator"],
        port_forwards=[
            str(42068+actor_number)+":42069", # appgateserver1 - exposes 42069, appgateserver2 exposes 42070, etc.
            str(40054+actor_number)+":40006", # DLV port. appgateserver1 - exposes 40055, appgateserver2 exposes 40056, etc.
            # Run `curl localhost:PORT` to see the current snapshot of appgateserver metrics.
            str(9079+actor_number)+":9090", # appgateserver metrics port. appgateserver1 - exposes 9080, appgateserver2 exposes 9081, etc.
        ],
    )

k8s_resource(
    "validator",
    labels=["pocket_network"],
    port_forwards=["36657", "36658", "40004"],
)

k8s_resource("anvil", labels=["data_nodes"], port_forwards=["8547"])
