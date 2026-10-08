# Security Policy

Heimda11 is security-sensitive infrastructure. Please report suspected vulnerabilities through a **private GitHub Security Advisory** for this repository rather than a public issue.

Do not include customer state, API keys, Vault master keys, agent credentials or other secrets in public reports.

## Supported version

The current `main` release line is supported. Security fixes should be applied by upgrading to the newest published release.

## Deployment responsibility

Self-hosted operators are responsible for TLS, network access controls, backups and protecting `HEIMDA11_ADMIN_TOKEN` and `HEIMDA11_MASTER_KEY`.

See [docs/security.md](docs/security.md) for the runtime threat model.
