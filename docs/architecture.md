# Heimda11 architecture

Heimda11 is a **self-hosted-first Digital Workforce OS**. A deployment is authoritative for one organization and keeps operational data inside that organization's infrastructure.

## Runtime

The first production cut is deliberately small:

- one Go binary;
- embedded administration UI;
- HTTP/JSON API;
- atomic JSON state store on a persistent volume;
- AES-256-GCM encryption for Vault values;
- outbound HTTPS only when Relay calls configured model providers.

The single-node store is suitable for initial and small/medium installations. The domain and service layers are separated from HTTP so a PostgreSQL store can be added without changing the API contracts.

## Systems

### Registry / Control
System of record for every AI worker: identity, role, department, human owner, manager agent, autonomy level, budget, tool allow-list and lifecycle state.

### Gate
Policy engine. Rules are evaluated in priority order. No matching rule means **deny**. Policies can allow, deny or require explicit human approval.

### Meter
Receives token and cost records from agents or integrations, calculates monthly spend and emits budget warnings.

### Watch
Receives operational events and traces. Performance is derived from task outcomes, attributed value and cost.

### Vault
Secrets are encrypted at rest with a customer-owned master key. Agents can receive short-lived, one-time leases; Relay can consume secrets internally without exposing them to the requesting agent.

### Relay
OpenAI-compatible chat relay with ordered providers. It retries the next enabled provider on transport or upstream failure and records each attempt.

### Flow
Sequential governed workflows. Every step is evaluated by Gate before execution. A run pauses for human approval, stops on deny/failure and records completed steps.

### Memory
Self-hosted text memory with namespaces, tags and optional agent ACLs.

### Training
Certification records bind a worker identity to a skill, score, issuer and optional expiration.

## Authentication

There are two credential classes:

1. **Administrator bearer token**: full control plane access.
2. **Agent API key**: generated once when an agent is registered. It can operate only as that agent.

Only hashes of agent API keys are persisted.

## Data ownership

The customer owns the state file and can export it through the admin API. There is no phone-home requirement in the Community build.

## Protocol direction

Adapters for MCP, A2A and OpenTelemetry belong at the edge of the system. Core authorization and workforce records remain protocol-neutral.
