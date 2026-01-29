# Agent Skills for Beads

> Design document for `bd skills` commands
> Status: Draft
> Date: 2026-01-28

## Overview

Add team-shared agent skills to beads — reusable instruction sets that enhance agent behavior. Skills are markdown files (SKILL.md) that agents load on-demand, following the open [Agent Skills specification](https://agentskills.io/specification).

The Agent Skills standard (originated by Anthropic, now adopted by OpenAI Codex, Cursor, Gemini CLI, GitHub Copilot, VS Code, and others) defines a portable, model-agnostic format. Beads implements this standard so that skills work with **any agent** that supports SKILL.md — not just a single vendor. The `bd skills` commands manage the install/update/sync lifecycle; the agent runtime handles discovery and loading.

This enables:

- **Team-shared knowledge** — Skills are git-versioned and auto-installed on `bd init`
- **Progressive disclosure** — Metadata loaded at startup (~100 tokens/skill), full content on-demand
- **Zero friction** — Install once with `bd skills add`, agent uses when relevant
- **Portability** — Same SKILL.md works across Claude Code, Codex, Cursor, Gemini CLI, and more

## The Key Idea: User Installs, Agent Activates

The central design principle is a clean split between **human decisions** and **agent behavior**:

1. **The user decides what skills to install** — a deliberate, one-time choice
2. **The agent decides when to use them** — automatic, based on task relevance

This mirrors how every successful tool ecosystem works:

| Tool | User Action (one-time) | Automatic Behavior |
|------|------------------------|--------------------|
| npm | `npm install eslint` | `require()` resolves automatically |
| VSCode | Install "Prettier" extension | Formats on save |
| Homebrew | `brew install jq` | Available in `$PATH` |
| Docker | `docker pull postgres` | `docker-compose up` uses it |
| Go modules | `go get pkg` | `import` resolves automatically |
| **beads** | `bd skills add security-review` | Agent loads when reviewing code |

> [!IMPORTANT]
> The alternative — the agent asking "Should I load the security skill?" every time it encounters security-related code — is disruptive and defeats the purpose. Once a team decides a skill is valuable, it should just work.

### Example: What This Looks Like in Practice

```
User: "Review this PR for security issues"

Agent (internally):
  1. Reads installed skill metadata (~100 tokens total)
  2. Sees: "security-review: Audits code for OWASP Top 10 vulnerabilities,
     injection risks, auth flaws. Use when reviewing code for security."
  3. Loads full skill: `bd skills load security-review`
  4. Follows skill instructions to perform structured review

User sees: A thorough security review — no setup, no prompting about skills.
```

The user's only interaction with skills was `bd skills add security-review` weeks ago. Everything else is agent-driven.

### Example: Team Onboarding

```bash
# Alice adds a security skill to the project
bd skills add trailofbits/skills/plugins/differential-review
git commit -m "Add security review skill" && git push

# Bob joins the team, clones the repo
git clone ... && bd init
# ✓ Installing team skills: differential-review

# Bob asks the agent to review a PR — the skill activates automatically.
# No setup, no "install these tools first" messages. It just works.
```

---

## Commands

| Command | Description |
|---------|-------------|
| `bd skills add <source>` | Install a skill |
| `bd skills list [--filter]` | List installed skills |
| `bd skills load <name>` | Output skill content (for agent) |
| `bd skills remove <name>` | Remove a skill |
| `bd skills sync` | Re-download from pinned sources |
| `bd skills update [--all]` | Update to latest versions |
| `bd skills outdated` | Check for available updates |
| `bd skills import <path>` | Convert existing docs to a skill |

> [!TIP]
> All commands support `--json` for machine-readable output.

### Usage Examples

```bash
# Install from GitHub shorthand
bd skills add trailofbits/skills/plugins/differential-review

# Install from a full URL
bd skills add https://github.com/anthropics/skills/tree/main/pdf-processing

# Install a Vercel skill
bd skills add vercel-labs/agent-skills --skill frontend-design

# Install as personal skill (not shared with team)
bd skills add my-shortcuts --local

# List all skills
bd skills list
# Output:
# • differential-review (3.2 KB) [team]
#   Security-focused differential review of code changes...
#   Tags: security, code-review, audit
# • pdf-processing (2.8 KB) [team]
#   Extract text and tables from PDFs, fill forms, merge documents...
#   Tags: pdf, documents, extraction

# Agent loads skill content
bd skills load differential-review
# Output: [full SKILL.md content]

# Check for updates
bd skills outdated
# differential-review: a1b2c3 → e5f6g7 (3 commits behind)
```

---

## Install and Load Flow

This section details exactly how skills move from source → project → agent context.

### Phase 1: Installation (`bd skills add`)

```
bd skills add trailofbits/skills/plugins/differential-review
       │
       ▼
┌──────────────────────────────────────────┐
│ 1. Resolve source                        │
│    "trailofbits/skills/plugins/..."      │
│    → https://github.com/trailofbits/     │
│      skills/tree/main/plugins/           │
│      differential-review                 │
│                                          │
│ 2. Download SKILL.md + supporting files  │
│    (scripts/, references/, assets/)      │
│                                          │
│ 3. Parse YAML frontmatter               │
│    → name, description, metadata         │
│                                          │
│ 4. Pin commit hash (e.g., a1b2c3d4)     │
│                                          │
│ 5. Write to .beads/skills.yaml           │
│    (team) or .beads/skills.local.yaml    │
│    (personal with --local)               │
│                                          │
│ 6. Cache content to .claude/skills/      │
│    differential-review/SKILL.md          │
└──────────────────────────────────────────┘
```

After installation, `.beads/skills.yaml` is committed to git. When teammates pull and run `bd init`, the same skills are installed automatically.

### Phase 2: Metadata Loading (`bd prime`)

At agent startup, `bd prime` injects lightweight metadata for all installed skills:

```
Installed Skills:
• differential-review: Security-focused differential review of code
  changes with git history analysis and blast radius estimation.
• pdf-processing: Extract text and tables from PDF files, fill
  forms, merge documents. Use when working with PDF documents.

Load a skill: bd skills load <name>
```

**Cost: ~100 tokens per skill.** Ten skills = ~1k tokens total. This is the same progressive disclosure pattern beads uses for its MCP tools (`discover_tools()` → `get_tool_info()` → full schema).

### Phase 3: On-Demand Loading (`bd skills load`)

When the agent determines a task matches a skill description, it loads the full content:

```
Agent calls: bd skills load differential-review
             │
             ▼
┌──────────────────────────────────────────┐
│ 1. Read .claude/skills/differential-     │
│    review/SKILL.md                       │
│                                          │
│ 2. Output full markdown content          │
│    (~2-5k tokens)                        │
│                                          │
│ 3. Agent now has detailed instructions   │
│    and follows them for the current task │
└──────────────────────────────────────────┘
```

### Phase 4: Resource Loading (Optional)

Skills may reference additional files (scripts, references) that the agent loads only when needed:

```
SKILL.md references:
  → references/owasp-checklist.md    (~1k tokens, loaded if needed)
  → scripts/blast-radius.sh          (executed if needed)
```

### Full Flow Summary

```
                    ┌─────────────┐
                    │  User runs  │
                    │ bd skills   │
                    │   add ...   │
                    └──────┬──────┘
                           │
        ┌──────────────────▼──────────────────┐
        │         .beads/skills.yaml          │
        │     (git-committed, team-shared)     │
        └──────────────────┬──────────────────┘
                           │
              bd init / bd skills sync
                           │
        ┌──────────────────▼──────────────────┐
        │    .claude/skills/<name>/SKILL.md   │
        │         (cached, git-ignored)        │
        └──────────────────┬──────────────────┘
                           │
                      bd prime
                           │
        ┌──────────────────▼──────────────────┐
        │     Agent context: metadata only     │
        │        (~100 tokens/skill)           │
        └──────────────────┬──────────────────┘
                           │
              Agent decides skill is relevant
                           │
        ┌──────────────────▼──────────────────┐
        │   bd skills load <name>              │
        │   Full SKILL.md → agent context      │
        │        (~2-5k tokens)                │
        └──────────────────┬──────────────────┘
                           │
              Agent reads references if needed
                           │
        ┌──────────────────▼──────────────────┐
        │   references/, scripts/, assets/     │
        │        (loaded on-demand)            │
        └──────────────────────────────────────┘
```

---

## File Structure

```
.beads/
├── skills.yaml           # Team skills (git-committed)
└── skills.local.yaml     # Personal skills (git-ignored)

.claude/
└── skills/               # Cached content (git-ignored)
    ├── differential-review/
    │   ├── SKILL.md
    │   ├── scripts/
    │   └── references/
    └── pdf-processing/
        └── SKILL.md
```

> [!NOTE]
> The `.claude/skills/` cache directory is reconstructable — running `bd skills sync` re-downloads everything from the sources pinned in `skills.yaml`. This is analogous to `node_modules` being derived from `package-lock.json`.

---

## Schema

### `skills.yaml`

```yaml
version: 1
skills:
  differential-review:
    source: trailofbits/skills/plugins/differential-review
    commit: a1b2c3d4e5f6
    description: >
      Security-focused differential review of code changes with
      git history analysis and blast radius estimation. Use when
      reviewing PRs, auditing commits, or assessing security impact.
    tags: ["security", "code-review", "audit"]
    size: 3241
    installed_at: "2026-01-28T10:30:00Z"
    installed_by: "alice@example.com"

  pdf-processing:
    source: anthropics/skills/pdf-processing
    commit: f7e8d9c0b1a2
    description: >
      Extract text and tables from PDF files, fill forms, merge
      documents. Use when working with PDF documents or when the
      user mentions PDFs, forms, or document extraction.
    tags: ["pdf", "documents", "extraction"]
    size: 2847
    installed_at: "2026-01-28T11:00:00Z"
    installed_by: "alice@example.com"
```

### SKILL.md Format

Skills follow the [Agent Skills specification](https://agentskills.io/specification) with YAML frontmatter:

```markdown
---
name: differential-review
description: >
  Security-focused differential review of code changes with git
  history analysis and blast radius estimation. Use when reviewing
  PRs, auditing commits, or assessing security impact.
---

# Differential Review

## When to Use
- Reviewing pull requests for security implications
- Auditing commits that touch auth, crypto, or input handling
- Estimating blast radius of code changes

## Steps
1. Identify changed files with `git diff`
2. Classify changes by risk tier...
[Instructions for agent...]
```

---

## Design Decisions

### 1. Description-Based Activation, Not File Patterns

> **Rejected:** `applies_to: ["*.py", "*.sol"]` — too rigid, causes false positives.
>
> **Chosen:** Natural-language description with trigger phrases (e.g., "Use when auditing Solidity code").

**Why this is right:**
- File patterns miss semantic context. A user discussing smart contract security in a `.md` file wouldn't trigger a pattern-based skill — but the description-based approach would.
- File patterns cause false positives. Every `.py` file would trigger a Python skill, even for trivial one-liners.
- The agent's job is reasoning about relevance. Pattern matching is rigid; description matching leverages what agents are good at.
- The Agent Skills spec explicitly states: "Include all 'when to use' information in the description — Not in the body."

**Real-world validation:** Trail of Bits' skill guidelines reinforce this — their descriptions use trigger phrases like "Use when auditing Solidity" and emphasize specificity ("Detects reentrancy vulnerabilities" not "Helps with security").

### 2. Opt-In at Installation, Not During Usage

> **Rejected:** Agent asks "Load the security skill?" before each use.
>
> **Chosen:** User installs once, agent uses automatically.

**Why this is right:**

This is the universal pattern for extending tool capabilities (see the table in [The Key Idea](#the-key-idea-user-installs-agent-activates) above). No successful tool ecosystem asks for permission at **usage time** for something the user already opted into at **install time**. The opt-in is the installation itself.

**Counter-example:** Imagine if VSCode asked "The Prettier extension wants to format this file. Allow?" every time you saved. That's what per-use skill activation would feel like.

### 3. Progressive Disclosure (Metadata → Content → Resources)

> **Rejected:** Load all skills into agent context at startup.
>
> **Chosen:** Three-tier lazy loading matching the Agent Skills spec.

**Why this is right:**

Context bloat is a real, measured problem. From [community research](https://news.ycombinator.com/item?id=46716016):

> "Indiscriminately loading hundreds of tools causes Context Saturation, Tool Bloat, high latency, financial waste, and context rot where the model becomes confused by irrelevant data."

The numbers make this concrete:

| Skills Installed | Metadata Only | If All Loaded | Context Saved |
|------------------|---------------|---------------|---------------|
| 5 | ~500 tokens | ~25k tokens | 98% |
| 10 | ~1k tokens | ~50k tokens | 98% |
| 20 | ~2k tokens | ~100k tokens | 98% |

**Beads already uses this pattern.** The beads MCP server implements the same approach for tool schemas:
- `discover_tools()` → ~500 bytes (names/descriptions only)
- `get_tool_info("ready")` → ~300 bytes (full schema on-demand)
- 95% reduction in initial context overhead

Skills follow the identical pattern. This isn't a new idea — it's consistency with how beads already works.

### 4. External Discovery, Not CLI Search

> **Rejected:** `bd skills search "security"` querying an API.
>
> **Chosen:** Users browse the web ([agentskills.io](https://agentskills.io), GitHub topics, awesome-agent-skills lists), copy install command.

**Why this is right:**
- Beads is offline-first. All `bd skills` commands work without internet after installation.
- Web UIs are better for browsing, ratings, and discovery. CLI search is always a worse browsing experience.
- Clear separation of concerns: discover on web, manage via CLI.
- `--filter` for local filtering of installed skills. `--search` for remote queries doesn't belong.

**Industry parallel:** Package managers increasingly emphasize web-based discovery — npm has [npmjs.com](https://npmjs.com), Rust has [crates.io](https://crates.io), Go has [pkg.go.dev](https://pkg.go.dev). CLI search exists but web UIs provide richer browsing with reviews, documentation, and usage statistics.

### 5. Commit Pinning, Not Branch Tracking

> **Rejected:** Track `main` branch, auto-update on sync.
>
> **Chosen:** Pin to commit hash, explicit `bd skills update`.

**Why this is right:**
- **Reproducibility**: Entire team gets exact same skill content. This is Go modules' approach (`go.sum` pins exact versions).
- **No surprise changes**: A skill author pushing a breaking change doesn't silently change your agent's behavior mid-sprint.
- **Controlled upgrades**: `bd skills outdated` → review changes → `bd skills update`.

**Industry pattern comparison:**

| Tool | Pinning Strategy | Update Command |
|------|-----------------|----------------|
| Go modules | `go.sum` exact hash | `go get -u` |
| npm | `package-lock.json` exact versions | `npm update` |
| Nix | `flake.lock` | `nix flake update` |
| Terraform | `.terraform.lock.hcl` | `terraform init -upgrade` |
| **beads** | `skills.yaml` commit hash | `bd skills update` |

### 6. Team YAML + Local YAML, Not Database

> **Rejected:** SQLite tables, JSONL audit logs.
>
> **Chosen:** `.beads/skills.yaml` (git-committed) + `.beads/skills.local.yaml` (git-ignored).

**Why this is right:**
- **Hand-editable**: Users can modify YAML directly, no special tooling needed.
- **Git-diffable**: Team sees exactly what changed in PRs. A `skills.yaml` diff is immediately readable.
- **Simple sync**: `git pull` gives you the team's skills. No export/import ceremony.
- Follows existing beads patterns (`.beads/config.yaml`, git config vs local config).
- No new dependencies — YAML parsing is already in the beads codebase.

### 7. Cache in `.claude/skills/`, Not `.beads/skills/`

> **Chosen:** Skill content cached in `.claude/skills/` (git-ignored).

**Why this is right:**
- Agent runtimes that support the Agent Skills standard already auto-discover skills from their respective directories (`.claude/skills/`, `.codex/skills/`, etc.).
- No custom discovery protocol needed — beads just writes files where agents already look.
- `.beads/` stays clean — only YAML config, not cached content.
- Like `node_modules`, this cache is fully reconstructable via `bd skills sync`.

> [!NOTE]
> **Cross-agent support:** If the project uses multiple agents, `bd skills sync` could write to multiple agent directories. The skill content is identical — only the cache location differs.

### 8. No Agent Proactive Suggestions

> **Rejected:** Agent detects unfamiliar code patterns and suggests "Install the X skill?"
>
> **Chosen:** Agent only uses already-installed skills.

**Why this is right:**
- **Intrusive**: Interrupts flow to suggest installations during focused work.
- **Wrong opt-in model**: Opt-in happens at install time, not during usage (see [Decision #2](#2-opt-in-at-installation-not-during-usage)).
- **Noisy**: Agent would suggest skills on every mention of a technology it recognizes.

> [!TIP]
> **Exception:** During planning/onboarding, an agent may note that an installed skill is relevant to the task at hand.

---

## Popular Skills in the Ecosystem

The Agent Skills ecosystem has grown rapidly since the standard's publication. These examples illustrate the range of what skills cover:

| Skill | Author | What It Does |
|-------|--------|-------------|
| [Manus-style Planning](https://github.com/topics/agent-skills) | Community | Persistent markdown planning workflow for structured agent work |
| [React Best Practices](https://github.com/vercel-labs/agent-skills) | Vercel | 40+ React/Next.js optimization rules across 8 categories |
| [Frontend Design](https://github.com/vercel-labs/agent-skills) | Vercel | 100+ rules for accessibility, performance, and UX auditing |
| [Security Review](https://github.com/trailofbits/skills) | Trail of Bits | Differential security review, blast radius estimation |
| [Building Secure Contracts](https://github.com/trailofbits/skills) | Trail of Bits | 11 specialized skills for smart contract security |
| [PDF Processing](https://agentskills.io) | Anthropic | Extract text, fill forms, merge PDFs |
| [Expo App Development](https://github.com/topics/agent-skills) | Expo | Build, deploy, and debug React Native/Expo apps |

<details>
<summary><strong>Example: What a real SKILL.md looks like (PDF Processing)</strong></summary>

From the official Agent Skills collection — a popular, model-agnostic example:

```markdown
---
name: pdf-processing
description: >
  Extract text and tables from PDF files, fill PDF forms, and merge
  multiple PDFs. Use when working with PDF documents or when the user
  mentions PDFs, forms, or document extraction.
license: Apache-2.0
metadata:
  author: anthropic
  version: "1.0"
---

# PDF Processing

## When to Use
- User asks to extract text from a PDF
- User needs to fill out a PDF form
- User wants to merge or split PDF files
- Working with document pipelines that involve PDFs

## Text Extraction
Use the extraction script to pull text from PDFs:

    scripts/extract.py <input.pdf> [--format markdown|text|json]

## Form Filling
For PDF forms, first inspect available fields:

    scripts/forms.py inspect <form.pdf>

Then fill using a JSON mapping:

    scripts/forms.py fill <form.pdf> --data '{"field": "value"}'

## Common Edge Cases
- Scanned PDFs: Use OCR flag `--ocr` for image-based PDFs
- Password-protected: Supply `--password` flag
- Large files: Use `--pages 1-10` to process in chunks
```

Note the description pattern: It says both what it does ("Extract text and tables...") and when to use it ("Use when working with PDF documents or when the user mentions PDFs..."). This is what the Agent Skills spec calls for — the description is the activation trigger.

</details>

---

## Integration Points

### `bd init`

Auto-installs team skills from `.beads/skills.yaml`:
```
$ bd init
✓ Installing team skills: differential-review, pdf-processing
```

If a skill's cached content is missing (e.g., fresh clone), `bd init` downloads from the pinned commit.

### `bd prime`

Includes skill metadata in agent context:
```
Installed Skills:
• differential-review: Security-focused differential review of code
  changes with git history analysis and blast radius estimation.
• pdf-processing: Extract text and tables from PDF files, fill forms,
  merge documents. Use when working with PDF documents.

Load a skill: bd skills load <name>
```

This metadata block costs ~100 tokens per skill. The agent uses this to decide relevance — it never loads a skill it doesn't need.

### `bd skills sync`

Re-downloads all skills from their pinned sources:
```
$ bd skills sync
✓ Syncing: differential-review (a1b2c3d4)
✓ Syncing: pdf-processing (f7e8d9c0)
2 skills synced
```

Useful after a fresh clone, or when `.claude/skills/` gets corrupted or deleted.

### Orchestration (Gas Town, etc.)

Skills require no special orchestration support. The entire interaction is **user → agent → `bd` binary**:

1. User installs a skill (`bd skills add`)
2. Agent reads metadata at startup (`bd prime`)
3. Agent loads full content when relevant (`bd skills load`)

Because every step goes through the `bd` CLI, any orchestrator that spawns agents — Gas Town rigs, CI pipelines, custom harnesses — gets skills support automatically. The orchestrator doesn't need to know about skills at all; it just runs agents that call `bd`. This is the same pattern beads uses for issues, dependencies, and molecules: the binary is the integration surface, not the orchestrator.

---

## Questions for Review

- [ ] **Skill size limits:** Warn at 10KB? Hard limit at 50KB? The spec recommends keeping SKILL.md under 500 lines.
- [ ] **Auto-sync on init:** Should `bd init` check for skill updates, or only install missing skills?
- [ ] **Skill composition:** Can skills reference other skills? The spec advises against deep reference chains.
- [ ] **Private repos:** Support company-internal skills via git credentials?
- [ ] **Multi-agent cache:** Should `bd skills sync` write to multiple agent directories (`.claude/`, `.codex/`, etc.) or just `.claude/`?
