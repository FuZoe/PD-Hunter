#!/usr/bin/env python3
"""
Backfill bounty_score and score_breakdown into enriched_bounties.json.

This script re-uses the calculate_bounty_score function from enrich_bounties.py
to add the missing fields to existing data WITHOUT calling any AI APIs.
"""

import json
import sys
import os

# Reuse the scoring function from the main module
sys.path.insert(0, os.path.dirname(__file__))
from enrich_bounties import calculate_bounty_score

DATA_FILE = "enriched_bounties.json"


def main():
    print(f"Loading {DATA_FILE}...")
    with open(DATA_FILE, "r", encoding="utf-8") as f:
        issues = json.load(f)

    print(f"Found {len(issues)} issues")

    updated = 0
    for issue in issues:
        intel = issue.get("hunter_intelligence", {})

        # Build the inputs for scoring
        score_input_intel = {
            "bounty_amount": intel.get("bounty_amount", 0),
            "friction_level": intel.get("friction_level", "Medium"),
        }

        score, breakdown = calculate_bounty_score(issue, score_input_intel)
        intel["bounty_score"] = score
        intel["score_breakdown"] = breakdown
        issue["hunter_intelligence"] = intel
        updated += 1

        print(
            f"  #{issue['number']:>6} | "
            f"Score: {score:>3} | "
            f"A:{breakdown['amount']:>3} F:{breakdown['feasibility']:>2} "
            f"C:{breakdown['competition']:>3} Fr:{breakdown['freshness']:>3} | "
            f"{issue['title'][:50]}"
        )

    # Write back
    with open(DATA_FILE, "w", encoding="utf-8") as f:
        json.dump(issues, f, indent=2, ensure_ascii=False)

    print(f"\nDone! Backfilled bounty_score for {updated}/{len(issues)} issues")

    # Also copy to frontend public data
    frontend_data = os.path.join("frontend", "public", "data", "enriched_bounties.json")
    if os.path.exists(os.path.dirname(frontend_data)):
        with open(frontend_data, "w", encoding="utf-8") as f:
            json.dump(issues, f, indent=2, ensure_ascii=False)
        print(f"Copied to {frontend_data}")


if __name__ == "__main__":
    main()
