# PD-Hunter 千星项目升级计划

> **目标**: 将 PD-Hunter 从个人工具升级为 GitHub 1000+ Star 级别的开源项目，使其成为简历上的亮点。
>
> **当前状态**: 3 个 Go/Python 脚本 + 1 个静态 HTML 仪表盘，仅追踪 3 个组织的赏金 issue。
>
> **执行者**: Claude Opus 4.6 Thinking

---

## 一、项目定位重塑

### 1.1 从"个人工具"到"平台级产品"

当前 PD-Hunter 是一个仅供个人使用的赏金追踪工具。要达到千星级别，需要重新定位为：

**"GitHub 开源赏金的一站式发现与智能分析平台"**

- **新名称建议**: `BountyRadar` 或保留 `PD-Hunter` 但加副标题 "Open Source Bounty Intelligence Platform"
- **核心卖点**: 帮助任何开发者在 5 秒内找到最适合自己技能的高价值赏金 issue
- **目标用户**: 全球开源贡献者、赏金猎人、想赚外快的开发者

### 1.2 差异化优势（为什么别人要 Star 这个项目）

1. **AI 驱动的难度评估** — 不只是列表，而是告诉你"这个 issue 你能不能做"
2. **竞争态势分析** — PR 数量、评论热度、活跃度一目了然
3. **个性化推荐** — 根据你的技术栈推荐合适的赏金
4. **实时数据** — 每 6 小时自动更新，不会错过新赏金

---

## 二、架构升级（最高优先级）

### 2.1 前端：从单文件 HTML 到 React/Next.js SPA

**现状问题**:
- `dashboard.html` 是 501 行的单文件，无法扩展
- 无路由、无状态管理、无组件化
- 依赖 CDN Tailwind，无法自定义构建

**升级方案**:

```
frontend/
├── src/
│   ├── app/                    # Next.js App Router
│   │   ├── page.tsx            # 首页/仪表盘
│   │   ├── bounty/[id]/        # 赏金详情页
│   │   ├── explore/            # 按组织/语言/标签浏览
│   │   ├── trending/           # 趋势赏金
│   │   └── layout.tsx
│   ├── components/
│   │   ├── ui/                 # shadcn/ui 基础组件
│   │   ├── BountyCard.tsx      # 赏金卡片（重构现有卡片）
│   │   ├── FilterBar.tsx       # 筛选栏
│   │   ├── StatsPanel.tsx      # 统计面板
│   │   ├── SearchBox.tsx       # 全文搜索
│   │   ├── TrendChart.tsx      # 趋势图表 (recharts)
│   │   └── SkillMatcher.tsx    # 技能匹配组件
│   ├── lib/
│   │   ├── api.ts              # 数据加载
│   │   └── utils.ts
│   └── hooks/
│       ├── useBounties.ts
│       └── useFilters.ts
├── public/
│   └── data/                   # 静态 JSON 数据
├── package.json
├── tailwind.config.ts
└── next.config.js
```

**技术选型**:
- **Next.js 14** (App Router + SSG) — 静态导出部署到 GitHub Pages
- **shadcn/ui** — 专业级 UI 组件
- **Recharts** — 数据可视化
- **Lucide Icons** — 现代图标
- **Framer Motion** — 动画

**关键页面**:

| 页面 | 功能 | 优先级 |
|------|------|--------|
| `/` | 仪表盘首页，统计概览 + 精选赏金 | P0 |
| `/explore` | 按组织/语言/标签/金额浏览 | P0 |
| `/bounty/[id]` | 赏金详情 + AI 分析 + 竞争态势 | P1 |
| `/trending` | 赏金趋势 + 新增/已关闭统计 | P1 |
| `/submit` | 用户提交新的组织/仓库追踪请求 | P2 |

### 2.2 后端：从脚本到可扩展的 CLI + Library

**现状问题**:
- Go 脚本没有 `go.mod`，不是正规的 Go 项目
- Python 脚本硬编码了文件路径
- 无测试、无错误恢复、无并发

**升级方案**:

```
cmd/
├── hunter/                     # CLI 入口
│   └── main.go
pkg/
├── scraper/                    # 抓取层
│   ├── github.go               # GitHub API 客户端
│   ├── github_test.go
│   ├── config.go               # 配置加载
│   └── types.go                # 共享类型
├── enricher/                   # AI 分析层
│   ├── openai.go               # OpenAI/GitHub Models 客户端
│   ├── analyzer.go             # issue 分析逻辑
│   └── analyzer_test.go
├── scorer/                     # 评分算法
│   ├── bounty_score.go         # 综合评分模型
│   └── bounty_score_test.go
└── exporter/                   # 输出层
    ├── json.go
    ├── markdown.go             # 生成 Markdown 报告
    └── rss.go                  # RSS feed 输出
scripts/
├── enrich.py                   # 保留 Python 版本作为备选
└── migrate.py                  # 数据迁移工具
go.mod
go.sum
```

