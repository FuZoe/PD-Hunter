# Contributing to PD-Hunter

Thank you for your interest in contributing to PD-Hunter! This guide will help you get started.

## Development Setup

### Prerequisites

- **Go** 1.22+
- **Node.js** 18+
- **Python** 3.11+
- **Git**

### Getting Started

```bash
# Clone the repo
git clone https://github.com/FuZoe/PD-Hunter.git
cd PD-Hunter

# Go backend
go mod tidy
go build ./cmd/hunter
go test ./... -v

# Frontend
cd frontend
npm install
npm run dev

# Python enrichment
pip install -r requirements.txt
```

## Project Structure

```
PD-Hunter/
├── cmd/hunter/          # CLI entry point (Go)
├── pkg/
│   ├── scraper/         # GitHub API scraping (Go)
│   └── exporter/        # Data export (Go)
├── frontend/            # Next.js dashboard
│   ├── src/app/         # Pages
│   ├── src/components/  # React components
│   ├── src/hooks/       # Custom hooks
│   └── src/lib/         # Utilities & types
├── enrich_bounties.py   # AI enrichment script
├── mapping.json         # Organization config
└── static/              # Legacy static dashboard
```

## How to Contribute

### Adding a New Organization

1. Edit `mapping.json` and add your organization entry
2. Include the org name, bounty labels, and a short note
3. Submit a PR with the title: `feat: add {org-name} to tracking`

### Bug Reports

Use the [Bug Report template](.github/ISSUE_TEMPLATE/bug_report.md) when filing bugs. Include:
- Steps to reproduce
- Expected vs actual behavior
- Environment details

### Feature Requests

Use the [Feature Request template](.github/ISSUE_TEMPLATE/feature_request.md).

### Code Contributions

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make your changes
4. Run tests:
   ```bash
   go test ./... -v       # Go tests
   cd frontend && npm run lint && npm run build  # Frontend
   ```
5. Commit using [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat: add new feature`
   - `fix: resolve bug`
   - `docs: update documentation`
   - `refactor: restructure code`
   - `chore: maintenance tasks`
6. Push and open a Pull Request

## Code Style

- **Go**: Follow standard Go conventions. Run `go vet ./...` before committing.
- **TypeScript/React**: ESLint + Prettier. Run `npm run lint`.
- **Python**: Follow PEP 8. Use type hints where possible.
- **Commits**: Conventional Commits format, lowercase prefix, no emojis.

## Pull Request Guidelines

- Keep PRs focused — one feature or fix per PR
- Update tests if you change behavior
- Update documentation if you change APIs
- Ensure CI passes before requesting review

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
