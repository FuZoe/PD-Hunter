# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 2.x     | Yes                |
| 1.x     | No                 |

## Reporting a Vulnerability

If you discover a security vulnerability in PD-Hunter, please report it responsibly:

1. **Do NOT** open a public GitHub issue
2. Email the maintainers or use GitHub's [private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability)
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

We will acknowledge your report within 48 hours and aim to release a fix within 7 days for critical issues.

## Scope

This policy covers:

- The PD-Hunter CLI tool (`cmd/hunter`)
- The frontend dashboard (`frontend/`)
- The enrichment pipeline (`enrich_bounties.py`)
- GitHub Actions workflows

## Best Practices

- Never commit API keys or tokens to the repository
- Use environment variables for sensitive configuration
- Keep dependencies up to date
