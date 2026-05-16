# Silo

**Markdown knowledge layer for Engram and Obsidian.** Silo turns persistent agent memory into human-readable notes without becoming a second memory database.

Silo is a bridge: **Engram stores memory**, **Silo renders and maintains Markdown**, and **Obsidian is the human interface** for reading, editing, and organizing that knowledge.

## Quick path

```bash
# Install
go install github.com/Nick-2455/silo/cmd/silo@latest

# Run the TUI
silo

# Or run as MCP server (for AI agents to query your knowledge)
silo --server
```

**[Engram](https://github.com/Gentleman-Programming/engram)** is required as the persistent memory backend. Install it first:

```bash
brew install gentleman-programming/tap/engram
```

## What it does

| You do | Silo does |
|--------|-----------|
| Capture knowledge with agents | Engram stores it as persistent memory |
| Ask for knowledge context | Silo reads Engram and Markdown notes |
| Maintain an Obsidian vault | Silo creates, updates, and searches Markdown notes |
| Use MCP clients | Silo exposes small tools for knowledge lookup and note maintenance |
| Keep legacy graph data | Existing TUI/MCP graph flows remain available during the transition |

Silo is not the memory source. It is the readable knowledge layer on top of Engram.

## Note model

Silo uses a community-facing note vocabulary. Every note has a **type** and an optional **kind**:

| Type | Description | Kinds |
|------|-------------|-------|
| `concept` | A discrete idea, term, or principle | — |
| `resource` | An external resource to read or watch | `article`, `book`, `video`, `tool`, `paper`, `course` |
| `roadmap` | A staged learning or project plan | — |
| `collection` | A named grouping of related notes | `subject`, `project`, `interest`, `career-path`, `research`, `team-space` |

When you call `create_or_update_note` with a `type`, Silo injects the appropriate frontmatter defaults automatically. Use `list_note_templates` to browse available templates and `get_note_template` to fetch the full Markdown body.

```yaml
# Example: a resource note
type: resource
kind: book
title: "my-book-title"
tags: []
url: ""
status: unread
created: 2024-01-01
```

Templates are generic — no personal taxonomy is baked in. You define what a `collection` means for your workflow.

## MVP scope

Silo's MVP scope is intentionally small.

### Silo does

- Read knowledge from Engram.
- Create and update Markdown notes in an Obsidian vault.
- Search Markdown files in the vault.
- Build concise knowledge context from Engram and vault notes.
- Expose simple MCP tools for agents to query and maintain knowledge.
- Keep existing legacy graph/TUI flows working while the new bridge is introduced.

### Silo does not do

- Replace Engram as persistent agent memory.
- Maintain a second structured memory database as source of truth.
- Provide perfect bidirectional sync between Engram and Obsidian.
- Resolve complex conflicts between edited notes and Engram observations.
- Rebuild every legacy graph feature in the first MVP slice.

## Screens

| Key | Screen | What you see |
|-----|--------|--------------|
| — | Dashboard | Active projects, resource buckets (inbox/active/later/archived) |
| `a` | Add | Save a resource with URL and title |
| `g` | Domains | Hierarchical tree — Domain → Subareas |
| `p` | Projects | Active and inactive projects, Enter for detail |
| `s` | Sessions | Work sessions grouped by project |
| `l` | Learnings | Everything you've extracted from sessions |
| `o` | Sync | Export graph to Obsidian (asks for vault path on first run) |
| `t` | Triage | Move resources between inbox/active/later/archived |
| `c` | Config | Read-only view of current configuration |

Navigation: `↑/↓` or `j/k`, `Enter` to select, `Esc` to go back, `q` to quit.

## Legacy knowledge graph

Silo still includes a legacy graph model for compatibility. New MVP work should treat it as a legacy projection, not the source of truth.

Every piece of knowledge is a **node**. Connections are **edges**.

```
Domain ──contains──▶ Subarea
Project ──applies_to──▶ Subarea
Session ──worked_on──▶ Project
Learning ──learned_from──▶ Session
Learning ──applies_to──▶ Subarea
Learning ──applies_to──▶ Project
Resource ──references──▶ Subarea
```

A single learning can connect to **multiple** subareas and projects. You debugged an MCP client bug in the `silo` project and learned something that applies to both Backend and iOS.

## Legacy ALM cycle

The ALM cycle below describes the original product direction. The MVP bridge direction is simpler: Engram persists memory, Silo writes Markdown, and Obsidian presents it to humans.

Silo moves knowledge through four phases:

| Phase | What happens | Agent model |
|-------|-------------|-------------|
| **Curation** | Resources enter the system, get pre-tagged | Fast model (GPT-4o mini) |
| **Strategic** | Resources matched to roadmap, prioritized | Reasoning model (Claude Sonnet) |
| **Synthesis** | Active resources distilled into atomic learnings | Synthesis model (Gemini Pro) |
| **Execution** | Track sessions, log what was built/learned | Optimized model (OpenCode) |

The cycle is per-resource, not a linear pipeline. Resources flow through as you engage with them.

## MCP tools

When running `silo --server`, these tools are available to AI agents:

> The current server still includes legacy graph tools. The MVP bridge tools are being introduced in slices.

| Tool | What it does |
|------|-------------|
| `search` | Search resources in your knowledge base |
| `add_resource` | Save a new resource |
| `get_roadmap` | View your current learning roadmap |
| `triage` | Move resources between buckets |
| `list_domains` | Browse your domain taxonomy |
| `list_projects` | List all projects with subarea links |
| `create_domain` | Create a new knowledge domain |
| `create_subarea` | Add a subarea under a domain |
| `create_project` | Start tracking a new project |
| `link_project` | Connect a project to a subarea |
| `toggle_project` | Mark a project active or inactive |
| `create_session` | Log a work session |
| `create_learning` | Extract a learning from a session |
| `list_sessions` | Browse work sessions |
| `list_learnings` | Browse extracted learnings |
| `link_resource` | Tag a resource with a subarea |
| `list_person` | View your profile node |
| `sync_obsidian` | Export the legacy graph to Obsidian |

MVP bridge tools (available now):

| Tool | What it does |
|------|-------------|
| `read_from_engram` | Read knowledge items from Engram |
| `sync_to_obsidian` | Write Engram knowledge into Markdown notes |
| `search_vault` | Search Markdown notes in the Obsidian vault |
| `create_or_update_note` | Create or update one Markdown note (supports `type` and `kind`) |
| `get_knowledge_context` | Combine Engram and vault results into agent context |
| `list_note_templates` | List community note types with metadata and frontmatter schema |
| `get_note_template` | Fetch the full Markdown body for a specific note type |

## Obsidian sync

Press `o` in the TUI. First run asks for your vault path — type it once, it's saved. Every sync after is one keystroke.

Your graph appears under `Silo/` in your vault:

```
Silo/
  Persona.md
  Domains/Engineering.md
  Subareas/Backend.md
  Projects/my-project.md
  Sessions/my-session.md
  Learnings/my-learning.md
```

Files use YAML frontmatter and `[[wikilinks]]` — open Obsidian's graph view and you'll see your knowledge as a connected web.

The MVP bridge writes simpler notes under:

```txt
Silo/
  Knowledge/
    <note>.md
```

Those notes are Markdown views over Engram knowledge, not a replacement for Engram memory.

## Configuration

`~/.config/silo/config.yaml`:

```yaml
profile: default
engram_path: engram
obsidian_vault_path: /path/to/your/vault  # set via TUI, not manually
```

The vault path is set interactively — press `o`, type the path, press Enter. No YAML editing needed.

## Architecture

```
Engram ──▶ Silo knowledge layer ──▶ Obsidian Markdown vault
  ▲                ▲
  │                │
Agents ◀──────── MCP tools

Legacy TUI/MCP ──▶ GraphStore (SQLite projection) ──▶ legacy Obsidian sync
                  └──────────────▶ Engram
```

- **Engram**: persistent agent memory and semantic search source.
- **Silo knowledge layer**: reads Engram, writes/searches Markdown, and builds context.
- **Obsidian**: human reading, editing, and organization interface.
- **SQLite graph store**: legacy local projection/cache, not the source of truth for new MVP flows.

The design goal is boring on purpose: small MCP tools, Markdown files, and no duplicated memory system.

## Contributing

Silo is a community product. The goal is a knowledge graph that works for anyone — not one person's specific taxonomy.

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT
