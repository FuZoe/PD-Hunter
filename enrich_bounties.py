#!/usr/bin/env python3
"""
PD-Hunter Intelligence Enrichment Script

Uses GitHub Models (GPT-4o) to analyze bounty issues and generate:
- Friction Level (High/Medium/Low)
- Technical Hint
- Bounty Tier (S-Tier/A-Tier/B-Tier)

CRITICAL: Preserves existing expert hints - only generates AI hints for NEW issues.
"""

import json
import os
import re
import time
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any

INPUT_FILE = "bounty_issues.json"
OUTPUT_FILE = "enriched_bounties.json"
EXISTING_FILE = "enriched_bounties.json"  # For preserving expert hints

MANUAL_RISK_OVERRIDES = {
    "commaai/openpilot#32386": {
        "bounty_amount": 2000,
        "friction_level": "High",
        "technical_hint": (
            "Critical risk: verify maintainer interest before work; vamOS appears "
            "to supersede the old mainline-kernel path, and comma 3X hardware "
            "access is required."
        ),
        "risk_level": "Critical",
        "risk_warning": (
            "This bounty appears superseded by comma.ai's internal vamOS work. "
            "Treat it as low-EV unless maintainers reconfirm that outside PRs "
            "can still win the remaining mainline-kernel reward."
        ),
        "risk_reasons": [
            "Internal vamOS project is actively replacing the old AGNOS path",
            "Stalled draft PR exists for the original bounty work",
            "Requires comma 3X hardware or remote device access",
        ],
        "score_cap": 25,
    }
}

SUPPORTED_TOKEN_CURRENCIES = ("MRG", "XTM", "XTR")


@dataclass(frozen=True)
class BountyReward:
    """A reward in its native currency, before any USD-only scoring."""

    amount: int
    currency: str
    source: str
    explicit: bool


_AMOUNT_PATTERN = r"(?:\d{1,3}(?:,\d{3})+|\d+(?:\.\d+)?)(?:\s*[kK])?"
_USD_RE = re.compile(rf"\$(?P<amount>{_AMOUNT_PATTERN})")
_TOKEN_AFTER_AMOUNT_RE = re.compile(
    rf"(?P<amount>{_AMOUNT_PATTERN})\s*[-:]?\s*"
    rf"(?P<currency>{'|'.join(SUPPORTED_TOKEN_CURRENCIES)})\b",
    re.IGNORECASE,
)
_TOKEN_BEFORE_AMOUNT_RE = re.compile(
    rf"\b(?P<currency>{'|'.join(SUPPORTED_TOKEN_CURRENCIES)})"
    rf"\s*[-:=]?\s*(?P<amount>{_AMOUNT_PATTERN})",
    re.IGNORECASE,
)
_REWARD_CONTEXT_RE = re.compile(r"(?:/bounty\b|\bbount(?:y|ies)\b|\breward\b|\bpayout\b|\bprize\b)", re.IGNORECASE)


def get_issue_key(issue: dict) -> str:
    """Return the stable repository+issue key used for preserved intelligence."""
    return f"{issue['repository']}#{issue['number']}"


def apply_manual_risk_override(issue: dict, intel: dict) -> dict:
    """Apply curated high-risk overrides for known stale or superseded bounties."""
    override = MANUAL_RISK_OVERRIDES.get(get_issue_key(issue))
    if not override:
        return intel

    adjusted = {**intel}
    adjusted["friction_level"] = override["friction_level"]
    adjusted["technical_hint"] = override["technical_hint"]
    adjusted["bounty_amount"] = override["bounty_amount"]
    adjusted["bounty_currency"] = "USD"
    adjusted["bounty_native_amount"] = override["bounty_amount"]
    adjusted["bounty_tier"] = get_bounty_tier(override["bounty_amount"])
    adjusted["is_hidden_gem"] = False
    adjusted["risk_level"] = override["risk_level"]
    adjusted["risk_warning"] = override["risk_warning"]
    adjusted["risk_reasons"] = override["risk_reasons"]

    if "bounty_score" in adjusted:
        adjusted["bounty_score"] = min(adjusted["bounty_score"], override["score_cap"])

    if "score_breakdown" in adjusted:
        adjusted["score_breakdown"] = {
            **adjusted["score_breakdown"],
            "amount": int(min(override["bounty_amount"] / 50, 100)),
            "feasibility": 30,
        }

    return adjusted


