---
name: ai-readiness-review
description: Evaluate whether a new AI coding agent can successfully work on this repository using only the AI Knowledge Base. Simulate real AI coding tasks, identify missing knowledge, and measure AI readiness.
---

# Role

You are NOT the maintainer.

You are NOT the documentation author.

You are NOT reviewing code quality.

You are simulating a brand-new AI coding agent.

Assume you have never seen this repository before.

Your goal is to determine whether the existing AI Knowledge Base is sufficient for you to complete real software engineering tasks without reading large portions of the source code.

Think exactly like a coding model such as Claude Code, Codex, Cursor, Gemini, Copilot, or ChatGPT.

---

# Mission

Evaluate whether the repository is truly AI-first.

Measure how effectively the AI Knowledge Base replaces reading source code.

The fewer source files an AI needs to inspect,

the better.

---

# Ground Rules

Pretend you are starting from zero.

Do NOT use prior knowledge about this framework.

Always attempt to solve tasks using only:

- AI Knowledge Base
- Machine-readable metadata
- Examples
- Skills
- Public documentation

Only inspect source code when absolutely necessary.

Whenever you must inspect source code,

record WHY.

That indicates missing knowledge.

---

# Simulated Tasks

Simulate realistic software engineering work.

Examples include

- create a controller

- create a module

- add middleware

- create websocket endpoint

- create scheduler job

- add broker event

- implement authentication

- implement authorization

- create interceptor

- create guard

- create cache layer

- create custom provider

- extend dependency injection

- debug middleware issue

- debug websocket lifecycle

- investigate panic

- optimize performance

- review pull request

- refactor existing feature

- rename public API

- migrate old API

- add tracing

- add logging

- write tests

- add benchmarks

- fix concurrency issue

- add transport

- add extension point

Generate additional realistic tasks whenever appropriate.

---

# For Every Task

Attempt to complete the task mentally.

Determine

Could this be completed using only the Knowledge Base?

If not,

what information is missing?

Which document failed?

Which metadata was missing?

Which source files had to be inspected?

Exactly why?

---

# Source Code Dependency

Track every source file you needed.

Categorize

Not needed

Helpful

Required

Impossible without

The objective is to minimize this list.

---

# Hallucination Risk

For every task,

estimate

How likely is an AI to invent incorrect code?

Explain why.

Examples

Missing lifecycle

Missing ownership

Missing invariant

Undocumented extension point

Unclear naming

Incomplete examples

Poor discoverability

---

# Discoverability

Evaluate whether important concepts are easy to discover.

Would a new AI know where to look?

Would multiple documents need to be searched?

Would terminology confuse retrieval?

Would retrieval produce conflicting answers?

---

# Context Efficiency

Estimate

How many tokens would an AI need before writing correct code?

Identify

Large documents

Repeated explanations

Poor structure

Missing summaries

Missing indexes

Poor cross references

Recommend reductions.

---

# Knowledge Gaps

Identify everything that forced source-code reading.

Examples

Undocumented lifecycle

Missing ownership

Unknown side effects

Unknown concurrency assumptions

Unknown recovery behavior

Unknown extension mechanism

Unknown invariants

Unknown API contracts

Unknown dependency graph

---

# Retrieval Quality

Evaluate

Document naming

Section naming

Heading quality

Keyword density

Cross references

Machine readability

JSON metadata usefulness

Skills usefulness

Examples usefulness

---

# Skills Review

Pretend you only have the available Skills.

Could you perform common engineering tasks?

Identify

Missing Skills

Redundant Skills

Poorly scoped Skills

Skills with too much context

Skills missing prerequisites

Recommend improvements.

---

# Machine Metadata Review

Evaluate whether

api-index.json

architecture.json

dependency-graph.json

ownership.json

lifecycle.json

patterns.json

anti-patterns.json

provide enough information for an autonomous coding agent.

Identify missing metadata.

---

# AI Failure Analysis

Predict where future AI agents are most likely to fail.

Examples

Wrong lifecycle

Wrong dependency injection

Wrong middleware order

Improper pooling

Context misuse

Improper concurrency

Resource leaks

Architecture violations

Document every predicted failure.

---

# Repository AI Score

Score

Architecture Discoverability

Knowledge Coverage

Retrieval Quality

Task Completion

Machine Metadata

Skills

Token Efficiency

Hallucination Resistance

Developer Experience

Overall AI Readiness

All scores from 0-100.

---

# Recommendations

Recommend improvements in priority order.

Estimate

Expected reduction in required context.

Expected reduction in hallucinations.

Expected reduction in token usage.

Expected improvement in task success.

---

# Final Verdict

Answer one question:

"If tomorrow an advanced coding agent joins this repository for the first time, can it confidently implement new features without spending significant time reading source code?"

Justify your answer with evidence collected during the review.