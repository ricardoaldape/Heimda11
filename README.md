# Heimda11

**Heimda11** is a self-hosted-first Digital Workforce OS for governing AI workers across models, frameworks and vendors.

The project treats an AI agent as an accountable digital worker with an identity, role, owner, permissions, compute budget, performance history and lifecycle.

## Product direction

Heimda11 is built around eight connected systems:

- **Registry / Control** — system of record for AI workers, ownership, org structure and lifecycle.
- **Gate** — authorization, policy enforcement and human approval.
- **Meter** — token/compute/API budget, cost attribution and ROI.
- **Watch** — traces, outcomes, incidents and operational health.
- **Vault** — short-lived credentials and secret brokering for agents.
- **Relay** — retries, provider/model fallback and continuity.
- **Flow** — governed agent-to-agent and human-in-the-loop workflows.
- **Memory** — controlled enterprise memory and knowledge access.

Phase 1 implements the minimum viable digital worker: **Registry + Gate + Meter + Watch**.

## Principles

1. **Self-hosted first.** Customer workloads, logs and secrets can remain inside customer infrastructure.
2. **Vendor neutral.** OpenAI, Anthropic, Gemini, Qwen, local models and custom agents are first-class citizens.
3. **Human authority.** Risky or financial actions can require explicit human approval.
4. **Open protocols.** MCP, A2A and OpenTelemetry are preferred over proprietary lock-in.
5. **Auditable by default.** Every governed action has an identity, policy decision and trace.
6. **Portable data.** Customers can export their data and configuration.
7. **Free + Pro.** Community self-hosted edition plus a full commercial Pro edition.

## Status

Heimda11 is in active development. The repository was initialized on 2026-10-05.

See `docs/architecture.md` and the project issues for the implementation roadmap.
