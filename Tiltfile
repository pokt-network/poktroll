load("ext://restart_process", "docker_build_with_restart")
load("ext://helm_resource", "helm_resource", "helm_repo")
load("ext://configmap", "configmap_create")
load("ext://secret", "secret_create_generic")
load("ext://deployment", "deployment_create")
load("ext://execute_in_pod", "execute_in_pod")
load("./tiltfiles/config.Tiltfile", "read_configs")
load("./tiltfiles/pocketdex.Tiltfile", "check_and_load_pocketdex")

# Avoid the header
analytics_settings(enable=False)

# A list of directories where changes trigger a hot-reload of the validator
hot_reload_dirs = ["app", "cmd", "tools", "x", "pkg", "telemetry"]

# TODO_IMPROVE: Non urgent requirement, but we need to find a way to ensure that the Tiltfile works (e.g. through config checks)
# so that if we merge something that passes E2E tests but was not manually validated by the developer, the developer
# environment is not broken for future engineers.

# Read configs
localnet_config = read_configs()

# Configure helm chart reference.
# If using a local repo, set the path to the local repo; otherwise, use our own helm repo.
helm_repo("pokt-network", "https://pokt-network.github.io/helm-charts/")
helm_repo("buildwithgrove", "https://buildwithgrove.github.io/helm-charts/")

# Configure POKT chart references
chart_prefix = "pokt-network/"
if localnet_config["helm_chart_local_repo"]["enabled"]:
    helm_chart_local_repo = localnet_config["helm_chart_local_repo"]["path"]
    chart_prefix = helm_chart_local_repo + "/charts/"
    # Build dependencies for the POKT chart
    # TODO_TECHDEBT(@okdas): Find a way to make this cleaner & performant w/ selective builds.
    local("cd " + chart_prefix + "pocketd && helm dependency update")
    local("cd " + chart_prefix + "pocket-validator && helm dependency update")
    local("cd " + chart_prefix + "relayminer && helm dependency update")
    hot_reload_dirs.append(helm_chart_local_repo)
    print("Using local helm chart repo " + helm_chart_local_repo)
    # TODO_IMPROVE: Use os.path.join to make this more OS-agnostic.


# Configure PATH references
grove_chart_prefix = "buildwithgrove/"
if localnet_config["grove_helm_chart_local_repo"]["enabled"]:
    grove_helm_chart_local_repo = localnet_config["grove_helm_chart_local_repo"]["path"]
    # Build dependencies for the PATH chart
    # TODO_TECHDEBT(@okdas): Find a way to make this cleaner & performant w/ selective builds.
    local("cd " + grove_helm_chart_local_repo + "/charts/path && helm dependency update")
    hot_reload_dirs.append(grove_helm_chart_local_repo)
    print("Using local grove helm chart repo " + grove_helm_chart_local_repo)
    # TODO_IMPROVE: Use os.path.join to make this more OS-agnostic.
    grove_chart_prefix = grove_helm_chart_local_repo + "/charts/"

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
configmap_create("pocketd-keys", from_file=listdir("localnet/pocketd/keyring-test/"))

# Import keyring/keybase files into Kubernetes Secret
secret_create_generic("pocketd-keys", from_file=listdir("localnet/pocketd/keyring-test/"))

# Import validator keys for the pocketd helm chart to consume
secret_create_generic(
    "pocketd-validator-keys",
    from_file=[
        "localnet/pocketd/config/node_key.json",
        "localnet/pocketd/config/priv_validator_key.json",
    ],
)

# Import configuration files into Kubernetes ConfigMap
configmap_create("pocketd-configs", from_file=listdir("localnet/pocketd/config/"), watch=True)

if localnet_config["hot-reloading"]:
    # Hot reload protobuf changes
    local_resource(
        "hot-reload: generate protobufs",
        "make proto_regen",
        deps=["proto"],
        labels=["hot-reloading"],
    )
    # Hot reload the pocketd binary used by the k8s cluster
    local_resource(
        "hot-reload: pocketd",
        "GOOS=linux ignite chain build --skip-proto --output=./bin --debug -v",
        deps=hot_reload_dirs,
        labels=["hot-reloading"],
        resource_deps=["hot-reload: generate protobufs"],
    )
    # Hot reload the local pocketd binary used by the CLI
    local_resource(
        "hot-reload: pocketd - local cli",
        "ignite chain build --skip-proto --debug -v -o $(go env GOPATH)/bin",
        deps=hot_reload_dirs,
        labels=["hot-reloading"],
        resource_deps=["hot-reload: generate protobufs"],
    )

