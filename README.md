# gh-active

[中文文档](README_zh.md)

CLI tool that generates weekly reports from GitHub user activity.

Fetches Events API data, automatically deduplicates commits (removes duplicates caused by PR merges), and uses LLM to generate structured Markdown weekly reports for your team or manager.

## Install

```bash
go install github.com/leavesster/gh-active/cmd/gh-active@latest
```

Or build from source:

```bash
git clone https://github.com/leavesster/gh-active.git
cd gh-active
go build -o gh-active ./cmd/gh-active/
```

Or run directly from source (no build needed):

```bash
go run ./cmd/gh-active/ report --user=torvalds --no-llm
```

## Quick Start

```bash
# Zero config if you're already logged in with gh CLI
# Omit --user to generate a report for yourself
gh-active report --no-llm

# Or specify a different user
gh-active report --user=torvalds --no-llm

# Or specify a token manually
export GITHUB_TOKEN=ghp_xxxxx
gh-active report --user=torvalds --no-llm

# Generate summary with Claude
export ANTHROPIC_API_KEY=sk-ant-xxxxx
gh-active report --user=torvalds --llm=claude

# Specify time range and output to file
gh-active report --user=torvalds --start=2026-02-10 --end=2026-02-16 -o report.md

# Or configure a default output path once
gh-active init
# then set report.output in ~/.gh-active.yaml
gh-active report --week=previous

# Auto-calculate Mon~Sun from any date in that week
gh-active report --user=torvalds --week=2026-02-12

# Query last week (previous completed Mon~Sun)
gh-active report --week=previous
```

## Usage

```
gh-active report [flags]

Flags:
      --user string     GitHub username (default: authenticated user)
      --week string     "previous" for last week, or any date (YYYY-MM-DD) to pick that week
      --start string    Start date (YYYY-MM-DD), default: this Monday
      --end string      End date (YYYY-MM-DD), default: now
      --llm string      LLM backend (claude, openai)
      --no-llm          Skip LLM, output structured data directly
  -o, --output string   Output file path
```

```
gh-active init          Create default config file at ~/.gh-active.yaml
```

## Configuration

Run `gh-active init` to generate a config file, or create `~/.gh-active.yaml` manually:

```yaml
github:
  token: ""           # or set GITHUB_TOKEN env var

llm:
  default: claude
  claude:
    api_key: ""       # or set ANTHROPIC_API_KEY env var
    model: claude-sonnet-4-5-20250929
    base_url: ""      # custom API endpoint (e.g. proxy)
  openai:
    api_key: ""       # or set OPENAI_API_KEY env var
    model: gpt-4o     # or set OPENAI_MODEL env var
    base_url: ""      # custom API endpoint (e.g. proxy)
    mode: responses   # responses or chat; can also be OPENAI_API_MODE

report:
  language: zh-CN
  output: ""          # default stdout; can be overridden by --output
```

Environment variables take priority over the config file. OpenAI mode can be set to `responses` for `/v1/responses` or `chat` for `/v1/chat/completions`.

### GitHub Authentication

GitHub token is resolved in the following order — **no duplicate setup needed**:

1. `GITHUB_TOKEN` environment variable
2. `github.token` in `~/.gh-active.yaml`
3. `gh auth token` (automatically reuses gh CLI login)

If you've already run `gh auth login`, it works out of the box with zero configuration.

## How It Works

```
GitHub Events API → Paginated fetch → Compare API for commits → SHA dedup → LLM summary → Markdown
```

1. Fetch all PushEvent and PullRequestEvent within the time range via Events API
2. For each PushEvent, use Compare API (`before...head`) to get the actual commit list
3. For merged PRs, fetch their commit SHA list
4. Automatically deduplicate using SHA set: commits in Push that belong to a PR are filtered out
5. Feed deduplicated activities to LLM with prompt constraints: group by repository and keep near one sentence per event. OpenAI-compatible backends can use either the Responses API (`/v1/responses`) or Chat Completions (`/v1/chat/completions`) via `llm.openai.mode`.
6. Output a Markdown report with "Completed" and "In Progress" sections

## Tracked Activity Types

| Event | Status | Description |
|-------|--------|-------------|
| PR opened | In Progress | PRs opened this week |
| PR review_requested | In Progress | PRs with review requested |
| PR merged / closed+merged | Completed | Merged PRs (participates in dedup) |
| PR closed (not merged) | Ignored | — |
| Push | Completed | Standalone commits after dedup |

## Limitations

- GitHub limits this Events API resource to 10 pages; the tool requests 100 items per page and warns if it hits that cap before reaching the requested start date
- Performed-events PR payloads may be partial; the tool falls back to `owner/repo#number` and may fetch full PR details when title or HTML URL is missing
- Compare API may fail for force pushes or deleted branches — these pushes are silently skipped

## License

MIT
