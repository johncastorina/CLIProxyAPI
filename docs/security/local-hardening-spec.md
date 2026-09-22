# Local security hardening specification

Scope: the CLIProxyAPI fork, based on upstream commit 555662940411a07460e9d24d14477a5f50dffdb5. Preserve native provider OAuth, account selection, protocol translation, and streaming. No account credentials or live inference are used during implementation. EasyCLIProxyAPI is outside this change.

Accepted security requirements:

1. Management source-address authorization uses the socket peer, never caller-supplied forwarding headers. The key remains required. Other framework client-IP use must not trust arbitrary proxy headers by default.
2. The default host is 127.0.0.1 and the management panel and automatic panel updates default to disabled, in both config parsing paths and the example. Explicit network opt-in remains possible.
3. Panel download fails closed: remove the fallback website; require a valid SHA-256 from release metadata; reject missing/malformed/mismatched digests before installing content; preserve any existing panel on failure. Re-enabling the panel explicitly trusts its selected release publisher; a checksum is not publisher provenance.
4. xAI refresh validates both stored and discovered token endpoints against HTTPS/x.ai policy before a token leaves the process. Reject credential-bearing URL userinfo. Redirects must not carry OAuth POST bodies to an unapproved endpoint. Tests use fake tokens and local/custom transports, not providers.
5. Auth, credential-bearing config, and detailed log files are owner-readable/writable only on Unix, including existing permissive files. New private directories are owner-only; do not chmod arbitrary pre-existing parent directories. Shared writing helpers may be introduced for this repeated boundary; reject symlink secret destinations rather than following them.
6. Disabling request logging must prevent forced error logs from persisting request/response bodies. Existing explicit logging retains functionality with private files. Keep ordinary redacted diagnostic logging.
7. Upgrade the audited vulnerable Go dependency pins to fixed versions, with minimum necessary transitive changes. Do not pretend unused OpenPGP or optional Git features are proven current remote exploits.
8. The fork uses read-only CI with immutable action references and the checked-in model catalog. Remove inherited release/Docker publishing and cross-branch automation rather than retaining an unaudited publishing path. CI must test, vet, build, and invoke the actual built executable's help path, without credentials or live provider calls. No release-binary provenance claim is made before a separately reviewed release process exists.
9. Home/cluster and native plugins remain off in the documented local profile. They are existing explicit high-trust admin features, not repaired by pretending same-channel hashes are signatures. Document remaining opt-in authority honestly; no remote enrollment or plugin install is performed.

Validation: baseline suite; RED then GREEN regressions for each changed security behavior; package tests; complete Go tests/vet/build; actual executable help invocation; source and quality review. Preserve baseline failure evidence if encountered; do not silently alter unrelated behavior.
