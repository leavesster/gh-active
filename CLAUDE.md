# gh-active

CLI tool: GitHub user activity → deduplicated weekly report (Markdown), optionally summarized by LLM.
ls -
## Build & Test

```bash
go build ./cmd/gh-active/          # build binary
go test ./...                       # run all tests
go test -v ./internal/github/...    # run GitHub client tests with verbose output
go run ./cmd/gh-active/ report --no-llm                    # quick smoke test (uses authenticated user)
go run ./cmd/gh-active/ report --user=<username> --no-llm  # smoke test for specific user
```

## Project Structure

```
cmd/gh-active/main.go         CLI entry (cobra). Flags, config loading, orchestration.
cmd/gh-active/main_test.go    Tests for time range parsing (weekMonday, --week flag).
pkg/model/types.go            Core data: Activity, Commit, PRStatus, WeeklyReport.
internal/github/client.go     GitHub API client. Event fetching, parsing, SHA dedup.
internal/github/client_test.go Unit tests for dedup and event parsing.
internal/llm/llm.go           LLM interface + prompt builder.
internal/llm/claude.go        Anthropic Claude backend.
internal/llm/openai.go        OpenAI backend.
internal/output/output.go     Output interface. Stdout/file writers for report content.
internal/report/markdown.go   Markdown report renderer (Completed / In Progress).
internal/config/config.go     YAML config + env var override + gh CLI auth fallback.
```

## Architecture

**Data flow:** `Events API → paginated fetch → Compare API (push commits) → SHA dedup → LLM summary → Markdown`

**Two-pass dedup in `ParseEvents()`:**
1. Pass 1: Process PullRequestEvents. For merged PRs, fetch commit SHAs via PR Commits API. Dedup same PR by URL — keep highest-priority status (merged > review > opened).
2. Pass 2: Process PushEvents. Use Compare API (`before...head`) to get commits. Filter out any commit SHA already seen in a PR.

**Key invariant:** A commit SHA appears in the report exactly once — either under its PR or as a standalone push commit, never both.

## Coding Conventions

- Go 1.24. Standard library preferred; minimal dependencies.
- `internal/` for non-exported packages, `pkg/` for shared types.
- **When adding or changing features, update README.md, README_zh.md, and CLAUDE.md accordingly.**
- No interface pollution — concrete types unless multiple implementations exist (LLM is the exception).
- Functions do one thing. No deep nesting. Keep it flat.
- Error handling: wrap with `fmt.Errorf("context: %w", err)`. No swallowed errors except intentional skips (e.g. Compare API failures on force push).
- Tests use `testing` stdlib only. No testify. Table-driven where appropriate.
- Test helpers (`makeEvent`, `makePRPayload`) live in `_test.go`, not exported.

## Key Design Decisions

- **Events API PushEvent has no commits array** (unlike webhook PushEvent). Only `ref`, `head`, `before` are available. We use Compare API to get actual commits.
- **PR status priority:** merged(3) > review(2) > opened(1). Same PR with multiple events in one week keeps the highest status. Performed-events may emit merged PRs as `action=merged` rather than `closed + merged=true`.
- **PR payloads may be partial:** `users/{username}/events` can omit PR title / HTML URL / merged bool. Parser should fall back to repo+number and fetch full PR details when needed.
- **GitHub auth chain:** `GITHUB_TOKEN` env → config file → `gh auth token` CLI fallback.
- **LLM `base_url`:** Both Claude and OpenAI support custom base URL for proxy/gateway setups. OpenAI-compatible backends select the API route with `llm.openai.mode`: `responses` for `/v1/responses`, `chat` for `/v1/chat/completions`.
- **LLM prompt output style:** Summary should be grouped by repository, and each input event should map to a single concise sentence whenever possible.
- **Report output:** `--output/-o` overrides `report.output` in config. Empty output means write to stdout. File output creates parent directories automatically.
- **`--week` flag:** `previous` for last completed week; or pass any date, `weekMonday()` computes the Monday 00:00 of that week. Time range priority: `--week` > `--start/--end` > default (current week, Mon~now).
- **Compare API failures are silent:** Force pushes or deleted branches cause 404 — these pushes are skipped, not errors.

## Common Pitfalls

- `go-github` `Event.ParsePayload()` returns `any` — always type-assert.
- `Event.GetRepo().GetName()` returns `"owner/repo"` format, needs `strings.SplitN(repo, "/", 2)` before API calls.
- Anthropic SDK `NewClient()` returns value type, not pointer.
- Events API is roughly newest-first, but older events may still appear interleaved in a page. `FetchEvents` requests 100 items per page, filters client-side across all available pages, and reports when GitHub's 10-page cap truncates the requested range.