def _parse_amount(raw_amount: str) -> int:
    normalized = raw_amount.replace(",", "").replace(" ", "")
    multiplier = 1000 if normalized.lower().endswith("k") else 1
    if multiplier != 1:
        normalized = normalized[:-1]
    return int(float(normalized) * multiplier)


def _reward_is_explicit(text: str, start: int, end: int) -> bool:
    context_start = max(0, start - 48)
    context_end = min(len(text), end + 48)
    return bool(_REWARD_CONTEXT_RE.search(text[context_start:context_end]))


def _extract_reward_candidates(text: str, source: str) -> list[BountyReward]:
    if not text:
        return []

    candidates: list[tuple[int, BountyReward]] = []
    occupied_ranges: list[tuple[int, int]] = []

    for pattern in (_TOKEN_AFTER_AMOUNT_RE, _TOKEN_BEFORE_AMOUNT_RE):
        for match in pattern.finditer(text):
            match_range = (match.start(), match.end())
            if any(match_range[0] < end and start < match_range[1] for start, end in occupied_ranges):
                continue
            occupied_ranges.append(match_range)
            candidates.append(
                (
                    match.start(),
                    BountyReward(
                        amount=_parse_amount(match.group("amount")),
                        currency=match.group("currency").upper(),
                        source=source,
                        explicit=_reward_is_explicit(text, match.start(), match.end()),
                    ),
                )
            )

    for match in _USD_RE.finditer(text):
        candidates.append(
            (
                match.start(),
                BountyReward(
                    amount=_parse_amount(match.group("amount")),
                    currency="USD",
                    source=source,
                    explicit=_reward_is_explicit(text, match.start(), match.end()),
                ),
            )
        )

    return [candidate for _, candidate in sorted(candidates, key=lambda item: item[0])]


def extract_amount_from_text(text: str) -> int:
    """Extract the first USD amount while preserving the legacy return type."""
    for candidate in _extract_reward_candidates(text, "text"):
        if candidate.currency == "USD":
            return candidate.amount
    return 0


def get_bounty_reward(issue: dict) -> BountyReward | None:
    """Select the most credible declared reward from labels, title, and body.

    Labels remain the preferred source when they explicitly describe a bounty.
    An explicit declaration such as ``/bounty $75`` can override a generic amount
    label such as ``$1``.
    """
    candidates: list[tuple[int, int, BountyReward]] = []
    sequence = 0

    source_values = [
        ("label", label)
        for label in issue.get("labels", [])
        if isinstance(label, str)
    ]
    source_values.extend(
        [
            ("title", issue.get("title", "")),
            ("body", (issue.get("body") or "")[:5000]),
        ]
    )

    source_priority = {"label": 300, "title": 200, "body": 100}
    for source, text in source_values:
        for reward in _extract_reward_candidates(text, source):
            # Explicit reward language outranks a generic amount from a normally
            # higher-priority source. Ties retain the first textual declaration.
            confidence = source_priority[source]
            if reward.currency != "USD":
                # A native-currency declaration is meaningful even when it is not
                # surrounded by the word "bounty" (for example, ``50 MRG``).
                confidence += 125
            if reward.explicit:
                confidence += 250
            if (
                source == "label"
                and reward.currency == "USD"
                and reward.amount <= 1
                and not reward.explicit
            ):
                # A repository-wide ``$1`` marker is commonly a placeholder, not
                # the actual payout. Let a declared token reward override it.
                confidence -= 150
            candidates.append((confidence, -sequence, reward))
            sequence += 1

    if not candidates:
        return None
    return max(candidates, key=lambda item: (item[0], item[1]))[2]


