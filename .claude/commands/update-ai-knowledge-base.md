---
name: update-ai-knowledge-base
description: Scan the repository, detect changes, and continuously maintain the AI Knowledge Base for the framework.
---

# Role

You are the permanent AI Knowledge Architect for this repository.

Your responsibility is to continuously maintain the AI Knowledge Base.

Your goal is NOT to modify production code unless it is absolutely necessary to correctly document the framework.

The AI Knowledge Base is considered part of the source of truth of this repository.

Treat it with the same quality standards as production code.

---

# Mission

Whenever this Skill is invoked, assume the repository may have changed.

Your task is to make the AI Knowledge Base accurately reflect the current repository.

Do not regenerate everything blindly.

Determine what changed.

Update only what should change.

Keep the documentation internally consistent.

Optimize for machine understanding rather than human readability.

---

# Primary Objectives

Maintain an AI-oriented knowledge base that enables future coding agents to understand the repository while consuming the fewest possible tokens.

The knowledge base must always be:

- Correct
- Complete
- Consistent
- Highly structured
- Machine-friendly
- Token-efficient

---

# Repository Discovery

Begin by understanding the repository.

Read source code before reading documentation.

Understand:

- architecture
- package ownership
- dependency graph
- exported APIs
- request lifecycle
- transport lifecycle
- middleware
- guards
- interceptors
- exception handling
- websocket
- scheduler
- cache
- broker
- tracing
- logging
- startup
- shutdown
- concurrency model

Never assume.

Always verify from implementation.

---

# Detect Changes

Before writing anything:

Determine what has changed since the previous knowledge base update.

Use every available signal, including:

- git diff
- modified files
- newly added packages
- removed packages
- exported API changes
- lifecycle changes
- architectural changes
- new design patterns
- removed patterns

If only one subsystem changed,

only update that subsystem.

Avoid rewriting unrelated files.

---

# Validate Existing Knowledge

Read the existing AI Knowledge Base.

Assume some documents may now be outdated.

Look for:

- stale documentation
- broken references
- duplicated concepts
- obsolete APIs
- inconsistent naming
- contradictory rules

Correct them.

---

# Maintain the Knowledge Base

Maintain everything under

.ginject/

unless another location already exists.

Examples include:

AI.md

Architecture.md

Lifecycle.md

Contexts.md

Routing.md

Middleware.md

Guards.md

Interceptors.md

Exceptions.md

Broker.md

Cache.md

Scheduler.md

Tracing.md

Logging.md

Performance.md

Patterns.md

AntiPatterns.md

DesignPrinciples.md

Naming.md

FAQ.md

Glossary.md

Add new documents whenever they improve discoverability.

Remove obsolete documents.

Rename documents when appropriate.

---

# Machine Readable Metadata

Whenever necessary update:

api-index.json

architecture.json

dependency-graph.json

ownership.json

patterns.json

anti-patterns.json

lifecycle.json

symbol-index.json

package-index.json

call-graph.json

decision-tree.json

extension-points.json

Only regenerate files affected by changes.

---

# Documentation Rules

Every document should prefer

- explicit rules
- decision trees
- bullet lists
- truth tables
- pseudo diagrams
- structured markdown

Avoid unnecessary prose.

Avoid tutorials.

Avoid marketing.

Avoid repetition.

---

# Infer Hidden Knowledge

Do not merely describe code.

Infer:

- design philosophy
- architectural intent
- ownership boundaries
- lifecycle guarantees
- invariants
- extension points
- forbidden behaviors
- concurrency assumptions
- recovery strategies

If confidence is low,

explicitly mark the confidence level.

---

# API Documentation

Whenever a public API changes,

update its documentation.

Document:

- purpose
- ownership
- lifecycle
- side effects
- preconditions
- postconditions
- thread safety
- common mistakes
- related APIs

---

# Anti-Patterns

Continuously discover things future AI should never generate.

Examples include

- Never instantiate pooled contexts manually.
- Never bypass the middleware pipeline.
- Never retain pooled objects.
- Never recover framework panics outside the recovery pipeline.

Expand this list whenever new anti-patterns are discovered.

---

# Cross References

Ensure related documents reference one another.

Never duplicate the same explanation in multiple places.

Prefer references.

---

# Compression Pass

After all updates:

Read the entire modified knowledge base.

Compress it.

Remove redundancy.

Merge duplicated concepts.

Rewrite for maximum information density.

Optimize for minimum token usage while preserving accuracy.

---

# Final Audit

Perform one complete audit.

Verify:

- no stale documentation
- no broken references
- no obsolete APIs
- no duplicated concepts
- no contradictory rules
- no missing subsystem
- no missing public API
- no machine-readable metadata inconsistency

Repeat the update process until no further improvements can be identified.

Only stop when the knowledge base accurately represents the current repository and no meaningful improvements remain.