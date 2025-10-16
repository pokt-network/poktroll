# ---- Target config (centralize it) ----
TARGET_GOOS = "linux"
TARGET_GOARCH = "amd64"

# Build configuration aligned with makefiles/ignite.mk
# For local/development builds with CGO
IGNITE_BUILD_TAGS_LOCAL = "--build.tags=ethereum_secp256k1"
# For cross-platform builds without CGO (uses Decred implementation)
IGNITE_BUILD_TAGS_CROSS = ""

# Unified build commands for different contexts
IGNITE_CMD_LOCAL = "ignite chain build %s --home $(HOME)/.pocket --skip-proto --debug -v" % IGNITE_BUILD_TAGS_LOCAL
IGNITE_CMD_CROSS = "ignite chain build %s --home $(HOME)/.pocket %s --skip-proto --debug -v" % IGNITE_BUILD_TAGS_CROSS

# Primary command for development (local + cross-compilation with CGO)
IGNITE_CMD = IGNITE_CMD_LOCAL

HOT_RELOAD_LABELS = ["hot-reloading"]
PROTO_RESOURCE = "hot-reload: generate protobufs"

CGO_CFLAGS = "-Wno-implicit-function-declaration -Wno-error=implicit-function-declaration"
IGNITE_CGO_CFLAGS = 'CGO_ENABLED=1 CGO_CFLAGS="%s"' % CGO_CFLAGS

# --- tiny helper ---
def _run(cmd):
    # local() returns a Blob in newer Tilt versions, string in older ones
    result = local(cmd, quiet=True)
    if hasattr(result, 'strip'):
        return result.strip()
    elif hasattr(result, 'read'):
        return result.read().strip()
    else:
        return str(result).strip()

def _host_os():
    # Darwin or Linux
    return _run("uname -s")

def _host_arch():
    # x86_64 / arm64 / aarch64 -> go arch
    m = _run("uname -m")
    return {"x86_64": "amd64", "arm64": "arm64", "aarch64": "arm64"}.get(m, m)

def _has_zig():
    return _run("command -v zig || true") != ""

def _zig_triple(goos, goarch):
    if goos == "linux" and goarch == "amd64":
        return "x86_64-linux-gnu"
    if goos == "linux" and goarch == "arm64":
        return "aarch64-linux-gnu"
    return ""

def build_env(target_goos="linux", target_goarch="amd64"):
    # Darwin / Linux
    host_os = _host_os()
    host_goos = "darwin" if host_os == "Darwin" else "linux" if host_os == "Linux" else host_os.lower()
    host_arch = _host_arch()

    need_cross = (target_goos != host_goos) or (target_goarch != host_arch)
    triple = _zig_triple(target_goos, target_goarch)

    if need_cross:
        # For cross-compilation, disable CGO and use pure Go implementation (Decred)
        # This avoids dynamic linking issues and works reliably in containers
        return {
            "GOOS": target_goos,
            "GOARCH": target_goarch,
            "CGO_ENABLED": "0",
        }
    else:
        # For native builds, use CGO with ethereum_secp256k1 for optimal performance
        return {
            "GOOS": target_goos,
            "GOARCH": target_goarch,
            "CGO_ENABLED": "1",
            "CGO_CFLAGS": CGO_CFLAGS,
        }

def build_cmd(target_goos="linux", target_goarch="amd64"):
    """Returns the appropriate ignite command with correct build tags for the target"""
    host_os = _host_os()
    host_goos = "darwin" if host_os == "Darwin" else "linux" if host_os == "Linux" else host_os.lower()
    host_arch = _host_arch()

    need_cross = (target_goos != host_goos) or (target_goarch != host_arch)

    if need_cross:
        # Cross-compilation: use no build tags (Decred implementation with CGO_ENABLED=0)
        return IGNITE_CMD_CROSS
    else:
        # Native compilation: use ethereum_secp256k1 tag with CGO
        return IGNITE_CMD_LOCAL