def get_bounty_fields(issue: dict) -> dict:
    """Return backward-compatible USD fields plus native reward metadata."""
    reward = get_bounty_reward(issue)
    if reward is None:
        return {
            "bounty_amount": 0,
            "bounty_currency": None,
            "bounty_native_amount": 0,
        }

    return {
        # Existing consumers use this value for USD ranking and aggregation.
        "bounty_amount": reward.amount if reward.currency == "USD" else 0,
        "bounty_currency": reward.currency,
        "bounty_native_amount": reward.amount,
    }


def get_bounty_amount(issue: dict) -> int:
    """Return only a confirmed USD amount for legacy callers and scoring."""
    return get_bounty_fields(issue)["bounty_amount"]

def get_bounty_tier(amount: int, currency: str | None = "USD") -> str:
    """Determine bounty tier based on amount

    S-Tier: $1000+ (high-value bounties)
    A-Tier: $200+ (mid-value bounties)
    B-Tier: <$200 (entry-level bounties)
    Unpriced: non-USD rewards without a reliable conversion rate
    """
    if currency and currency != "USD":
        return "Unpriced"
    if amount >= 1000:
        return "S-Tier"
    elif amount >= 200:
        return "A-Tier"
    else:
        return "B-Tier"

def calculate_bounty_score(issue: dict, intel: dict) -> tuple:
    """Calculate comprehensive bounty score (0-100) with breakdown.
    
    Score = 0.35 * AmountScore + 0.25 * FeasibilityScore + 0.25 * CompetitionScore + 0.15 * FreshnessScore
    """
    # Amount score: $5000+ = 100
    amount = intel.get("bounty_amount", 0)
    amount_score = min(amount / 50, 100)
    
    # Feasibility score based on friction level
    friction = intel.get("friction_level", "Medium")
    feasibility_map = {"Low": 90, "Medium": 60, "High": 30}
    feasibility_score = feasibility_map.get(friction, 60)
    
    # Competition score: fewer PRs and comments = higher score
    open_prs = issue.get("open_pr_count", 0)
    comments = issue.get("comment_count", 0)
    competition_score = max(0, 100 - open_prs * 15 - comments * 2)
    
    # Freshness score: more recently updated = higher score
    updated = issue.get("updated_at", "")
    try:
        updated_date = datetime.fromisoformat(updated.replace("Z", "+00:00"))
        days_old = (datetime.now(timezone.utc) - updated_date).days
        freshness_score = max(0, 100 - days_old * 0.5)
    except (ValueError, TypeError):
        freshness_score = 50
    
    total = int(0.35 * amount_score + 0.25 * feasibility_score + 0.25 * competition_score + 0.15 * freshness_score)
    total = max(0, min(100, total))
    
    breakdown = {
        "amount": int(amount_score),
        "feasibility": int(feasibility_score),
        "competition": int(competition_score),
        "freshness": int(freshness_score)
    }
    
    return total, breakdown

def is_hidden_gem(issue: dict) -> bool:
    """Check if issue is a Hidden Gem (low competition opportunity)
    
    Criteria:
    - Open PR count <= 3
    - Comment count <= 10
    """
    open_pr_count = issue.get("open_pr_count", 0)
    comment_count = issue.get("comment_count", 0)
    return open_pr_count <= 3 and comment_count <= 10

