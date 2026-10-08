# Heimda11

**Heimda11** is a self-hosted-first Digital Workforce OS: a system of record and control plane for AI workers.

It treats an AI agent like an accountable digital worker with an identity, human owner, role, permissions, compute budget, operational history, certifications and lifecycle.

## What ships in v0.1.0

- **Registry / Control** — AI worker identity, role, owner, department, autonomy, budget and lifecycle.
- **Gate** — default-deny authorization, tool allow-lists and human approval.
- **Meter** — token/model/provider spend, monthly budgets and cost attribution.
- **Watch** — operational events, traces, incidents, task outcomes and ROI.
- **Vault** — AES-256-GCM encrypted secrets and short-lived one-time leases.
- **Relay** — OpenAI-compatible model-provider fallback with telemetry.
- **Flow** — governed multi-agent workflows; Gate is enforced before every step.
- **Memory** — self-hosted enterprise context with namespaces, tags and agent ACLs.
- **Training** — certifications tied to AI worker identities.

The server and administration console are a single Go binary. No mandatory cloud service or external database is required.

## Quick start

```sh
git clone https://github.com/ricardoaldape/Heimda11.git
cd Heimda11
cp .env.example .env
```

Generate two long secrets:

```sh
openssl rand -hex 32
openssl rand -hex 32
```

Put one in `HEIMDA11_ADMIN_TOKEN` and the other in `HEIMDA11_MASTER_KEY`, then:

```sh
docker compose up -d --build
```

Open `http://localhost:8080` on a trusted network. Use HTTPS before exposing the control plane outside a private network.

## Core security rules

- Gate is **default deny**.
- Agent API keys are scoped to one worker identity and only hashes are persisted.
- Vault values are encrypted with a customer-owned master key.
- Secret leases are one-time and expire in at most 15 minutes.
- Flow cannot complete a step until Gate authorizes it.
- Admin-only configuration is separated from agent operational credentials.
- Customer state stays on the customer's host.

## Editions

**Community** is self-hosted and supports up to 10 registered AI workers.

**Pro** is the commercial direction: unlimited workforce plus enterprise distribution/support. The architecture is intentionally self-hosted-first so customer token, inference, logs and storage costs remain customer-controlled.

## Development

```sh
go test ./...
go vet ./...
go build ./cmd/heimda11
```

CI is configured for a GitHub **self-hosted runner** and produces Linux and Windows AMD64 binaries.

## Documentation

- [Architecture](docs/architecture.md)
- [Deployment](docs/deployment.md)
- [Security model](docs/security.md)
- [API quick start](docs/api.md)

## Status

v0.1.0 is the first integrated self-hosted product cut. The source of truth for development, review and releases is this GitHub repository.