**关键改进**:

1. **正规 Go Module**: 添加 `go.mod`，使项目可被 `go install` 安装
2. **并发抓取**: 使用 goroutine + semaphore 并发抓取，速度提升 5-10x
3. **增量更新**: 只抓取上次更新后变化的 issue，减少 API 调用
4. **CLI 框架**: 使用 `cobra` 提供子命令（`hunter scan`, `hunter enrich`, `hunter serve`）
5. **可插拔 AI**: 支持 OpenAI / Claude / Gemini / 本地 LLM 作为分析后端

### 2.3 数据层升级

**现状问题**:
- 原始 JSON 文件直接作为数据库，无索引、无历史
- `enriched_bounties.json` 已达 88KB，随 issue 增长会膨胀

**升级方案**:

1. **SQLite 作为本地数据库**: 存储历史快照，支持趋势分析
2. **JSON 保留为导出格式**: 前端仍然消费 JSON，但由 SQLite 生成
3. **历史数据**: 每次更新保留快照，用于绘制趋势图
4. **数据模型**:

```sql
-- 核心表
CREATE TABLE bounties (
    id INTEGER PRIMARY KEY,
    github_number INTEGER,
    repository TEXT,
    title TEXT,
    url TEXT,
    bounty_amount INTEGER,
    bounty_tier TEXT,
    friction_level TEXT,
    technical_hint TEXT,
    is_hidden_gem BOOLEAN,
    open_pr_count INTEGER,
    comment_count INTEGER,
    created_at DATETIME,
    updated_at DATETIME,
    first_seen DATETIME,        -- 新增：首次发现时间
    last_checked DATETIME       -- 新增：最后检查时间
);

-- 历史快照（用于趋势分析）
CREATE TABLE bounty_snapshots (
    id INTEGER PRIMARY KEY,
    bounty_id INTEGER,
    snapshot_date DATE,
    open_pr_count INTEGER,
    comment_count INTEGER,
    FOREIGN KEY (bounty_id) REFERENCES bounties(id)
);

-- 组织配置
CREATE TABLE organizations (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE,
    labels TEXT,  -- JSON array
    note TEXT,
    enabled BOOLEAN DEFAULT TRUE
);
```

---

## 三、功能升级

### 3.1 P0 — 核心功能增强

#### 3.1.1 支持 50+ 组织追踪

当前仅追踪 3 个组织。千星项目需要覆盖主流赏金生态：

```json
{
  "organizations": [
    {"name": "projectdiscovery", "labels": ["💎 Bounty", "bounty"]},
    {"name": "onyx-dot-app", "labels": ["💎 Bounty", "Bounty"]},
    {"name": "commaai", "labels": ["bounty"]},
    {"name": "appwrite", "labels": ["bounty"]},
    {"name": "supabase", "labels": ["bounty"]},
    {"name": "cal-com", "labels": ["bounty", "💰 Bounty"]},
    {"name": "twentyhq", "labels": ["bounty"]},
    {"name": "formbricks", "labels": ["bounty"]},
    {"name": "documenso", "labels": ["bounty"]},
    {"name": "highlight", "labels": ["bounty"]},
    {"name": "trigger-dev", "labels": ["bounty"]},
    {"name": "algora-io", "labels": ["bounty", "💎 Bounty"]},
    {"name": "hoppscotch", "labels": ["bounty"]},
    {"name": "nhost", "labels": ["bounty"]},
    {"name": "zeropsio", "labels": ["bounty"]}
  ]
}
```

**同时支持**:
- Algora.io 赏金平台的 issue 抓取
- IssueHunt 平台的 issue 抓取
- BOSS.dev 平台的赏金数据

#### 3.1.2 综合评分模型 (Bounty Score)

当前只有简单的 S/A/B 分级。需要一个 0-100 的综合评分：

```
BountyScore = w1 * AmountScore + w2 * FeasibilityScore + w3 * CompetitionScore + w4 * FreshnessScore

AmountScore:      基于赏金金额的归一化分数 (0-100)
FeasibilityScore: AI 评估的可行性分数 (0-100)，考虑难度、所需技能
CompetitionScore: 基于 open PR 数量和评论活跃度的竞争分 (0-100)，竞争越少分越高
FreshnessScore:   基于 issue 发布时间的新鲜度 (0-100)，越新分越高
```