# Build an image with a pocketd binary
docker_build_with_restart(
    "pocketd",
    ".",
    dockerfile_contents="""FROM golang:1.24.3
RUN apt-get -q update && apt-get install -qyy curl jq less
RUN go install github.com/go-delve/delve/cmd/dlv@latest
COPY bin/pocketd /usr/local/bin/pocketd
WORKDIR /
""",
    only=["./bin/pocketd"],
    entrypoint=["pocketd"],
    live_update=[sync("bin/pocketd", "/usr/local/bin/pocketd")],
)

# Run data nodes & validators
k8s_yaml(
    ["localnet/kubernetes/anvil.yaml", "localnet/kubernetes/rest.yaml", "localnet/kubernetes/validator-volume.yaml"]
)

# Provision validator
helm_resource(
    "validator",
    chart_prefix + "pocket-validator",
    flags=[
        "--values=./localnet/kubernetes/values-common.yaml",
        "--values=./localnet/kubernetes/values-validator.yaml",
        "--set=persistence.cleanupBeforeEachStart=" + str(localnet_config["validator"]["cleanupBeforeEachStart"]),
        "--set=logs.level=" + str(localnet_config["validator"]["logs"]["level"]),
        "--set=logs.format=" + str(localnet_config["validator"]["logs"]["format"]),
        "--set=serviceMonitor.enabled=" + str(localnet_config["observability"]["enabled"]),
        "--set=development.delve.enabled=" + str(localnet_config["validator"]["delve"]["enabled"]),
        "--set=image.repository=pocketd",
    ],
    image_deps=["pocketd"],
    image_keys=[("image.repository", "image.tag")],
)

# Provision RelayMiners
actor_number = 0
for x in range(localnet_config["relayminers"]["count"]):
    actor_number = actor_number + 1

    flags=[
            "--values=./localnet/kubernetes/values-common.yaml",
            "--values=./localnet/kubernetes/values-relayminer-common.yaml",
            "--values=./localnet/kubernetes/values-relayminer-" + str(actor_number) + ".yaml",
            "--set=metrics.serviceMonitor.enabled=" + str(localnet_config["observability"]["enabled"]),
            "--set=development.delve.enabled=" + str(localnet_config["relayminers"]["delve"]["enabled"]),
            "--set=logLevel=" + str(localnet_config["relayminers"]["logs"]["level"]),
            # Default queryCaching to false if not set in localnet_config
            "--set=queryCaching=" + str(localnet_config["relayminers"].get("queryCaching", False)),
            "--set=image.repository=pocketd",
    ]

    #############
    # NOTE: To provide a proper configuration for the relayminer, we dynamically
    # define the supplier configuration overrides for the relayminer helm chart
    # so that every service enabled in the localnet configuration (ollama, rest)
    # file are also declared in the relayminer.config.suppliers list.
    #############

    supplier_number = 0

    flags.append("--set=config.suppliers["+str(supplier_number)+"].service_id=anvil")
    flags.append("--set=config.suppliers["+str(supplier_number)+"].listen_url=http://0.0.0.0:8545")
    flags.append("--set=config.suppliers["+str(supplier_number)+"].service_config.backend_url=http://anvil:8547/")
    flags.append("--set=config.suppliers["+str(supplier_number)+"].rpc_type_service_configs.json_rpc.backend_url=http://anvil:8547/")
    supplier_number = supplier_number + 1

    flags.append("--set=config.suppliers["+str(supplier_number)+"].service_id=anvilws")
    flags.append("--set=config.suppliers["+str(supplier_number)+"].listen_url=http://0.0.0.0:8545")
    flags.append("--set=config.suppliers["+str(supplier_number)+"].service_config.backend_url=http://anvil:8547/")
    flags.append("--set=config.suppliers["+str(supplier_number)+"].rpc_type_service_configs.websocket.backend_url=ws://anvil:8547/")
    supplier_number = supplier_number + 1

    if localnet_config["rest"]["enabled"]:
       flags.append("--set=config.suppliers["+str(supplier_number)+"].service_id=rest")
       flags.append("--set=config.suppliers["+str(supplier_number)+"].listen_url=http://0.0.0.0:8545")
       flags.append("--set=config.suppliers["+str(supplier_number)+"].service_config.backend_url=http://rest:10000/")
       supplier_number = supplier_number + 1

    if localnet_config["ollama"]["enabled"]:
       flags.append("--set=config.suppliers["+str(supplier_number)+"].service_id=ollama")
       flags.append("--set=config.suppliers["+str(supplier_number)+"].listen_url=http://0.0.0.0:8545")
       flags.append("--set=config.suppliers["+str(supplier_number)+"].service_config.backend_url=http://ollama:11434/")
       supplier_number = supplier_number + 1

    helm_resource(
        "relayminer" + str(actor_number),
        chart_prefix + "relayminer",
        flags=flags,
        image_deps=["pocketd"],
        image_keys=[("image.repository", "image.tag")],
    )

    k8s_resource(
        "relayminer" + str(actor_number),
        labels=["suppliers"],
        resource_deps=["validator", "anvil"],
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
            str(7000 + actor_number) + ":8081", # Relayminer ping port. relayminer1 - exposes 7001, relayminer2 exposes 7002, etc.
        ],
    )

if localnet_config["path_local_repo"]["enabled"]:
    docker_build("path-local", path_local_repo)

# TODO_TECHDEBT(@okdas): Find and replace all `appgateserver` in ./localnet/grafana-dashboards`
# with PATH metrics (see the .json files)
# Ref: https://github.com/buildwithgrove/path/pull/72

# Provision PATH Gateway(s)
actor_number = 0
# Loop to configure and apply multiple PATH gateway deployments
for x in range(localnet_config["path_gateways"]["count"]):
    actor_number += 1

    resource_flags = [
        # PATH global values
        "--set=fullnameOverride=path" + str(actor_number),
        "--set=nameOverride=path" + str(actor_number),
        "--set=global.serviceAccount.name=path" + str(actor_number),
        "--set=metrics.serviceMonitor.enabled=" + str(localnet_config["observability"]["enabled"]),
        # PATH config values
        "--set=config.fromConfigMap.enabled=true",
        "--set=config.fromConfigMap.name=path-config-" + str(actor_number),
        "--set=config.fromConfigMap.key=.config.yaml",
        # GUARD values
        "--set=guard.global.namespace=default", # Ensure GUARD runs in the default namespace
        "--set=guard.global.serviceName=path" + str(actor_number) + "-http", # Override the default service name
        "--set=guard.services[0].serviceId=anvil", # Ensure HTTPRoute resources are created for Anvil
        "--set=observability.enabled=false",
        # TODO_TECHDEBT(@okdas): Remove the need for an override that uses a pre-released version of RLS.
        # See 'guard-overrides.yaml' for more details and TODOs.
        "--values=./localnet/kubernetes/guard-overrides.yaml",
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
        update_dependencies=localnet_config["path_local_repo"]["enabled"],
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
        # TODO_TECHDEBT(@okdas): Enable using port `3070` for consistency with path_localnet.
        # Changing this port will make E2E start failing.
        # See the `envoy proxy` k8s resource below for in progress work.
        port_forwards=[
                # See PATH for the default port used by the gateway. As of PR #1026, it is :3069.
                # https://github.com/buildwithgrove/path/blob/main/config/router.go
                str(3068 + actor_number) + ":3069"
        ],
        extra_pod_selectors=[
            {"app.kubernetes.io/instance": "path" + str(actor_number)},
            {"app.kubernetes.io/name": "path" + str(actor_number)},
        ],
        discovery_strategy="selectors-only", # Extra pod selectors didn't work without this
    )

    # DO NOT DELETE, this is an example of how we'll turn on envoy proxy in the future
    # Envoy Proxy / Gateway. Endpoint that requires authorization header (unlike 3069 - accesses path directly)
    # k8s_resource(
    #     "path" + str(actor_number),
    #     new_name="path-envoy-proxy",
    #     extra_pod_selectors=[{"gateway.envoyproxy.io/owning-gateway-name": "guard-envoy-gateway", "app.kubernetes.io/component": "proxy"}],
    #     port_forwards=["3070:3070"],
    # )


# Provision Validators
k8s_resource(
    "validator",
    labels=["pocket_network"],
    port_forwards=[
        "26657",  # CometBFT JSON-RPC
        "9090",  # the gRPC server address
        "40004",  # use with `dlv` when it's turned on in `localnet_config.yaml`
        "1317", # CosmosSDK REST API
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

# Check if sibling pocketdex repo exists.
# If it does, load the pocketdex.tilt file from the sibling repo.
# Otherwise, check the `indexer.clone_if_not_present` flag in `localnet_config.yaml` and EITHER:
#   1. clone pocketdex to ../pocketdex
#   -- OR --
#   2. Prints a message if true or false
check_and_load_pocketdex(localnet_config["indexer"])


### Pocketd Faucet
if localnet_config["faucet"]["enabled"]:
    helm_resource(
        "pocketd-faucet",
        chart_prefix + "pocketd-faucet",
        flags=[
            "--values=./localnet/kubernetes/values-pocketd-faucet.yaml",
            "--set=image.repository=pocketd",
        ],
        image_deps=["pocketd"],
        image_keys=[("image.repository", "image.tag")],
        resource_deps=["validator"],
    )

    k8s_resource(
        "pocketd-faucet",
        labels=["faucet"],
        resource_deps=["validator"],
        port_forwards=[
            "8080:8080",
        ],
    )