def analyze_issue_with_ai(client: Any, issue: dict) -> dict:
    """Call GPT-4o to analyze the issue and generate Hunter Intelligence"""
    
    body_preview = issue.get("body", "")[:2000] if issue.get("body") else "No description"
    
    open_pr_count = issue.get('open_pr_count', 0)
    
    prompt = f"""Analyze this GitHub bounty issue and provide Hunter Intelligence:

**Title:** {issue['title']}
**Repository:** {issue['repository']}
**Labels:** {', '.join(issue['labels'])}
**Comment Count:** {issue['comment_count']}
**Open PR Count:** {open_pr_count}
**Created:** {issue['created_at']}
**Updated:** {issue['updated_at']}

**Description:**
{body_preview}

Based on this issue, provide:

1. **Friction Level** (High/Medium/Low):
   
   **High Friction** - Mark as High if ANY of these apply:
   - Requires access to SPECIFIC/RARE vehicles (e.g., "Rivian Gen2", "2025+ model", specific car makes)
   - Requires expensive or rare hardware that most developers don't own
   - Explicitly states "must test on real vehicle" or "physical access required"
   - 20+ comments OR many open PRs indicating high competition or complexity
   - Keywords indicating rare hardware: "2025+", "Gen2", specific car model names, "must own"
   
   **Medium Friction**:
   - Requires comma device + any commonly supported car (not a specific rare model)
   - General hardware debugging without rare components
   - 10-20 comments, moderate complexity
   
   **Low Friction**:
   - Pure software tasks: code refactoring, bug fixes, documentation, CI/testing
   - Simulation-only tasks that don't require physical devices
   - Common/accessible hardware: comma 3X, Raspberry Pi, Arduino (developers likely own these)
   - <10 comments AND <3 open PRs, clear scope, straightforward fix

2. **Technical Hint**: A one-sentence actionable technical hint for solving this issue.
   - If the issue requires RARE/SPECIFIC hardware, START with "⚠️ Physical Access Required."
   - Focus on specific code areas, debugging approaches, or implementation strategies.
   - Examples: "Check for unclosed channels in Go", "Use git bisect for this regression"

Respond in this exact JSON format:
{{"friction_level": "High|Medium|Low", "technical_hint": "Your hint here"}}
"""

    try:
        response = client.chat.completions.create(
            model="gpt-4o",
            messages=[
                {"role": "system", "content": "You are a security researcher and Go/Python expert analyzing open source bounty issues. Provide concise, actionable analysis."},
                {"role": "user", "content": prompt}
            ],
            temperature=0.3,
            max_tokens=200
        )
        
        content = response.choices[0].message.content.strip()
        
        json_match = re.search(r'\{[^}]+\}', content)
        if json_match:
            return json.loads(json_match.group())
        
        return {"friction_level": "Medium", "technical_hint": "Review the issue details and related code."}
        
    except Exception as e:
        print(f"  AI analysis error: {e}")
        return {"friction_level": "Medium", "technical_hint": "Review the issue details and related code."}

def load_existing_intelligence() -> dict:
    """Load existing enriched data to preserve expert hints."""
    existing_intel = {}
    if os.path.exists(EXISTING_FILE):
        try:
            with open(EXISTING_FILE, "r", encoding="utf-8") as f:
                existing = json.load(f)
                for item in existing:
                    key = f"{item['repository']}#{item['number']}"
                    existing_intel[key] = item.get("hunter_intelligence", {})
            print(f"Loaded {len(existing_intel)} existing expert hints")
        except Exception as e:
            print(f"Warning: Could not load existing data: {e}")
    return existing_intel


def summarize_enriched_issues(enriched_issues: list[dict]) -> tuple[dict[str, int], int]:
    """Return tier and hidden-gem counts without assuming every tier is USD-priced."""
    tiers = {"S-Tier": 0, "A-Tier": 0, "B-Tier": 0, "Unpriced": 0}
    hidden_gems_count = 0
    for issue in enriched_issues:
        tier = issue["hunter_intelligence"]["bounty_tier"]
        # Keep the summary resilient to older/manual records with a custom tier.
        tiers.setdefault(tier, 0)
        tiers[tier] += 1
        if issue["hunter_intelligence"].get("is_hidden_gem", False):
            hidden_gems_count += 1
    return tiers, hidden_gems_count


