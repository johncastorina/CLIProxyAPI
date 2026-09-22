#!/usr/bin/env python3
"""Exercise local authentication boundaries against a built CLIProxyAPI binary."""

from __future__ import annotations

import json
import os
from pathlib import Path
import secrets
import socket
import stat
import subprocess
import sys
import tempfile
import time
from urllib import error, request


STARTUP_TIMEOUT_SECONDS = 20
REQUEST_TIMEOUT_SECONDS = 2


def available_loopback_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
        listener.bind(("127.0.0.1", 0))
        return int(listener.getsockname()[1])


def get_status(opener: request.OpenerDirector, url: str, token: str | None = None) -> int:
    headers = {}
    if token is not None:
        headers["Authorization"] = f"Bearer {token}"
    req = request.Request(url, headers=headers, method="GET")
    try:
        with opener.open(req, timeout=REQUEST_TIMEOUT_SECONDS) as response:
            response.read()
            return int(response.status)
    except error.HTTPError as exc:
        exc.read()
        return int(exc.code)


def require_status(
    opener: request.OpenerDirector,
    url: str,
    expected: int,
    token: str | None = None,
) -> None:
    actual = get_status(opener, url, token)
    if actual != expected:
        raise RuntimeError(f"GET {url} returned {actual}, expected {expected}")


def wait_for_models(
    process: subprocess.Popen[bytes],
    opener: request.OpenerDirector,
    url: str,
    client_key: str,
    log_path: Path,
) -> None:
    deadline = time.monotonic() + STARTUP_TIMEOUT_SECONDS
    last_error: Exception | None = None
    while time.monotonic() < deadline:
        if process.poll() is not None:
            output = log_path.read_text(encoding="utf-8", errors="replace")
            raise RuntimeError(
                f"server exited with status {process.returncode} before readiness\n{output}"
            )
        try:
            require_status(opener, url, 200, client_key)
            return
        except (error.URLError, TimeoutError, RuntimeError) as exc:
            last_error = exc
            time.sleep(0.1)
    raise RuntimeError(f"server did not become ready at {url}: {last_error}")


def run(binary: Path) -> None:
    if not binary.is_file():
        raise RuntimeError(f"built server binary does not exist: {binary}")

    client_key = secrets.token_urlsafe(32)
    management_key = secrets.token_urlsafe(32)
    wrong_key = secrets.token_urlsafe(32)
    port = available_loopback_port()

    with tempfile.TemporaryDirectory(prefix="cli-proxy-security-") as temp_name:
        work_dir = Path(temp_name)
        work_dir.chmod(0o700)
        auth_dir = work_dir / "auths"
        config_path = work_dir / "config.yaml"
        config_text = f"""host: "127.0.0.1"
port: {port}
auth-dir: {json.dumps(str(auth_dir))}
api-keys:
  - {json.dumps(client_key)}
request-log: false
logging-to-file: false
remote-management:
  allow-remote: false
  secret-key: {json.dumps(management_key)}
  disable-control-panel: true
  disable-auto-update-panel: true
plugins:
  enabled: false
discovery:
  enabled: false
pprof:
  enable: false
  addr: "127.0.0.1:0"
"""
        config_path.write_text(config_text, encoding="utf-8")
        config_path.chmod(0o600)

        child_environment = {
            "LANG": "C.UTF-8",
            "PATH": os.environ.get("PATH", "/usr/bin:/bin"),
        }
        log_path = work_dir / "server.log"
        process: subprocess.Popen[bytes] | None = None
        with log_path.open("wb") as log_file:
            try:
                process = subprocess.Popen(
                    [str(binary), "--config", str(config_path), "--local-model"],
                    cwd=work_dir,
                    env=child_environment,
                    stdin=subprocess.DEVNULL,
                    umask=0o022 if os.name == "posix" else -1,
                    stdout=log_file,
                    stderr=subprocess.STDOUT,
                )
                opener = request.build_opener(request.ProxyHandler({}))
                base_url = f"http://127.0.0.1:{port}"
                models_url = f"{base_url}/v1/models"
                wait_for_models(process, opener, models_url, client_key, log_path)
                if os.name != "nt":
                    auth_mode = stat.S_IMODE(auth_dir.stat().st_mode)
                    if auth_mode != 0o700:
                        raise RuntimeError(f"server-created auth directory mode is {auth_mode:o}, expected 700")

                require_status(opener, models_url, 401)
                require_status(opener, models_url, 401, wrong_key)
                require_status(opener, models_url, 200, client_key)
                require_status(opener, f"{base_url}/management.html", 404)

                management_url = f"{base_url}/v0/management/config"
                require_status(opener, management_url, 401)
                require_status(opener, management_url, 401, wrong_key)
                require_status(opener, management_url, 200, management_key)
            finally:
                if process is not None and process.poll() is None:
                    process.terminate()
                    try:
                        process.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        process.kill()
                        process.wait(timeout=5)


def main() -> int:
    if len(sys.argv) != 2:
        print(f"usage: {Path(sys.argv[0]).name} BUILT_SERVER_BINARY", file=sys.stderr)
        return 2
    try:
        run(Path(sys.argv[1]).resolve())
    except Exception as exc:  # Keep CI output concise while preserving the exact failure.
        print(f"local security smoke test failed: {exc}", file=sys.stderr)
        return 1
    print("local security smoke test passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
