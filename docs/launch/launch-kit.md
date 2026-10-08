# Heimda11 v0.1.0 — Launch Kit

Heimda11 is a **self-hosted Digital Workforce OS** for AI agents.

Core message:

> Give every AI worker an identity, a boss, permissions, a budget, credentials, memory, performance history and a kill switch.

Repository: https://github.com/ricardoaldape/Heimda11  
Release: https://github.com/ricardoaldape/Heimda11/releases/tag/v0.1.0

## Commercial offer

### Community — free
- Self-hosted
- Up to 10 registered AI workers
- Registry / Control
- Gate
- Meter
- Watch
- Vault
- Relay
- Flow
- Memory
- Training

### Founding Company — USD $500/month
Target: companies already operating multiple AI agents in production.

Offer:
- Pro usage for the organization
- Guided self-hosted onboarding
- Architecture review for the company's agent workforce
- Initial policy / budget / approval setup
- Direct product feedback channel
- Early access to enterprise capabilities

The customer supplies its infrastructure and AI-provider credentials.

## X launch

Your AI agents are becoming employees.

But most companies still don't know:
- which agents exist
- what they can access
- what they spend
- who owns them
- what happens when they make a mistake

I built Heimda11: a self-hosted Digital Workforce OS.

Identity. Permissions. Budgets. Human approvals. Secrets. Observability. Memory. Governed workflows.

Runs on your infrastructure.

GitHub:
https://github.com/ricardoaldape/Heimda11

## LinkedIn launch

AI agents are quickly becoming part of the workforce, but most companies are still managing them like scripts.

That creates a new operational problem.

Who owns each agent?
What systems can it access?
How much can it spend?
Which credentials can it use?
How is its performance measured?
What happens when an action is risky?
How do you deactivate it safely?

I built Heimda11 to explore that problem as infrastructure rather than as another agent framework.

Heimda11 is a self-hosted Digital Workforce OS. Every AI worker gets an identity, role, human owner, autonomy level, budget, permissions, audit history and lifecycle.

The first release includes Registry/Control, Gate, Meter, Watch, Vault, Relay, Flow, Memory and Training.

It is vendor-neutral and self-hosted first. Your operational data, secrets and logs can remain inside your infrastructure.

Community edition is free and available now.

I'm also opening a small Founding Company program for teams already running multiple AI agents in production.

GitHub:
https://github.com/ricardoaldape/Heimda11

## Hacker News

Title:

Show HN: Heimda11 – a self-hosted operating system for AI workers

Submission URL:

https://github.com/ricardoaldape/Heimda11

First comment:

I built Heimda11 around a question I think more teams will have as agents move from demos into production: if an AI agent is allowed to do real work, what is the equivalent of employee identity, permissions, budget, supervision and offboarding?

Instead of building another agent framework, Heimda11 treats agents as a digital workforce.

The current release is a single self-hosted Go binary with an embedded UI. It includes an agent registry, default-deny policy engine, human approvals, token/cost accounting, event tracing, encrypted secrets with short-lived leases, provider fallback, private memory, certifications and governed multi-agent flows.

A design constraint was that the customer should be able to keep the control plane, credentials and operational history on their own infrastructure.

It is early and intentionally small. I would especially like feedback on:
1. whether “digital workforce management” matches a real operational problem for teams running agents;
2. which integrations would make it useful in an existing agent stack;
3. where the security model is still too naive.

There is no signup required to try it.

## Reddit — r/selfhosted

Title:

I built a self-hosted control plane for treating AI agents like digital workers

Body:

I've been thinking about a problem that appears after the “build an agent” stage: what happens when a company has many agents doing real work?

I built Heimda11 as a self-hosted Digital Workforce OS rather than another agent framework.

Each agent can have:
- an identity and human owner
- role / department / autonomy level
- tool permissions
- a monthly AI budget
- default-deny policies
- human approval for sensitive actions
- encrypted/scoped secrets
- traces and performance history
- private memory
- certifications
- governed multi-agent workflows

The control plane is a single Go binary with an embedded UI and can run entirely inside your infrastructure.

The Community edition is free and currently supports up to 10 registered AI workers.

GitHub:
https://github.com/ricardoaldape/Heimda11

I'd genuinely like feedback from people running self-hosted agents: which part of operating agents becomes painful first for you — permissions, credentials, observability, cost, memory or reliability?

AI-use disclosure: AI tools were used extensively during development, review and testing. The project itself is executable software with CI and releases built on a self-hosted runner.

## Reddit — r/LocalLLaMA / agent communities

Title:

Heimda11: self-hosted identity, permissions, budgets and operations for AI agents

Body:

Most agent projects focus on creating the agent. I wanted to work on the layer that becomes necessary after you have several of them in production.

Heimda11 treats an agent as a digital worker with an identity, owner, permissions, budget, secrets, performance history and lifecycle.

It is self-hosted and model/vendor neutral. The current release includes Registry/Control, Gate, Meter, Watch, Vault, Relay, Flow, Memory and Training.

The Relay path is OpenAI-compatible and the architecture is intended to sit around existing agent stacks rather than replace them.

GitHub:
https://github.com/ricardoaldape/Heimda11

I'm especially interested in what local/self-hosted users would want first: Ollama/vLLM integrations, MCP policy enforcement, OpenTelemetry, or something else.

## DEV.to article

Title:

Your AI agents are becoming employees. Who manages them?

Tags:

ai, opensource, selfhosted, agents

Opening:

The first wave of agent tooling answered: “How do I build an agent?”

The next operational question is different:

“What happens when my company has 10, 100 or 1,000 agents doing real work?”

At that point the problem starts to resemble workforce operations: identity, ownership, permissions, budgets, credentials, performance, supervision and offboarding.

That is the idea behind Heimda11, a self-hosted Digital Workforce OS.

### The digital-worker model

A Heimda11 agent record can carry a role, department, human owner, autonomy level, monthly budget and allowed tools.

Gate evaluates actions with default-deny policies and can require human approval.

Meter attributes model/token cost to the worker.

Watch records traces, failures, value and performance.

Vault encrypts provider credentials and can issue short-lived one-time leases.

Relay adds provider fallback.

Flow runs multi-agent processes but forces each step through Gate.

Memory provides private operational context with per-agent ACLs.

Training records certifications.

### Why self-hosted first?

If this layer eventually controls many AI workers, it becomes sensitive infrastructure. Operational logs, credentials and company memory should not be forced through a third-party SaaS just to manage the agents.

Heimda11 therefore ships as a single Go binary with an embedded administration console and Docker support.

### Try it

https://github.com/ricardoaldape/Heimda11

The Community edition is free. Feedback from teams already running agents in production is particularly useful.

## Short demo scenarios

1. **Refund guardrail**
   - AI worker attempts a high-value refund.
   - Gate returns approval_required.
   - Human approves/rejects.
   - Audit event is retained.

2. **Digital payroll**
   - Worker records model usage.
   - Meter attributes monthly spend.
   - Worker exceeds budget.
   - Watch records the incident.

3. **Credential isolation**
   - Agent requests a scoped secret.
   - Vault creates a short-lived one-time lease.
   - Another agent cannot redeem it.

4. **Governed workflow**
   - Multi-agent workflow reaches a sensitive step.
   - Flow pauses because Gate requires human approval.
   - Workflow resumes only after approval.

## Launch metrics

Track:
- GitHub stars
- unique cloners / traffic
- release downloads
- issues / discussions
- inbound DMs
- Community installs
- Founding Company conversations
- paid conversions

The first commercial validation milestone is **5 paying companies**, not vanity reach.
