def repo_remote_name(repo_root):
    result = local(
        "git rev-parse --abbrev-ref @{u} 2>/dev/null || echo origin",
        dir=repo_root,
        quiet=True,
        echo_off=True,
    )
    return str(result).strip()

def fetch_repo_branch(repo_root, branch="main"):
    remote = repo_remote_name(repo_root)
    cmd = "git ls-remote --heads {} 2>/dev/null || true".format(remote)
    ls_remote = str(local(cmd, dir=repo_root, echo_off=True)).strip()

    if "refs/heads/{}".format(branch) in ls_remote:
        print("📥 Fetching {}/{}...".format(remote, branch))
        local("git fetch {} {}".format(remote, branch), dir=repo_root, echo_off=True)
    else:
        print("⚠️  Branch '{}' not found on remote '{}'. Skipping fetch.".format(branch, remote))

def repo_changes(repo_root, branch="main"):
    remote = repo_remote_name(repo_root)
    cmd = "git rev-list --left-right --count HEAD...{}/{} 2>/dev/null || echo '0 0'".format(remote, branch)
    out = str(local(cmd, dir=repo_root, echo_off=True)).strip()
    parts = out.split()
    if len(parts) == 2:
        return (int(parts[0]), int(parts[1]))
    return (0, 0)

def repo_is_outdated(repo_root, branch="main"):
    fetch_repo_branch(repo_root, branch=branch)
    _, remote_changes = repo_changes(repo_root, branch)
    return remote_changes != 0

def clone_repo(repo_url, local_path, branch="main"):
    print("📦 Cloning {} into {} on branch '{}'...".format(repo_url, local_path, branch))
    cmd = "git clone --branch {} {} {} 2>/dev/null || git clone {} {}".format(
        branch, repo_url, local_path, repo_url, local_path
    )
    local(cmd, echo_off=True)
