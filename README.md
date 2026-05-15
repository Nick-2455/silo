# Silo

**Second brain for knowledge work.** Track what you learn, connect it to your projects, and build a living map of everything you know — not just a library of bookmarks.

Silo is a knowledge graph orchestrator: a TUI app + MCP server that helps you organize resources, track projects, extract learnings from work sessions, and sync your knowledge to Obsidian.

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
| Save a resource (URL, PDF, video) | Tags it by domain, places it in your roadmap |
| Define your domains (Dev, Philosophy, Cinema) | Builds a taxonomy hierarchy (Domain → Subareas) |
| Track active projects | Links projects to subareas, shows what you're working on |
| Log a work session | Connects it to projects, extracts what you learned |
| Review your learnings | Shows everything you've learned, filterable by domain |
| Press `o` | Syncs your entire knowledge graph to Obsidian with wikilinks |

Silo is the orchestrator — **you** decide what matters, Silo organizes it and keeps agents informed.

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

## Knowledge graph

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

## ALM cycle

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
| `sync_obsidian` | Export the graph to Obsidian |

## Obsidian sync

Press `o` in the TUI. First run asks for your vault path — type it once, it's saved. Every sync after is one keystroke.

Your graph appears under `Silo/` in your vault:

```
Silo/
  Persona.md
  Domains/Dev.md
  Subareas/Backend.md
  Projects/silo.md
  Sessions/Debug de Engram MCP client.md
  Learnings/mem_update reemplaza contenido entero.md
```

Files use YAML frontmatter and `[[wikilinks]]` — open Obsidian's graph view and you'll see your knowledge as a connected web.

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
TUI (Bubble Tea) ──▶ GraphStore (SQLite topology) 
                   ──▶ Engram (MCP, durable content)
                   
MCP Server ──▶ Same backend, exposes tools to agents

Obsidian ◀── Syncer exports GraphStore as .md files
```

- **SQLite**: fast graph queries (neighbors, paths, active projects). Local cache of graph topology.
- **Engram**: durable content storage with semantic search. Agents query it via `mem_search`.
- **Syncer**: one-directional export to Obsidian for human browsing and graph visualization.

## Contributing

Silo is a community product. The goal is a knowledge graph that works for anyone — not one person's specific taxonomy.

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT