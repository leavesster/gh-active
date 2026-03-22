# gh-active

[English](README.md)

从 GitHub 用户活动生成周报的 CLI 工具。

拉取 Events API 数据，自动去重（PR merge 产生的重复 commits），用 LLM 生成面向团队/老板的结构化 Markdown 周报。

## 安装

```bash
go install github.com/leavesster/gh-active/cmd/gh-active@latest
```

或从源码构建：

```bash
git clone https://github.com/leavesster/gh-active.git
cd gh-active
go build -o gh-active ./cmd/gh-active/
```

或直接从源码运行（无需构建）：

```bash
go run ./cmd/gh-active/ report --user=torvalds --no-llm
```

## 快速开始

```bash
# 如果你已经用 gh CLI 登录过，直接用，零配置
# 不传 --user 默认查自己的活动
gh-active report --no-llm

# 或者查看其他用户
gh-active report --user=torvalds --no-llm

# 或者手动指定 token
export GITHUB_TOKEN=ghp_xxxxx
gh-active report --user=torvalds --no-llm

# 用 Claude 生成摘要
export ANTHROPIC_API_KEY=sk-ant-xxxxx
gh-active report --user=torvalds --llm=claude

# 指定时间范围，输出到文件
gh-active report --user=torvalds --start=2026-02-10 --end=2026-02-16 -o report.md

# 或者在配置里设置默认输出路径
gh-active init
# 然后在 ~/.gh-active.yaml 里设置 report.output
gh-active report --week=previous

# 传入任意日期，自动计算所在周的周一~周日
gh-active report --user=torvalds --week=2026-02-12

# 查上周（上一个完整周一~周日）
gh-active report --week=previous
```

## 用法

```
gh-active report [flags]

Flags:
      --user string     GitHub 用户名（默认：当前认证用户）
      --week string     "previous" 查上周，或任意日期 (YYYY-MM-DD) 查所在周
      --start string    起始日期 (YYYY-MM-DD)，默认本周一
      --end string      结束日期 (YYYY-MM-DD)，默认当前时间
      --llm string      LLM 后端 (claude, openai)
      --no-llm          跳过 LLM，直接输出结构化数据
  -o, --output string   输出文件路径
```

```
gh-active init          创建默认配置文件 ~/.gh-active.yaml
```

## 配置

运行 `gh-active init` 生成配置文件，或手动创建 `~/.gh-active.yaml`：

```yaml
github:
  token: ""           # 或设置 GITHUB_TOKEN 环境变量

llm:
  default: claude
  claude:
    api_key: ""       # 或设置 ANTHROPIC_API_KEY 环境变量
    model: claude-sonnet-4-5-20250929
    base_url: ""      # 自定义 API 地址（如代理）
  openai:
    api_key: ""       # 或设置 OPENAI_API_KEY 环境变量
    model: gpt-4o
    base_url: ""      # 自定义 API 地址（如代理）

report:
  language: zh-CN
  output: ""          # 默认输出到 stdout，可被 --output 覆盖
```

环境变量优先级高于配置文件。

### GitHub 鉴权

按以下优先级获取 GitHub Token，**无需重复配置**：

1. `GITHUB_TOKEN` 环境变量
2. `~/.gh-active.yaml` 中的 `github.token`
3. `gh auth token`（自动复用 gh CLI 的登录状态）

已经 `gh auth login` 过的用户开箱即用，零配置。

## 工作原理

```
GitHub Events API → 分页拉取 → Compare API 获取 commits → SHA 去重 → LLM 摘要 → Markdown
```

1. 通过 Events API 拉取用户在时间范围内的所有 PushEvent 和 PullRequestEvent
2. 对每个 PushEvent，用 Compare API (`before...head`) 获取实际 commit 列表
3. 对已合并的 PR，获取其 commit SHA 列表
4. 用 SHA 集合自动去重：Push 中属于 PR 的 commits 被过滤掉
5. 将去重后的活动喂给 LLM，并通过 prompt 约束为按仓库分组、尽量一条 event 一句话
6. 输出分"已完成"和"进行中"两个板块的 Markdown 周报

## 跟踪的活动类型

| 事件 | 状态 | 说明 |
|------|------|------|
| PR opened | 进行中 | 本周新开的 PR |
| PR review_requested | 进行中 | 请求 review 的 PR |
| PR merged / closed+merged | 已完成 | 已合并的 PR（参与去重） |
| PR closed (未合并) | 忽略 | — |
| Push | 已完成 | 去重后的独立 commits |

## 限制

- GitHub 对这个 Events API 资源限制最多 10 页；工具会按每页 100 条拉取，并在尚未到达起始时间就撞到页上限时给出告警
- performed-events 里的 PR payload 可能是缩略版；当缺少标题或 HTML 链接时，工具会回退到 `owner/repo#number`，必要时补拉完整 PR 信息
- Compare API 对 force push 或已删除的分支可能失败，这些 push 会被静默跳过

## License

MIT
