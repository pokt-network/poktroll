load("ext://restart_process", "docker_build_with_restart")
load("ext://helm_resource", "helm_resource", "helm_repo")
load("ext://configmap", "configmap_create")
load("ext://secret", "secret_create_generic")
load("ext://deployment", "deployment_create")
load("ext://execute_in_pod", "execute_in_pod")

# A list of directories where changes trigger a hot-reload of the validator
hot_reload_dirs = ["app", "cmd", "tools", "x", "pkg", "telemetry"]


def merge_dicts(base, updates):
    for k, v in updates.items():
        if k in base and type(base[k]) == "dict" and type(v) == "dict":
            # Assume nested dict and merge
            for vk, vv in v.items():
                base[k][vk] = vv
        else:
            # Replace or set the value
            base[k] = v


# Create a localnet config file from defaults, and if a default configuration doesn't exist, populate it with default values
# TODO_TEST: Non urgent requirement, but we need to find a way to ensure that the Tiltfile works (e.g. through config checks)
#            so that if we merge something that passes E2E tests but was not manually validated by the developer, the developer
#            environment is not broken for future engineers.
localnet_config_path = "localnet_config.yaml"
localnet_config_defaults = {
    "hot-reloading": True,
    "validator": {
        "cleanupBeforeEachStart": True,
        "logs": {
            "level": "info",
            "format": "json",
        },
        "delve": {"enabled": False},
    },
    "observability": {
        "enabled": True,
        "grafana": {"defaultDashboardsEnabled": False},
    },
    "relayminers": {
        "count": 1,
        "delve": {"enabled": False},
        "logs": {
            "level": "debug",
        },
    },
    "ollama": {
        "enabled": False,
        "model": "qwen:0.5b",
    },
    "rest": {
        "enabled": True,
    },
    "path_gateways": {
        "count": 1,
    },

    #############
    # NOTE: git submodule usage was explicitly avoided for the repositories below
    # to reduce environment complexity.
    #############

    # By default, we use the `helm_repo` function below to point to the remote repository
    # but can update it to the locally cloned repo for testing & development
    "helm_chart_local_repo": {
        "enabled": False,
        "path": os.path.join("..", "helm-charts")
    },

    # By default, we use a pre-built PATH image, but can update it to use a local
    # repo instead.
    "path_local_repo": {
        "enabled": False,
        "path": "../path"
    },

    "indexer": {
        "repo_path": os.path.join("..", "pocketdex"),
        "enabled": True,
        "clone_if_not_present": False,
    },
}
localnet_config_file = read_yaml(localnet_config_path, default=localnet_config_defaults)
# Initial empty config
localnet_config = {}
# Load the existing config file, if it exists, or use an empty dict as fallback
localnet_config_file = read_yaml(localnet_config_path, default={})
# Merge defaults into the localnet_config first
merge_dicts(localnet_config, localnet_config_defaults)
# Then merge file contents over defaults
merge_dicts(localnet_config, localnet_config_file)
# Check if there are differences or if the file doesn't exist
if (localnet_config_file != localnet_config) or (not os.path.exists(localnet_config_path)):
    print("Updating " + localnet_config_path + " with defaults")
    local("cat - > " + localnet_config_path, stdin=encode_yaml(localnet_config))

# Configure helm chart reference.
# If using a local repo, set the path to the local repo; otherwise, use our own helm repo.
helm_repo("pokt-network", "https://pokt-network.github.io/helm-charts/")
helm_repo("buildwithgrove", "https://buildwithgrove.github.io/helm-charts/")

# Configure POKT chart references
chart_prefix = "pokt-network/"
if localnet_config["helm_chart_local_repo"]["enabled"]:
    helm_chart_local_repo = localnet_config["helm_chart_local_repo"]["path"]
    hot_reload_dirs.append(helm_chart_local_repo)
    print("Using local helm chart repo " + helm_chart_local_repo)
    # TODO_IMPROVE: Use os.path.join to make this more OS-agnostic.
    chart_prefix = helm_chart_local_repo + "/charts/"

# Configure PATH references
grove_chart_prefix = "buildwithgrove/"
# If using a local repo, set the path to the local repo; otherwise, use our own helm repo.
path_local_repo = ""
if localnet_config["path_local_repo"]["enabled"]:
    path_local_repo = localnet_config["path_local_repo"]["path"]
    hot_reload_dirs.append(path_local_repo)
    print("Using local PATH repo " + path_local_repo)

