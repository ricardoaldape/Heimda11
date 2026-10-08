# Security model

Heimda11 sits on a sensitive control path. Treat the deployment like an identity or secrets system.

## Required controls

- Put Heimda11 behind HTTPS (reverse proxy, private ingress or VPN).
- Generate a long random `HEIMDA11_ADMIN_TOKEN`.
- Generate a unique 32-byte Vault key and provide it as 64 hex characters in `HEIMDA11_MASTER_KEY`.
- Never commit either value.
- Back up the persistent data volume and the master key separately.
- Restrict network access to the control panel.
- Run the container as the included non-root user.
- Rotate agent keys and provider secrets when personnel or integrations change.

## Authorization

Gate is default-deny. A missing policy is not an implicit allow.

Agent bearer keys are scoped to one Agent ID. Operational endpoints reject an agent attempting to submit usage, events, Gate decisions or Relay requests for another Agent ID.

## Vault

Vault data is encrypted with AES-256-GCM. The master key is not stored in the state file. One-time leases expire in at most 15 minutes and cannot be redeemed by another agent.

Relay provider credentials are read from Vault internally.

## Human approval

An `approval_required` Gate result is not equivalent to an allow. External executors must wait for the approval status before executing. Heimda11 Flow does this automatically.

## Relay

Relay base URLs are configured only by administrators. Both HTTP and HTTPS are supported because local inference servers frequently use private HTTP; Internet-facing providers should always use HTTPS.

## Current trust boundary

The initial store is single-process and single-node. Do not mount the same state file into multiple running Heimda11 containers. High-availability storage is a future adapter, not something the JSON store pretends to provide.

## Reporting vulnerabilities

Do not post credentials, exploit payloads containing private data, or customer state in public issues. Use a private security advisory in GitHub for vulnerabilities.
