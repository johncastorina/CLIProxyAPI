# Local use security guide

Build in the reviewed source repository and verify the generated executable before moving to a private runtime directory:

```sh
cd /path/to/reviewed/CLIProxyAPI
git rev-parse HEAD
go mod verify
go build -trimpath -o cli-proxy-api ./cmd/server
./cli-proxy-api --help
```

Run CLIProxyAPI from a private working directory because the directory may contain configuration, authentication material, and logs. Create the directory with a restrictive process umask so new files are private by default:

```sh
umask 077
mkdir -p "$HOME/.local/share/cliproxyapi/auths"
cd "$HOME/.local/share/cliproxyapi"
```

Use unique, randomly generated client and management keys. Do not reuse the literal placeholder keys from an example configuration. Keep the server on loopback and leave both management panel controls disabled unless you have reviewed and accepted the added exposure:

```yaml
host: "127.0.0.1"
port: 8317
auth-dir: "./auths"
api-keys:
  - "replace-with-a-unique-random-client-key"
request-log: false
remote-management:
  allow-remote: false
  secret-key: "replace-with-a-different-unique-random-management-key"
  disable-control-panel: true
  disable-auto-update-panel: true
plugins:
  enabled: false
```

Keeping `request-log` disabled suppresses error conversation logs that can contain request or response content. The configured client and management keys remain sensitive to the local administrator and anyone who can read the process or its files.

CLIProxyAPIHome is not required for local use. Dynamic plugins are also optional and should remain disabled when they are not needed; enabled plugins are trusted code loaded into the process.

Start with `--local-model` to use the model catalogs committed in the reviewed source tree and skip remote model catalog updates:

```sh
/absolute/path/to/reviewed/CLIProxyAPI/cli-proxy-api --config ./config.yaml --local-model
```

Explicitly enabling the management panel also enables trust in the configured release publisher and its downloadable panel asset. These settings reduce accidental network and management exposure. A process administrator can still read local secrets, replace the executable, change its configuration, or inspect its memory. A local build also does not establish binary provenance by itself; record the reviewed commit and protect the source tree, build environment, and resulting executable according to your trust requirements.
