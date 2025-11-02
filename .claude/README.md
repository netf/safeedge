# Claude Code Configuration for SafeEdge

This directory contains Claude Code configuration following best practices from [How I Use Every Claude Code Feature](https://blog.sshh.io/p/how-i-use-every-claude-code-feature).

## Directory Structure

```
.claude/
├── README.md                    # This file
├── settings.local.json          # Local settings (gitignored)
├── commands/                    # Custom slash commands
│   ├── catchup.md              # /catchup - Read changed files in branch
│   ├── pr.md                   # /pr - Prepare pull request
│   ├── test-e2e.md             # /test-e2e - Run E2E tests
│   └── dbgen.md                # /dbgen - Regenerate sqlc code
└── hooks/                       # Automation hooks
    └── pre-commit.sh           # Pre-commit validation (non-blocking)
```

## Custom Slash Commands

### `/catchup`
Reads all files changed in the current git branch to catch up on recent work.

**Usage:** `/catchup`

### `/pr`
Prepares a pull request by running formatting, linting, tests, and staging changes.

**Usage:** `/pr`

**What it does:**
- Runs `go fmt ./...`
- Runs `go vet ./...`
- Regenerates sqlc if database queries changed
- Runs tests
- Prompts for commit message
- Creates commit

### `/test-e2e`
Runs the E2E test suite with full Docker infrastructure.

**Usage:** `/test-e2e`

**What it does:**
- Starts Docker Compose test infrastructure
- Runs Playwright E2E tests
- Provides detailed failure reports

### `/dbgen`
Regenerates sqlc database code after schema or query changes.

**Usage:** `/dbgen`

**What it does:**
- Navigates to `internal/controlplane/database`
- Runs `sqlc generate`
- Reports generated files

## Hooks

### Pre-Commit Hook (UserPromptSubmit)

Runs **non-blocking validation** before commits. Following the blog's guidance, this hook provides **hints only** and never blocks Claude from proceeding.

**Checks:**
- ✓ Go code formatting (`gofmt`)
- ✓ sqlc regeneration needed
- ✓ Potential secrets in staged files
- ✓ .env files not staged

**Exit Behavior:** Always exits with code 0 (success) to avoid blocking Claude mid-plan.

## Settings Configuration

### Timeouts (`.claude/settings.local.json`)

```json
{
  "env": {
    "BASH_MAX_TIMEOUT_MS": "300000",  // 5 minutes for long tests
    "MCP_TOOL_TIMEOUT": "120000"       // 2 minutes for MCP
  }
}
```

### Permissions

Pre-approved permissions for common operations:
- `WebFetch(domain:github.com)` - Fetch GitHub documentation
- `WebFetch(domain:blog.sshh.io)` - Fetch Claude Code blog posts
- `Bash(chmod:*)` - Make scripts executable

## Philosophy

This configuration follows key principles from the blog:

1. **Minimal Magic Commands** - Only 4 essential slash commands
2. **Non-Blocking Hints** - Hooks never block Claude's planning
3. **Token Efficiency** - CLAUDE.md is concise (~8KB), references external docs
4. **Guardrails First** - Document what Claude gets wrong, not comprehensive manuals
5. **Master-Clone Architecture** - Let Claude decide when to delegate, don't force workflows

## Development Workflow

**Typical session:**
1. Start: `claude` (in project root)
2. Catch up: `/catchup` (if rejoining after time away)
3. Work on features
4. Prepare PR: `/pr`
5. Run tests: `/test-e2e`

**Database changes:**
1. Edit `internal/controlplane/database/schema.sql`
2. Edit `internal/controlplane/database/queries/*.sql`
3. Run: `/dbgen`
4. Service layer automatically uses new generated types

## Updating Configuration

### Add a new slash command:
```bash
# Create new command file
cat > .claude/commands/mycommand.md <<EOF
---
description: Brief description of command
---

Command instructions here...
EOF
```

### Modify hook behavior:
Edit `.claude/hooks/pre-commit.sh` - remember to keep it non-blocking!

### Add new permissions:
Edit `.claude/settings.local.json` permissions section.

## Best Practices

**DO:**
- Keep CLAUDE.md concise and focused on gotchas
- Use slash commands for repetitive multi-step tasks
- Let hooks provide hints, not block progress
- Update this README when adding new commands

**DON'T:**
- Force engineers to learn complex magic commands
- Block Claude mid-plan with hooks
- Embed full documentation in CLAUDE.md (bloats context)
- Create specialized subagents with gatekept context

## References

- [How I Use Every Claude Code Feature](https://blog.sshh.io/p/how-i-use-every-claude-code-feature)
- [Claude Code Documentation](https://docs.claude.com/en/docs/claude-code)
