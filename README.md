# ahv — Agent History Viewer

A terminal UI and HTML report generator for browsing, searching, and
summarizing chat history from AI agent CLIs.

This project merges two tools: the multi-agent TUI/search engine of
[agent-chat-viewer](https://github.com/manno/agent-chat-viewer) with the
grouped, AI-summarized HTML report of
[claude-history-viewer](https://github.com/mbologna/claude-history-viewer).

## Supported agents

- **Claude Code** — `~/.claude/projects/`
- **Gemini CLI** — `~/.gemini/tmp/`
- **GitHub Copilot CLI** — `~/.copilot/session-state/`
- **Google Antigravity CLI (`agy`)** — `~/.gemini/antigravity-cli/brain/`

## Installation

Requires Go 1.21+.

```bash
git clone https://github.com/mbologna/agent-history-viewer
cd agent-history-viewer
./build.sh
mv ahv /usr/local/bin/
```

## Usage

### TUI (default)

```bash
ahv

# Jump straight into a conversation by session ID (a prefix works too)
ahv 8d2c6cc4-8a5d-4dd8-984d-93e8f78e7786
```

| Key | Action |
|-----|--------|
| `↑↓` / `jk` | navigate |
| `Enter` | open session |
| `/` | filter list by project or agent |
| `s` | search message content |
| `r` | reload |
| `q` | quit / go back |
| **In detail view** | |
| `u` / `a` / `0` | show user / assistant / all messages |
| `g` / `G` | jump to top / bottom |
| `PgUp` / `PgDn` | page scroll |

### CLI mode

```bash
# List sessions
ahv -no-tui

# List with start times
ahv -no-tui -s

# View session by index
ahv 0

# Filter by role
ahv 0 user
ahv 0 assistant

# Search across all sessions
ahv -f "*FreeCAD*"
```

See `ahv -h` for the full flag list, including `-agent`, `-project`,
`-since`/`-until`, `-regex`, `-phrase`, `-limit`, and `-json`.

### HTML report

Export a static, browsable report grouped by project → session → prompt,
across every agent at once:

```bash
ahv -html
open ~/.config/agent-history-viewer/report.html
```

Related projects are grouped together automatically (e.g.
`myapp-frontend` and `myapp-backend` both fall under `myapp`). The
grouping is saved to `~/.config/agent-history-viewer/groups.json` on
first run — edit it to merge or split groups by hand, or pass
`-reset-groups` to regenerate it from scratch. Trivial prompts
("yes", "continue", slash commands, pasted-content placeholders) are
filtered out of the report.

Useful flags with `-html`:

- `-output PATH` — where to write the report (default:
  `~/.config/agent-history-viewer/report.html`)
- `-project NAME` — only include groups matching NAME (substring)
- `-agent NAME` — restrict to one agent
- `-serve` — start a local server so the report's "Summarize" buttons
  work, generating a 1-2 sentence AI summary of a session or project on
  demand (requires the `claude` CLI on `PATH`; summaries are cached in
  `~/.config/agent-history-viewer/summary_cache.json`)
- `-port N` — port for `-serve` (default: auto)

## Agent Skills

The `skills/` directory contains reusable agent skill definitions (e.g. for
GitHub Copilot CLI).

**Install a skill:**

```bash
# 1. Copy the skill folder into the canonical store
cp -r skills/search-prior-sessions ~/.config/agent-skills/

# 2. Sync it into each agent's skill directory (creates symlinks)
ahv -sync-skills
```

`ahv -sync-skills` moves any per-agent skill directories into
`~/.config/agent-skills/` and symlinks them back so all agents share the
same canonical copy. You can also run the sync interactively from the
`ahv` TUI (Skills tab → `s`).

- **[search-prior-sessions](skills/search-prior-sessions/SKILL.md)** — skill that teaches agents to use `ahv` for recalling prior work across Copilot, Claude, Gemini, and agy sessions.

## License

MIT
