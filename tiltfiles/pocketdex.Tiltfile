load(
    './git.Tiltfile',
    'clone_repo',
    'repo_is_outdated',
    'repo_changes'
)

# pocketdex_disabled_resource creates a tilt resource that prints a message indicating
# that the indexer is disabled and how to enable it.
def pocketdex_disabled_resource(reason):
    local_resource(
        "⚠️ Indexer Disabled",
        "echo '{}'".format(reason),
        labels=["pocketdex"]
    )


# pocketdex_outdated_resource creates a tilt resource that prints a message indicating
# that the indexer is outdated and how many commits behind it is.
def pocketdex_outdated_resource(pocketdex_root_path):
    _, num_remote_changes = repo_changes(pocketdex_root_path)
    local_resource(
        "🔄 Updates Available",
        """
        echo 'Pocketdex main branch is outdated; {} commits behind. Please `git pull --ff-only` to update pocketdex.'
        """.format(num_remote_changes),
        labels=["pocketdex"]
    )


# load_pocketdex loads the pocketdex.tilt file from the pocketdex repo at pocketdex_root_path.
# It also checks if the pocketdex repo has updates and, if so, creates a resource which prints instructions to update.
def load_pocketdex(pocketdex_root_path, pocketdex_repo_branch, pocketdex_entrypoint_path, pocketdex_params):
    if repo_is_outdated(pocketdex_root_path, branch=pocketdex_repo_branch):
        pocketdex_outdated_resource(pocketdex_root_path)

    pocketdex_tilt_path = os.path.join(pocketdex_root_path, pocketdex_entrypoint_path)
    if not os.path.exists(pocketdex_tilt_path):
        pocketdex_disabled_resource("Pocketdex entrypoint path not found at: {}".format(pocketdex_tilt_path))
        return

    pocketdex_tilt = load_dynamic(pocketdex_tilt_path)

    # just in case you want to index something else that is not localnet from this repository
    # maybe for debug the a live network? - who knows...
    network = pocketdex_params.get("network")
    genesis_file_path = pocketdex_params.get("genesis_file_path")
    if network == 'localnet' and (not os.path.exists(genesis_file_path)):
        fail("genesis file not found at: {}".format(genesis_file_path))

    pgadmin = pocketdex_params.get("pgadmin", {})

    pocketdex_tilt["pocketdex"](
        network=network,
        base_path=pocketdex_root_path,
        genesis_file_path=genesis_file_path,
        indexer_params_overwrite=pocketdex_params.get('overwrite', {}),
        indexer_resource_deps=['validator'],
        pgadmin_enabled=pgadmin.get('enabled'),
        pgadmin_email=pgadmin.get('email'),
        pgadmin_password=pgadmin.get('password'),
        apps_labels=['pocketdex'],
        tools_labels=['pocketdex-db'],
        helm_repo_labels=['pocketdex-helm-repo'],
        only_db=False, # we do not want this on here
    )


# check_and_load_pocketdex checks if sibling pocketdex repo exists.
# If it does, load the pocketdex.tilt file from the sibling repo.
# Otherwise, check the `indexer.clone_if_not_present` flag in `localnet_config.yaml` and EITHER:
#   1. clone pocketdex to ../pocketdex
#   -- OR --
#   2. Prints a message if true or false
def check_and_load_pocketdex(indexer_config):
    if indexer_config.get("enabled", False):
        pocketdex_root_path = indexer_config.get("repo_path", "../pocketdex")
        pocketdex_repo_url = indexer_config.get("repo_url", "https://github.com/pokt-network/pocketdex")
        pocketdex_repo_branch =  indexer_config.get("repo_branch", "main")
        # REQUIREMENT: this file needs to have a pocketdex function
        pocketdex_entrypoint_path = indexer_config.get("entrypoint_path", "tilt/Tiltfile")
        pocketdex_params = indexer_config.get("params", {})

        if not os.path.exists(pocketdex_root_path):
            if indexer_config.get("clone_if_not_present", False):
                clone_repo(pocketdex_repo_url, pocketdex_root_path, pocketdex_repo_branch)
                load_pocketdex(pocketdex_root_path, pocketdex_repo_branch, pocketdex_entrypoint_path, pocketdex_params)
            else:
                pocketdex_disabled_resource("Pocketdex repo not found at {}. Set `clone_if_not_present` to `true` in `localnet_config.yaml`.".format(pocketdex_root_path))
        else:
            print("Using existing pocketdex repo")
            load_pocketdex(pocketdex_root_path, pocketdex_repo_branch, pocketdex_entrypoint_path, pocketdex_params)
    else:
        pocketdex_disabled_resource("Pocketdex indexer disabled. Set `indexer.enabled` to `true` in `localnet_config.yaml` to enable it.")
