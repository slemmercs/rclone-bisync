"""
Compiler: CLI Arguments to Rclone RC Parameters & Connection Strings
"""

import os
import logging
from typing import List, Dict, Any, Tuple

# Mapping of CLI arguments to Rclone Remote Control (RC) API parameters.
CLI_TO_RC_MAP = {
    # Logging & console formatting
    "--use-json-log": ("config_bool", "LogJSON"),
    "--color": ("ignore", "color"),

    # Drive-specific options (must be passed as connection string parameters on path1)
    "--drive-export-formats": ("backend_str", "export_formats"),
    "--drive-import-formats": ("backend_str", "import_formats"),
    "--drive-skip-gdocs": ("backend_bool", "skip_gdocs"),
    "--drive-use-trash": ("backend_bool", "use_trash"),
    "--drive-acknowledge-abuse": ("backend_bool", "acknowledge_abuse"),
    "--drive-skip-shortcuts": ("backend_bool", "skip_shortcuts"),

    # Bisync parameters (top-level parameters to /sync/bisync)
    "--check-access": ("param_bool", "checkAccess"),
    "--check-first": ("param_bool", "checkFirst"),
    "--track-renames": ("param_bool", "trackRenames"),
    "--remove-empty-dirs": ("param_bool", "removeEmptyDirs"),
    "--resilient": ("param_bool", "resilient"),
    "--recover": ("param_bool", "recover"),
    "--compare": ("param", "compare"),

    # Conflict handling
    "--conflict-resolve": ("param", "conflictResolve"),
    "--conflict-loser": ("param", "conflictLoser"),

    # Global Engine Config (passed under _config)
    "--fast-list": ("config_bool", "UseListR"),
    "--buffer-size": ("config_str", "BufferSize"),
    "--retries": ("config_int", "Retries"),
    "--retries-sleep": ("config_str", "RetriesSleep"),
    "--low-level-retries": ("config_int", "LowLevelRetries"),
    "--tpslimit": ("config_float", "TPSLimit"),
    "--tpslimit-burst": ("config_int", "TPSLimitBurst"),
}


def compile_bisync_payload(
    args_list: List[str],
    local_path: str,
    remote_path: str,
    resync_attempt: bool = False,
    force_attempt: bool = False,
    dry_run_attempt: bool = False
) -> Dict[str, Any]:
    """
    Translates RCLONE_COMMON_ARGS into valid parameters, backend connection
    string options, and _config settings for the Rclone Remote Control (RC)
    API endpoint (/sync/bisync).
    """
    bisync_params = {
        "path1": remote_path,
        "path2": local_path,
        "resync": bool(resync_attempt),
        "force": bool(force_attempt),
        "dryRun": bool(dry_run_attempt),
    }
    engine_config: Dict[str, Any] = {}
    backend_options: List[str] = []

    i = 0
    while i < len(args_list):
        arg = args_list[i]
        if arg in CLI_TO_RC_MAP:
            action, target_key = CLI_TO_RC_MAP[arg]
            if action == "ignore":
                if i + 1 < len(args_list) and not args_list[i + 1].startswith("-"):
                    i += 1
            elif action == "param_bool":
                bisync_params[target_key] = True
            elif action == "config_bool":
                engine_config[target_key] = True
            elif action == "backend_bool":
                backend_options.append(f"{target_key}=true")
            elif action == "backend_str":
                i += 1
                if i < len(args_list):
                    val = args_list[i]
                    if "," in val or " " in val:
                        backend_options.append(f'{target_key}="{val}"')
                    else:
                        backend_options.append(f"{target_key}={val}")
            elif action == "param":
                i += 1
                if i < len(args_list):
                    bisync_params[target_key] = args_list[i]
            elif action == "config_str":
                i += 1
                if i < len(args_list):
                    engine_config[target_key] = args_list[i]
            elif action == "config_int":
                i += 1
                if i < len(args_list):
                    try:
                        engine_config[target_key] = int(args_list[i])
                    except ValueError:
                        engine_config[target_key] = args_list[i]
            elif action == "config_float":
                i += 1
                if i < len(args_list):
                    try:
                        engine_config[target_key] = float(args_list[i])
                    except ValueError:
                        engine_config[target_key] = args_list[i]
        else:
            if not arg.startswith("-"):
                logging.warning(f"Unexpected argument in RCLONE_COMMON_ARGS: {arg}")
            else:
                if i + 1 < len(args_list) and not args_list[i + 1].startswith("-"):
                    logging.warning(f"Unmapped CLI option '{arg} {args_list[i + 1]}' skipped for RC API")
                    i += 1
                else:
                    logging.warning(f"Unmapped CLI flag '{arg}' skipped for RC API")
        i += 1

    # Attach backend-specific connection string parameters to path1
    if backend_options:
        curr_remote = bisync_params["path1"]
        if curr_remote.endswith(":"):
            base_remote = curr_remote[:-1]
            opts_str = ",".join(backend_options)
            bisync_params["path1"] = f"{base_remote},{opts_str}:"

    # Strict Canary Guard: checkAccess requires RCLONE_TEST on local filesystem
    if bisync_params.get("checkAccess"):
        test_file = os.path.join(local_path, "RCLONE_TEST")
        if not os.path.exists(test_file):
            raise FileNotFoundError(
                f"Safety guard triggered: --check-access is enabled, but '{test_file}' was not found. "
                "Aborting sync to prevent unintended deletions or sync of an unmounted filesystem."
            )

    if engine_config:
        bisync_params["_config"] = engine_config

    return bisync_params
