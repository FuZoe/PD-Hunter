# PD-Hunter Intelligence

AI-powered bounty intelligence dashboard for ProjectDiscovery repositories.
## Preview

see the [dashboard](https://fuzoe.github.io/PD-Hunter/static/dashboard.html)

## Features

- **Hunter Cards** - Technical hints, bounty amounts, friction levels
- **S-Tier Highlighting** - High-value bounties prominently featured
- **Expert Hint Preservation** - Manual hints preserved across updates
- **Auto-Update** - GitHub Actions refreshes data every 6 hours

## Quick Start

### Local Run

```bash
# 1. Fetch bounty issues
go run fetch_bounty_issues.go

# 2. Enrich with AI (requires GITHUB_TOKEN)
export GITHUB_TOKEN=your_token
pip install -r requirements.txt
python enrich_bounties.py

# 3. Copy to static folder
cp enriched_bounties.json static/

# 4. Open dashboard
# Open static/dashboard.html in browser
```

### GitHub Pages

Deploy to GitHub Pages and the dashboard auto-updates every 6 hours.

## Files

| File | Description |
|------|-------------|
| `fetch_bounty_issues.go` | Go scraper for GitHub API |
| `enrich_bounties.py` | GPT-4o analysis via GitHub Models |
| `static/dashboard.html` | Hacker Dark Mode dashboard |
| `.github/workflows/update_bounties.yml` | Automation |

## License

MIT
