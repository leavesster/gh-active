# gh-active

从 GitHub 用户活动生成周报的 CLI 工具。

拉取 Events API 数据，自动去重（PR merge 产生的重复 commits），用 LLM 生成面向团队/老板的结构化 Markdown 周报。

## 安装

```bash
go install github.com/yleaf/gh-active/cmd/gh-active@latest
```

或从源码构建：

```bash
git clone https://github.com/yleaf/gh-active.git
cd gh-active
go build -o gh-active ./cmd/gh-active/
```

## 快速开始

```bash
# 如果你已经用 gh CLI 登录过，直接用，零配置
gh-active report --user=torvalds --no-llm

# 或者手动指定 token
export GITHUB_TOKEN=ghp_xxxxx
gh-active report --user=torvalds --no-llm

# 用 Claude 生成摘要
export ANTHROPIC_API_KEY=sk-ant-xxxxx
gh-active report --user=torvalds --llm=claude

# 指定时间范围，输出到文件
gh-active report --user=torvalds --start=2026-02-10 --end=2026-02-16 -o report.md
```

## 用法

```
gh-active report [flags]

Flags:
      --user string     GitHub 用户名（必填）
      --start string    起始日期 (YYYY-MM-DD)，默认上周一
      --end string      结束日期 (YYYY-MM-DD)，默认上周日
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
  openai:
    api_key: ""       # 或设置 OPENAI_API_KEY 环境变量
    model: gpt-4o

report:
  language: zh-CN
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
5. 将去重后的活动喂给 LLM 生成摘要
6. 输出分"已完成"和"进行中"两个板块的 Markdown 周报

## 跟踪的活动类型

| 事件 | 状态 | 说明 |
|------|------|------|
| PR opened | 进行中 | 本周新开的 PR |
| PR review_requested | 进行中 | 请求 review 的 PR |
| PR merged | 已完成 | 已合并的 PR（参与去重） |
| PR closed (未合并) | 忽略 | — |
| Push | 已完成 | 去重后的独立 commits |

## 限制

- Events API 最多返回 300 个事件（30 天内），对周报场景足够
- Compare API 对 force push 或已删除的分支可能失败，这些 push 会被静默跳过

## License

MIT
