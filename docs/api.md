# API quick start

All protected endpoints use `Authorization: Bearer <token>`.

## Register an AI worker

```http
POST /v1/agents
{
  "name":"SOFIA-SALES-01",
  "role":"SDR",
  "department":"Sales",
  "human_owner":"Sales Ops",
  "autonomy_level":2,
  "monthly_budget_usd":75,
  "allowed_tools":["crm","gmail"]
}
```

The response contains the agent API key **once**.

## Create a Gate policy

```http
POST /v1/policies
{
  "name":"Large refunds need a human",
  "priority":100,
  "agent_id":"*",
  "tool":"stripe",
  "action":"refund",
  "min_amount_usd":500,
  "effect":"approval_required",
  "reason":"Financial action above delegated authority"
}
```

## Evaluate an action

```http
POST /v1/gate/evaluate
{
  "agent_id":"agt_...",
  "tool":"stripe",
  "action":"refund",
  "amount_usd":800,
  "trace_id":"customer-ticket-221"
}
```

Possible decisions are `allow`, `deny`, and `approval_required`.

## Record model usage

```http
POST /v1/meter/usage
{
  "agent_id":"agt_...",
  "provider":"openai",
  "model":"model-name",
  "input_tokens":1200,
  "output_tokens":300,
  "cost_usd":0.04,
  "trace_id":"task-123"
}
```

## Record outcome

```http
POST /v1/watch/events
{
  "agent_id":"agt_...",
  "type":"task.completed",
  "status":"success",
  "trace_id":"task-123",
  "value_usd":42,
  "message":"Qualified lead delivered"
}
```

## Vault lease

An administrator creates a lease for an agent. The agent redeems it once before expiration:

```
POST /v1/vault/leases
POST /v1/vault/leases/{lease_id}/redeem
```

## Relay

Configure one or more providers in `/v1/relay/providers`, each referencing a Vault secret. Then an agent calls:

```http
POST /v1/relay/chat/completions
{
  "agent_id":"agt_...",
  "trace_id":"task-123",
  "payload":{
    "model":"requested-model",
    "messages":[{"role":"user","content":"Hello"}]
  }
}
```

Heimda11 attempts providers by ascending priority and returns the first successful response.

## Flow

Create a definition, start a run, then inspect the current step:

```
POST /v1/flows
POST /v1/flows/{flow_id}/runs
GET  /v1/flow-runs/{run_id}/current
POST /v1/flow-runs/{run_id}/complete
```

Gate is evaluated automatically for every current step.
