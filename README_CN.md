# PD-Hunter 赏金猎人情报中心

AI 驱动的 ProjectDiscovery 赏金情报仪表盘。
v
[English](./README.md)| [简体中文]

## 直接预览

see the [**dashboard**](https://fuzoe.github.io/PD-Hunter/static/dashboard.html)

## ✨ 功能特点

- **猎人卡片** - 技术提示、赏金金额、难度等级
- **S-Tier 高亮** - 高价值赏金醒目显示
- **专家提示保留** - 人工提示不会被 AI 覆盖
- **自动更新** - GitHub Actions 每 6 小时刷新数据

## 🚀 快速开始

### 本地运行

```bash
# 1. 爬取赏金 issues
go run fetch_bounty_issues.go

# 2. AI 分析 (需要 GITHUB_TOKEN)
export GITHUB_TOKEN=你的_token
pip install -r requirements.txt
python enrich_bounties.py

# 3. 复制到 static 文件夹
cp enriched_bounties.json static/

# 4. 打开仪表盘
# 浏览器打开 static/dashboard.html
```

### GitHub Pages 部署

部署到 GitHub Pages 后，仪表盘每 6 小时自动更新。

**在线访问**：https://fuzoe.github.io/PD-Hunter/static/dashboard.html

## 📁 项目结构

| 文件 | 说明 |
|------|------|
| `fetch_bounty_issues.go` | Go 爬虫 - 从 GitHub API 获取赏金 issues |
| `enrich_bounties.py` | Python 脚本 - 使用 GPT-4o 分析 issues |
| `static/dashboard.html` | 前端仪表盘 - Hacker Dark Mode 主题 |
| `.github/workflows/update_bounties.yml` | 自动化工作流 |

## 🔧 技术栈

- **Go 1.22+** - 爬虫
- **Python 3.11+** - AI 分析
- **OpenAI SDK** - 调用 GitHub Models (GPT-4o)
- **Tailwind CSS** - 样式
- **GitHub Actions** - 自动化

## 🎯 赏金分级

| 等级 | 金额 | 说明 |
|------|------|------|
| **S-Tier** | $500+ | 高价值，值得深入研究 |
| **A-Tier** | $200+ | 中等价值 |
| **B-Tier** | 其他 | 入门级 |

## 📝 Expert Hint Preservation

AI 分析时会保留已有的人工专家提示：

```python
if issue_num in existing_intel:
    # 保留人工提示，不调用 AI
else:
    # 新 issue → 调用 AI 分析
```

## 📄 许可证

MIT
