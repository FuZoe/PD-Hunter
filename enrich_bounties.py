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
from openai import OpenAI

INPUT_FILE = "bounty_issues.json"
OUTPUT_FILE = "enriched_bounties.json"
EXISTING_FILE = "enriched_bounties.json"  # For preserving expert hints

def extract_amount_from_text(text: str) -> int:
    """Extract dollar amount from text (e.g., '$100', '$1.2k', '$1,000')"""
    if not text:
        return 0
    
    # Match patterns like $100, $1.2k, $1,000, $1000
    patterns = [
        r'\$(\d{1,3}(?:,\d{3})+)',  # $1,000 or $10,000
        r'\$(\d+\.?\d*)k',           # $1.2k or $1k
        r'\$(\d+)',                   # $100 or $1000
    ]
    
    for pattern in patterns:
        match = re.search(pattern, text, re.IGNORECASE)
        if match:
            amount_str = match.group(1).replace(',', '')
            if 'k' in text[match.start():match.end()].lower():
                return int(float(amount_str) * 1000)
            return int(float(amount_str))
    
    return 0

def get_bounty_amount(issue: dict) -> int:
    """Smart extraction of bounty amount from title, labels, and body"""
    # Priority 1: Check labels (most reliable)
    for label in issue.get("labels", []):
        amount = extract_amount_from_text(label)
        if amount > 0:
            return amount
    
    # Priority 2: Check title
    amount = extract_amount_from_text(issue.get("title", ""))
    if amount > 0:
        return amount
    
    # Priority 3: Check body (first 1000 chars)
    body = issue.get("body", "")[:1000]
    amount = extract_amount_from_text(body)
    if amount > 0:
        return amount
    
    return 0

def get_bounty_tier(amount: int) -> str:
    """Determine bounty tier based on amount
    
    S-Tier: $1000+ (high-value bounties)
    A-Tier: $200+ (mid-value bounties)
    B-Tier: <$200 (entry-level bounties)
    """
    if amount >= 1000:
        return "S-Tier"
    elif amount >= 200:
        return "A-Tier"
    else:
        return "B-Tier"

def is_hidden_gem(issue: dict) -> bool:
    """Check if issue is a Hidden Gem (low competition opportunity)
    
    Criteria:
    - Open PR count <= 3
    - Comment count <= 10
    """
    open_pr_count = issue.get("open_pr_count", 0)
    comment_count = issue.get("comment_count", 0)
    return open_pr_count <= 3 and comment_count <= 10

def analyze_issue_with_ai(client: OpenAI, issue: dict) -> dict:
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
   - High: 20+ comments OR many open PRs (high competition), complex debugging, multiple failed attempts
   - Medium: 10-20 comments, moderate complexity
   - Low: <10 comments AND <3 open PRs, clear scope, straightforward fix

2. **Technical Hint**: A one-sentence actionable technical hint for solving this issue.
   Focus on specific code areas, debugging approaches, or implementation strategies.
   Examples: "Check for unclosed channels in Go", "Use git bisect for this regression", "Look for race conditions in concurrent code"

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
                    existing_intel[item["number"]] = item.get("hunter_intelligence", {})
            print(f"Loaded {len(existing_intel)} existing expert hints")
        except Exception as e:
            print(f"Warning: Could not load existing data: {e}")
    return existing_intel


def main():
    token = os.getenv("GITHUB_TOKEN")
    if not token:
        print("Error: GITHUB_TOKEN environment variable not set")
        print("GitHub Models requires a GitHub token for authentication")
        return
    
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
        bounty_amount = get_bounty_amount(issue)
        bounty_tier = get_bounty_tier(bounty_amount)
        
        # Check if we already have expert intelligence for this issue
        if issue_num in existing_intel:
            # PRESERVE existing expert hints
            existing = existing_intel[issue_num]
            print(f"[{i}/{len(issues)}] PRESERVED: #{issue_num} - {issue['title'][:50]}...")
            
            hidden_gem = is_hidden_gem(issue)
            enriched_issue = {
                **issue,
                "hunter_intelligence": {
                    "friction_level": existing.get("friction_level", "Medium"),
                    "technical_hint": existing.get("technical_hint", "Review the issue details."),
                    "bounty_tier": bounty_tier,  # Recalculate tier in case bounty changed
                    "bounty_amount": bounty_amount,
                    "is_hidden_gem": hidden_gem
                }
            }
            preserved_count += 1
        else:
            # NEW issue - generate AI analysis
            print(f"[{i}/{len(issues)}] NEW: #{issue_num} - {issue['title'][:50]}...")
            
            ai_analysis = analyze_issue_with_ai(client, issue)
            
            hidden_gem = is_hidden_gem(issue)
            enriched_issue = {
                **issue,
                "hunter_intelligence": {
                    "friction_level": ai_analysis.get("friction_level", "Medium"),
                    "technical_hint": ai_analysis.get("technical_hint", "Review the issue details."),
                    "bounty_tier": bounty_tier,
                    "bounty_amount": bounty_amount,
                    "is_hidden_gem": hidden_gem
                }
            }
            
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
    
    tiers = {"S-Tier": 0, "A-Tier": 0, "B-Tier": 0}
    hidden_gems_count = 0
    for issue in enriched_issues:
        tier = issue["hunter_intelligence"]["bounty_tier"]
        tiers[tier] += 1
        if issue["hunter_intelligence"].get("is_hidden_gem", False):
            hidden_gems_count += 1
    
    print(f"\nTier Summary:")
    print(f"  S-Tier ($500+): {tiers['S-Tier']}")
    print(f"  A-Tier ($200+): {tiers['A-Tier']}")
    print(f"  B-Tier (other): {tiers['B-Tier']}")
    print(f"\nHidden Gems (low competition): {hidden_gems_count}")

if __name__ == "__main__":
    main()
