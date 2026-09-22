#!/usr/bin/env python3
"""Launch native Codex with the local subscription proxy; preserve CLI arguments."""

import json
import os
from pathlib import Path
import shutil
import sys


def main():
    codex = shutil.which("codex")
    if codex is None:
        sys.exit("codex-proxy: codex is not available on PATH")
    config_path = Path.home() / ".local/share/cliproxyapi/config.yaml"
    try:
        config = json.loads(config_path.read_text())
        port = config["port"]
        token = config["api-keys"][0]
        if config.get("host") != "127.0.0.1":
            raise ValueError("non-loopback proxy")
        if type(port) is not int or not 1 <= port <= 65535:
            raise ValueError("invalid port")
        if not isinstance(token, str) or not token.strip():
            raise ValueError("missing token")
    except (OSError, ValueError, KeyError, IndexError, TypeError):
        sys.exit("codex-proxy: expected a valid local proxy configuration with a client token")

    provider = (
        '{name="Local subscription proxy",'
        f'base_url="http://127.0.0.1:{port}/v1",'
        'env_key="CLIPROXY_LOCAL_TOKEN",wire_api="responses",'
        'requires_openai_auth=false,supports_websockets=false,'
        'request_max_retries=0,stream_max_retries=0}'
    )
    env = os.environ.copy()
    env["CLIPROXY_LOCAL_TOKEN"] = token
    args = [codex, "-c", 'model_provider="subscription_proxy"',
            "-c", "model_providers.subscription_proxy=" + provider, *sys.argv[1:]]
    os.execve(codex, args, env)


if __name__ == "__main__":
    main()
