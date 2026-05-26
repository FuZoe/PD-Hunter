---
name: testing-pd-hunter-frontend
description: Test the PD-Hunter Next.js frontend end-to-end. Use when verifying UI changes, routing, or data display.
---

# Testing PD-Hunter Frontend

## Prerequisites
- Node.js 18+ installed
- `enriched_bounties.json` present at repo root (used by `api-server.ts` as fallback)

## Setup

```bash
cd frontend
npm install
npm run build   # generates static export in out/
```

## Serving for Testing

**Use static build, NOT dev server** for testing routes with `output: "export"`. The dev server has strict `generateStaticParams` enforcement that may differ from production behavior.

```bash
npm install -g serve
serve frontend/out -l 3002 --no-clipboard
# WITHOUT -s flag (SPA mode breaks static routing)
```

Clean URLs work automatically with `serve` (e.g., `/bounty/commaai-flash--42` maps to `commaai-flash--42.html`).

**Do NOT use `serve -s`** — it enables SPA fallback that serves index.html for all routes, breaking static page routing.

**Do NOT use Python `http.server`** — it doesn't handle clean URLs (requires `.html` extension).

## Key Test Scenarios

### Bounty Detail Routes
- Bounty detail pages use composite slug format: `/bounty/{owner}-{repo}--{number}`
- The `bountySlug()` function in `frontend/src/lib/utils.ts` generates slugs
- Collision data (as of May 2026): #42 (flash/website), #30 (agnos-builder/website)

### Backward Compatibility
- Unique bare numbers (e.g., `/bounty/2281`) resolve directly to the bounty
- Colliding bare numbers (e.g., `/bounty/42`) show a disambiguation page (`BountyDisambiguation.tsx`)
- Non-existent numbers return 404

### Data Loading
- Homepage loads data client-side from `./data/enriched_bounties.json`
- Bounty detail pages are statically generated (SSG) — data is baked into HTML at build time
- Data endpoint: `/data/enriched_bounties.json`

## Verification Commands

```bash
# Lint and typecheck
cd frontend && npm run lint && npx tsc --noEmit

# Check build generates expected page count
npm run build 2>&1 | grep "Generating static pages"

# Verify no duplicate static paths
ls frontend/out/bounty/ | sort | uniq -d  # should be empty

# Verify collision files exist separately
ls frontend/out/bounty/ | grep -E '(flash--42|website--42)'
```

## Common Issues
- Dev server (`npm run dev`) might fail with "missing param in generateStaticParams" for dynamically added routes — use static build instead
- Homepage may show "Loading bounties" indefinitely in dev mode if the data file path doesn't resolve — this is a dev-mode issue, not a production bug
- The `serve` package requires explicit `--no-clipboard` on headless environments

## Devin Secrets Needed
None — no authentication required for local testing.