# Observability
print("Observability enabled: " + str(localnet_config["observability"]["enabled"]))
if localnet_config["observability"]["enabled"]:
    helm_repo("prometheus-community", "https://prometheus-community.github.io/helm-charts")
    helm_repo("grafana-helm-repo", "https://grafana.github.io/helm-charts")

    # Timeout is increased to 120 seconds (default is 30) because a slow internet connection
    # could timeout pulling the image.
    # container images.
    update_settings(k8s_upsert_timeout_secs=120)

    helm_resource(
        "observability",
        "prometheus-community/kube-prometheus-stack",
        flags=[
            "--values=./localnet/kubernetes/observability-prometheus-stack.yaml",
            "--set=grafana.defaultDashboardsEnabled="
            + str(localnet_config["observability"]["grafana"]["defaultDashboardsEnabled"]),
        ],
        resource_deps=["prometheus-community"],
    )

    helm_resource(
        "loki",
        "grafana-helm-repo/loki-stack",
        flags=[
            "--values=./localnet/kubernetes/observability-loki-stack.yaml",
        ],
        labels=["monitoring"],
        resource_deps=["grafana-helm-repo"],
    )

    # TODO_BUG(@okdas): There is an occasional issue where grafana hits a "Database locked"
    # error when updating grafana. There is likely a weird race condition happening
    # that requires a restart of LocalNet. Look into it.
    k8s_resource(
        new_name="grafana",
        workload="observability",
        extra_pod_selectors=[{"app.kubernetes.io/name": "grafana"}],
        port_forwards=["3003:3000"],
        labels=["monitoring"],
        links=[
            link("localhost:3003", "Grafana"),
        ],
        pod_readiness="wait",
        discovery_strategy="selectors-only",
    )

    # Import our custom grafana dashboards into Kubernetes ConfigMap
    configmap_create("protocol-dashboards", from_file=listdir("localnet/grafana-dashboards/"))

    # Grafana discovers dashboards to "import" via a label
    local_resource(
        "protocol-dashboards-label",
        "kubectl label configmap protocol-dashboards grafana_dashboard=1 --overwrite",
        resource_deps=["protocol-dashboards"],
    )

# Import keyring/keybase files into Kubernetes ConfigMap
configmap_create("poktrolld-keys", from_file=listdir("localnet/poktrolld/keyring-test/"))

# Import keyring/keybase files into Kubernetes Secret
secret_create_generic("poktrolld-keys", from_file=listdir("localnet/poktrolld/keyring-test/"))

# Import validator keys for the poktrolld helm chart to consume
secret_create_generic(
    "poktrolld-validator-keys",
    from_file=[
        "localnet/poktrolld/config/node_key.json",
        "localnet/poktrolld/config/priv_validator_key.json",
    ],
)

# Import configuration files into Kubernetes ConfigMap
configmap_create("poktrolld-configs", from_file=listdir("localnet/poktrolld/config/"), watch=True)

if localnet_config["hot-reloading"]:
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
    dockerfile_contents="""FROM golang:1.23.0
RUN apt-get -q update && apt-get install -qyy curl jq less
RUN go install github.com/go-delve/delve/cmd/dlv@latest
COPY bin/poktrolld /usr/local/bin/poktrolld
WORKDIR /
""",
    only=["./bin/poktrolld"],
    entrypoint=[
        "poktrolld",
    ],
    live_update=[sync("bin/poktrolld", "/usr/local/bin/poktrolld")],
)

# Run data nodes & validators
k8s_yaml(
    ["localnet/kubernetes/anvil.yaml", "localnet/kubernetes/rest.yaml", "localnet/kubernetes/validator-volume.yaml"]
)

# Provision validator
helm_resource(
    "validator",
    chart_prefix + "poktroll-validator",
    flags=[
        "--values=./localnet/kubernetes/values-common.yaml",
        "--values=./localnet/kubernetes/values-validator.yaml",
        "--set=persistence.cleanupBeforeEachStart=" + str(localnet_config["validator"]["cleanupBeforeEachStart"]),
        "--set=logs.level=" + str(localnet_config["validator"]["logs"]["level"]),
        "--set=logs.format=" + str(localnet_config["validator"]["logs"]["format"]),
        "--set=serviceMonitor.enabled=" + str(localnet_config["observability"]["enabled"]),
        "--set=development.delve.enabled=" + str(localnet_config["validator"]["delve"]["enabled"]),
        "--set=image.repository=poktrolld",
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
            "--values=./localnet/kubernetes/values-relayminer-" + str(actor_number) + ".yaml",
            "--set=metrics.serviceMonitor.enabled=" + str(localnet_config["observability"]["enabled"]),
            "--set=development.delve.enabled=" + str(localnet_config["relayminers"]["delve"]["enabled"]),
            "--set=logLevel=" + str(localnet_config["relayminers"]["logs"]["level"]),
            "--set=image.repository=poktrolld",
        ],
        image_deps=["poktrolld"],
        image_keys=[("image.repository", "image.tag")],
    )
    k8s_resource(
        "relayminer" + str(actor_number),
        labels=["suppliers"],
        resource_deps=["validator"],
        links=[
            link(
                "http://localhost:3003/d/relayminer/relayminer?orgId=1&var-relayminer=relayminer" + str(actor_number),
                "Grafana dashboard",
            ),
        ],
        port_forwards=[
            str(8084 + actor_number) + ":8545",  # relayminer1 - exposes 8545, relayminer2 exposes 8546, etc.
            str(40044 + actor_number)
            + ":40004",  # DLV port. relayminer1 - exposes 40045, relayminer2 exposes 40046, etc.
            # Run `curl localhost:PORT` to see the current snapshot of relayminer metrics.
            str(9069 + actor_number)
            + ":9090",  # Relayminer metrics port. relayminer1 - exposes 9070, relayminer2 exposes 9071, etc.
            # Use with pprof like this: `go tool pprof -http=:3333 http://localhost:6070/debug/pprof/goroutine`
            str(6069 + actor_number)
            + ":6060",  # Relayminer pprof port. relayminer1 - exposes 6070, relayminer2 exposes 6071, etc.
        ],
    )