#### 3.1.3 技能匹配系统

用户可以输入自己的技术栈，系统自动推荐匹配的赏金：

- 从 issue 标签和正文中提取技术栈关键词（Go, Python, React, Docker 等）
- 与用户输入的技能集做匹配
- 在卡片上显示 "Match Score: 85%"

#### 3.1.4 全文搜索

- 前端实现基于 Fuse.js 的模糊搜索
- 搜索范围：标题、正文、仓库名、标签、技术提示

### 3.2 P1 — 增值功能

#### 3.2.1 赏金趋势分析

- 每日/每周新增赏金数量图表
- 各组织赏金总金额趋势
- 平均赏金金额变化
- 最活跃的仓库排行

#### 3.2.2 通知系统

- **GitHub Actions Bot**: 新 S-Tier 赏金时发送 GitHub Discussion 通知
- **RSS Feed**: 生成 RSS feed，用户可订阅
- **Webhook**: 支持配置 Discord/Slack webhook 推送新赏金
- **邮件摘要**: 每日/每周赏金摘要邮件（可选）

#### 3.2.3 赏金日历视图

- 以日历形式展示赏金的创建/关闭时间线
- 标记即将过期的赏金

#### 3.2.4 多语言 AI 分析

- 当前仅英文 prompt。支持 `--lang zh` 参数生成中文分析
- 中英双语仪表盘

### 3.3 P2 — 社区功能

#### 3.3.1 用户贡献机制

- 通过 PR 提交新的组织追踪配置（`mapping.json`）
- 通过 Issue 模板报告错误数据
- 社区维护的 Expert Hints

#### 3.3.2 排行榜

- 追踪已关闭的赏金 issue，统计各开发者的赏金收入
- "Top Bounty Hunters" 排行榜

---

## 四、工程质量升级

### 4.1 测试

```
tests/
├── go/
│   ├── scraper_test.go         # 单元测试：GitHub API mock
│   ├── scorer_test.go          # 单元测试：评分模型
│   └── integration_test.go     # 集成测试：端到端流程
├── python/
│   ├── test_enrich.py          # 单元测试：AI 分析
│   └── test_extract.py         # 单元测试：金额提取
└── e2e/
    └── playwright/             # E2E 测试：前端交互
        ├── dashboard.spec.ts
        └── filter.spec.ts
```

**目标覆盖率**: Go >= 80%, Python >= 70%

### 4.2 CI/CD 流水线

```yaml
# .github/workflows/ci.yml
# PR 检查：lint + test + build
# - Go: golangci-lint + go test
# - Python: ruff + pytest
# - Frontend: eslint + tsc + next build
# - E2E: Playwright

# .github/workflows/deploy.yml
# main 分支合并后自动部署到 GitHub Pages

# .github/workflows/update_bounties.yml (现有，需增强)
# - 添加数据校验步骤
# - 添加失败告警
# - 添加手动触发特定组织的选项
```

### 4.3 代码质量

- **Go**: 添加 `golangci-lint` 配置，使用 `errcheck`, `staticcheck`
- **Python**: 添加 `ruff` + `mypy` 类型检查
- **Frontend**: ESLint + Prettier + TypeScript strict mode
- **Pre-commit hooks**: husky + lint-staged

### 4.4 Docker 支持

```dockerfile
# 一键运行完整 pipeline
FROM golang:1.22-alpine AS go-builder
COPY . .
RUN go build -o /hunter ./cmd/hunter

FROM python:3.11-slim
COPY --from=go-builder /hunter /usr/local/bin/
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY . /app
WORKDIR /app
ENTRYPOINT ["hunter"]
```

```bash
# 一键运行
docker run -e GITHUB_TOKEN=xxx ghcr.io/fuzoe/pd-hunter scan --enrich --export
```

---

## 五、文档与社区运营

### 5.1 README 重写

千星项目的 README 必须包含：

1. **Hero Banner**: 设计一个带 logo 的项目 Banner 图
2. **Badges**: stars, forks, license, CI status, last update
3. **动图演示**: 录制一个 30 秒的 GIF 展示核心功能
4. **一句话说明**: "Find high-value open source bounties matched to your skills, powered by AI."
5. **截图**: 仪表盘截图 (light/dark mode)
6. **Quick Start**: 3 行命令跑起来
7. **Architecture Diagram**: Mermaid 流程图（现有，需更新）
8. **Contributing Guide**: 如何贡献
9. **Roadmap**: 公开的路线图