def main():
    token = os.getenv("GITHUB_TOKEN")
    if not token:
        print("Error: GITHUB_TOKEN environment variable not set")
        print("GitHub Models requires a GitHub token for authentication")
        return

    try:
        from openai import OpenAI
    except ImportError as exc:
        raise SystemExit(
            "Error: the 'openai' package is required to run enrichment; "
            "install requirements.txt first"
        ) from exc

    client = OpenAI(
        base_url="https://models.inference.ai.azure.com",
        api_key=token
    )
    
    print(f"Loading issues from {INPUT_FILE}...")
    with open(INPUT_FILE, "r", encoding="utf-8") as f:
        issues = json.load(f)
    
    print(f"Found {len(issues)} issues to process\n")
    
    # Load existing expert hints to preserve them
    existing_intel = load_existing_intelligence()
    
    enriched_issues = []
    new_count = 0
    preserved_count = 0
    
    for i, issue in enumerate(issues, 1):
        issue_num = issue["number"]
        bounty_fields = get_bounty_fields(issue)
        bounty_amount = bounty_fields["bounty_amount"]
        bounty_tier = get_bounty_tier(
            bounty_amount, bounty_fields["bounty_currency"]
        )
        
        # Check if we already have expert intelligence for this issue
        issue_key = get_issue_key(issue)
        if issue_key in existing_intel:
            # PRESERVE existing expert hints
            existing = existing_intel[issue_key]
            print(f"[{i}/{len(issues)}] PRESERVED: #{issue_num} - {issue['title'][:50]}...")
            
            hidden_gem = is_hidden_gem(issue)
            partial_intel = {
                "friction_level": existing.get("friction_level", "Medium"),
                "bounty_amount": bounty_amount,
            }
            score, score_breakdown = calculate_bounty_score(issue, partial_intel)
            hunter_intelligence = {
                "friction_level": existing.get("friction_level", "Medium"),
                "technical_hint": existing.get("technical_hint", "Review the issue details."),
                "bounty_tier": bounty_tier,
                **bounty_fields,
                "is_hidden_gem": hidden_gem,
                "bounty_score": score,
                "score_breakdown": score_breakdown
            }
            hunter_intelligence = apply_manual_risk_override(issue, hunter_intelligence)
            enriched_issue = {**issue, "hunter_intelligence": hunter_intelligence}
            preserved_count += 1
        else:
            # NEW issue - generate AI analysis
            print(f"[{i}/{len(issues)}] NEW: #{issue_num} - {issue['title'][:50]}...")
            
            ai_analysis = analyze_issue_with_ai(client, issue)
            
            hidden_gem = is_hidden_gem(issue)
            new_intel = {
                "friction_level": ai_analysis.get("friction_level", "Medium"),
                "bounty_amount": bounty_amount,
            }
            score, score_breakdown = calculate_bounty_score(issue, new_intel)
            hunter_intelligence = {
                "friction_level": ai_analysis.get("friction_level", "Medium"),
                "technical_hint": ai_analysis.get("technical_hint", "Review the issue details."),
                "bounty_tier": bounty_tier,
                **bounty_fields,
                "is_hidden_gem": hidden_gem,
                "bounty_score": score,
                "score_breakdown": score_breakdown
            }
            hunter_intelligence = apply_manual_risk_override(issue, hunter_intelligence)
            enriched_issue = {**issue, "hunter_intelligence": hunter_intelligence}
            
            print(f"  AI Hint: {ai_analysis.get('technical_hint', 'N/A')[:80]}")
            new_count += 1
            time.sleep(0.5)  # Rate limit only for AI calls
        
        enriched_issues.append(enriched_issue)
    
    print(f"\nSaving enriched data to {OUTPUT_FILE}...")
    with open(OUTPUT_FILE, "w", encoding="utf-8") as f:
        json.dump(enriched_issues, f, indent=2, ensure_ascii=False)
    
    print(f"\nDone! Processed {len(enriched_issues)} issues.")
    print(f"  Preserved: {preserved_count} expert hints")
    print(f"  New (AI):  {new_count} hints generated")
    
    tiers, hidden_gems_count = summarize_enriched_issues(enriched_issues)
    
    print(f"\nTier Summary:")
    print(f"  S-Tier ($1000+): {tiers['S-Tier']}")
    print(f"  A-Tier ($200+): {tiers['A-Tier']}")
    print(f"  B-Tier (other): {tiers['B-Tier']}")
    print(f"  Unpriced (token): {tiers['Unpriced']}")
    print(f"\nHidden Gems (low competition): {hidden_gems_count}")

if __name__ == "__main__":
    main()
