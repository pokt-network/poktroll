load('ext://uibutton', 'cmd_button', 'location')
load("./defaults.Tiltfile", "get_defaults")
load("./utils.Tiltfile", "deep_merge_dicts")

header = """
# ⚠️ Tilt auto-generated This file.
# Default values come from `tiltfiles/defaults.Tiltfile:get_defaults()`.
# It merges your localnet_config.yaml with any new defaults.
# Use the "💾 Apply Expanded Config" button in the Tilt UI to update your config.
"""

config_file_name = "localnet_config.yaml"
extended_config_file_name = "localnet_config_extended.yaml"


def get_file_content(config):
    return "{}\n{}".format(header.rstrip().lstrip(), str(encode_yaml(config)).lstrip())


def find_missing_keys(defaults, user_config, prefix=""):
    missing = []

    for key in defaults.keys():
        full_key = prefix + key
        if key not in user_config:
            missing.append(full_key)
        else:
            default_val = defaults[key]
            user_val = user_config[key]
            if type(default_val) == type({}) and type(user_val) == type({}):
                missing.extend(find_missing_keys(default_val, user_val, full_key + "."))
    return missing


def read_configs():
    defaults = get_defaults()
    file_exists = os.path.exists(config_file_name)

    if not file_exists:
        print("📄 Writing new config to {}".format(config_file_name))
        local(
            "cat - > {}".format(config_file_name),
            stdin=get_file_content(defaults)
        )
        return defaults  # nothing else needed

    user_cfg = read_yaml(config_file_name, default={})
    merged_cfg = deep_merge_dicts(defaults, user_cfg)

    missing = find_missing_keys(defaults, user_cfg)

    if missing:
        msg = "⚠️ Your config is missing keys: " + ", ".join(missing)
        print(msg)

        local(
            "cat - > {}".format(extended_config_file_name),
            stdin=get_file_content(merged_cfg)
        )

        # Only define the resource if it's actually needed
        local_resource(
            name="config-file",
            cmd="echo '{}'".format(msg),
            auto_init=False,
            allow_parallel=True,
            deps=[extended_config_file_name],
            labels=['config-file-outdated']
        )

        # Create a button to copy expanded config back to main config
        cmd_button(
            name='💾 Apply Expanded Config',
            argv=['sh', '-c', 'cp {} {} && rm {}'.format(extended_config_file_name, config_file_name, extended_config_file_name)],
            location=location.RESOURCE,
            resource='config-file',
            icon_name='upload_file',
            text='Apply the merged config to {}'.format(config_file_name),
            requires_confirmation=True
        )

        cmd_button(
            name='🔍 Show Diff',
            argv=['diff', '-u', config_file_name, extended_config_file_name],
            location=location.RESOURCE,
            resource='config-file',
            icon_name='visibility',
            text='Show diff between your config and the expanded version',
        )

    return merged_cfg