if localnet_config["path_local_repo"]["enabled"]:
    docker_build("path-local", path_local_repo)

# TODO_MAINNET(@okdas): Find and replace all `appgateserver` in ./localnet/grafana-dashboards`
# with PATH metrics (see the .json files)
# Ref: https://github.com/buildwithgrove/path/pull/72

# Provision PATH Gateway(s)
actor_number = 0
# Loop to configure and apply multiple PATH gateway deployments
for x in range(localnet_config["path_gateways"]["count"]):
    actor_number += 1

    resource_flags = [
        "--values=./localnet/kubernetes/values-common.yaml",
        "--set=metrics.serviceMonitor.enabled=" + str(localnet_config["observability"]["enabled"]),
        "--set=path.mountConfigMaps[0].name=path-config-" + str(actor_number),
        "--set=path.mountConfigMaps[0].mountPath=/app/config/",
        "--set=fullnameOverride=path" + str(actor_number),
        "--set=nameOverride=path" + str(actor_number),
        "--set=global.serviceAccount.name=path" + str(actor_number),
    ]

    if localnet_config["path_local_repo"]["enabled"]:
        path_image_deps = ["path-local"]
        path_image_keys = [("image.repository", "image.tag")]
        path_deps=["path-local"]
        resource_flags.append("--set=global.imagePullPolicy=Never")
    else:
        path_image_deps = []
        path_image_keys = []
        path_deps=[]

    configmap_create(
        "path-config-" + str(actor_number),
        from_file=".config.yaml=./localnet/kubernetes/config-path-" + str(actor_number) + ".yaml"
    )

    helm_resource(
        "path" + str(actor_number),
        grove_chart_prefix + "path",
        flags=resource_flags,
        image_deps=path_image_deps,
        image_keys=path_image_keys,
    )

    # Apply the deployment to Kubernetes using Tilt
    k8s_resource(
        "path" + str(actor_number),
        labels=["gateways"],
        resource_deps=path_deps,
        # TODO_IMPROVE(@okdas): Update this once PATH has grafana dashboards
        # links=[
        #     link(
        #         "http://localhost:3003/d/path/protocol-path?orgId=1&refresh=5s&var-path=gateway"
        #         + str(actor_number),
        #         "Grafana dashboard",
        #     ),
        # ],
        # TODO_IMPROVE(@okdas): Add port forwards to grafana, pprof, like the other resources
        port_forwards=[
                # See PATH for the default port used by the gateway. As of PR #1026, it is :3069.
                # https://github.com/buildwithgrove/path/blob/main/config/router.go
                str(3068 + actor_number) + ":3069"
        ],
    )


# Provision Validators
k8s_resource(
    "validator",
    labels=["pocket_network"],
    port_forwards=[
        "26657",  # RPC
        "9090",  # the gRPC server address
        "40004",  # use with `dlv` when it's turned on in `localnet_config.yaml`
        # Use with pprof like this: `go tool pprof -http=:3333 http://localhost:6050/debug/pprof/goroutine`
        "6050:6060",
    ],
    links=[
        link(
            "http://localhost:3003/d/cosmoscometbft/protocol-cometbft-dashboard?orgId=1&from=now-1h&to=now",
            "Validator dashboard",
        ),
    ],
)

# Provision anvil (test Ethereum) service nodes
k8s_resource("anvil", labels=["data_nodes"], port_forwards=["8547"])

# Provision ollama (LLM) service nodes
if localnet_config["ollama"]["enabled"]:
    print("Ollama enabled: " + str(localnet_config["ollama"]["enabled"]))

    deployment_create(
        "ollama",
        image="ollama/ollama",
        command=["ollama", "serve"],
        ports="11434",
    )

    k8s_resource("ollama", labels=["data_nodes"], port_forwards=["11434"])

    local_resource(
        name="ollama-pull-model",
        cmd=execute_in_pod("ollama", "ollama pull " + localnet_config["ollama"]["model"]),
        resource_deps=["ollama"],
    )

# Provision RESTful (not JSON-RPC) test service nodes
if localnet_config["rest"]["enabled"]:
    print("REST enabled: " + str(localnet_config["rest"]["enabled"]))
    deployment_create("rest", image="davarski/go-rest-api-demo")
    k8s_resource("rest", labels=["data_nodes"], port_forwards=["10000"])

### Pocketdex Shannon Indexer
load("./tiltfiles/pocketdex.tilt", "check_and_load_pocketdex")

# Check if sibling pocketdex repo exists.
# If it does, load the pocketdex.tilt file from the sibling repo.
# Otherwise, check the `indexer.clone_if_not_present` flag in `localnet_config.yaml` and EITHER:
#   1. clone pocketdex to ../pocketdex
#   -- OR --
#   2. Prints a message if true or false
check_and_load_pocketdex(localnet_config["indexer"])