### 5.2 文档站

```
docs/
├── getting-started.md          # 快速开始
├── configuration.md            # 配置说明
├── adding-organizations.md     # 如何添加新组织
├── ai-models.md                # 支持的 AI 模型
├── architecture.md             # 架构设计
├── api-reference.md            # CLI 命令参考
├── faq.md                      # 常见问题
└── changelog.md                # 变更日志
```

### 5.3 社区文件

```
.github/
├── ISSUE_TEMPLATE/
│   ├── bug_report.md
│   ├── feature_request.md
│   └── add_organization.md     # 请求追踪新组织
├── PULL_REQUEST_TEMPLATE.md
├── CONTRIBUTING.md
├── CODE_OF_CONDUCT.md
├── SECURITY.md
└── FUNDING.yml                 # GitHub Sponsors
```

### 5.4 推广策略

1. **Reddit**: 发布到 r/opensource, r/github, r/webdev, r/programming
2. **Hacker News**: "Show HN: AI-powered bounty hunter for open source"
3. **Twitter/X**: 每周发布 "Bounty of the Week" 精选
4. **Dev.to / Hashnode**: 写一篇 "How I Built an AI-Powered Bounty Tracker" 教程
5. **Product Hunt**: 作为开发者工具发布
6. **中文社区**: V2EX、掘金、知乎发布中文介绍

---

## 六、实施路线图

### Phase 1: 基础重构（1-2 周）

- [ ] 初始化 Go Module (`go.mod`) + 重构为 `cmd/` + `pkg/` 结构
- [ ] 添加 Go 单元测试（覆盖率 >= 60%）
- [ ] 初始化 Next.js 前端项目
- [ ] 迁移现有 dashboard.html 的 UI 到 React 组件
- [ ] 添加 CI workflow (lint + test + build)
- [ ] 社区文件模板 (CONTRIBUTING, Issue templates, PR template)

### Phase 2: 功能扩展（2-3 周）

- [ ] 扩展 `mapping.json` 到 20+ 组织
- [ ] 实现综合评分模型 (BountyScore)
- [ ] 实现全文搜索 (Fuse.js)
- [ ] 添加赏金详情页 (`/bounty/[id]`)
- [ ] 添加趋势图表 (Recharts)
- [ ] Docker 支持
- [ ] Python 测试 (pytest)

### Phase 3: 增值功能（2-3 周）

- [ ] 技能匹配系统
- [ ] RSS Feed 输出
- [ ] Discord/Slack Webhook 通知
- [ ] 赏金日历视图
- [ ] 多 AI 模型支持 (Claude, Gemini)
- [ ] SQLite 数据层 + 历史快照
- [ ] E2E 测试 (Playwright)

### Phase 4: 打磨与推广（1-2 周）

- [ ] 设计 Logo + Hero Banner
- [ ] 录制演示 GIF
- [ ] 重写 README（中英双语）
- [ ] 编写文档站
- [ ] 发布到 Reddit / Hacker News / Product Hunt
- [ ] 配置 GitHub Sponsors

---

## 七、技术栈总览

| 层级 | 当前 | 升级后 |
|------|------|--------|
| **爬虫** | 单文件 Go 脚本 | Go Module + cobra CLI + 并发抓取 |
| **AI 分析** | Python + OpenAI | Python + 多模型支持 (OpenAI/Claude/Gemini) |
| **前端** | 单文件 HTML + CDN Tailwind | Next.js 14 + shadcn/ui + Recharts |
| **数据** | JSON 文件 | SQLite + JSON 导出 |
| **部署** | GitHub Pages (静态) | GitHub Pages (SSG) + Docker |
| **CI/CD** | 单一 cron workflow | lint + test + build + deploy pipeline |
| **测试** | 无 | Go test + pytest + Playwright |
| **文档** | README only | README + docs/ + CONTRIBUTING |

---

## 八、成功标准

- [ ] **代码质量**: Go/Python lint 零警告，测试覆盖率 >= 70%
- [ ] **功能完整**: 追踪 20+ 组织，综合评分，搜索，筛选，趋势图
- [ ] **用户体验**: 页面加载 < 1s，移动端适配，暗色/亮色主题
- [ ] **社区健康**: CONTRIBUTING + Issue 模板 + PR 模板 + CoC
- [ ] **文档完善**: 快速开始 < 3 步，架构文档，API 参考
- [ ] **推广覆盖**: Reddit + HN + Twitter + 中文社区各至少 1 篇

---

*本计划由 Cascade 分析现有代码后编写，供 Claude Opus 4.6 Thinking 执行。*
