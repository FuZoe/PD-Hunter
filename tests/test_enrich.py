"""Tests for enrich_bounties.py core functions."""
import json
import sys
import os
import pytest

# Add project root to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from enrich_bounties import (
    apply_manual_risk_override,
    extract_amount_from_text,
    get_bounty_amount,
    get_bounty_fields,
    get_bounty_reward,
    get_bounty_tier,
    is_hidden_gem,
    calculate_bounty_score,
    load_existing_intelligence,
    summarize_enriched_issues,
)


class TestExtractAmount:
    def test_dollar_simple(self):
        assert extract_amount_from_text("$100 bounty") == 100

    def test_dollar_large(self):
        assert extract_amount_from_text("$4000 bounty") == 4000

    def test_dollar_comma(self):
        assert extract_amount_from_text("$1,000 bounty") == 1000

    def test_dollar_comma_large(self):
        assert extract_amount_from_text("$10,000 reward") == 10000

    def test_dollar_k_integer(self):
        assert extract_amount_from_text("$1k bounty") == 1000

    def test_dollar_k_decimal(self):
        assert extract_amount_from_text("$1.2k bounty") == 1200

    def test_no_amount(self):
        assert extract_amount_from_text("no amount here") == 0

    def test_empty_string(self):
        assert extract_amount_from_text("") == 0

    def test_none_input(self):
        assert extract_amount_from_text(None) == 0

    def test_multiple_amounts_takes_first(self):
        # Should find the first dollar amount
        result = extract_amount_from_text("$100 and $200")
        assert result > 0

    def test_bounty_in_brackets(self):
        assert extract_amount_from_text("[$500 bounty] Fix this bug") == 500


class TestGetBountyTier:
    def test_s_tier_boundary(self):
        assert get_bounty_tier(1000) == "S-Tier"

    def test_s_tier_high(self):
        assert get_bounty_tier(5000) == "S-Tier"

    def test_a_tier_boundary(self):
        assert get_bounty_tier(200) == "A-Tier"

    def test_a_tier_mid(self):
        assert get_bounty_tier(500) == "A-Tier"

    def test_a_tier_high(self):
        assert get_bounty_tier(999) == "A-Tier"

    def test_b_tier_zero(self):
        assert get_bounty_tier(0) == "B-Tier"

    def test_non_usd_reward_is_unpriced(self):
        assert get_bounty_tier(0, "MRG") == "Unpriced"

    def test_b_tier_low(self):
        assert get_bounty_tier(50) == "B-Tier"

    def test_b_tier_boundary(self):
        assert get_bounty_tier(199) == "B-Tier"


class TestEnrichedSummary:
    def test_token_rewards_are_counted_as_unpriced(self):
        tiers, hidden_gems = summarize_enriched_issues([
            {"hunter_intelligence": {"bounty_tier": "Unpriced", "is_hidden_gem": True}},
            {"hunter_intelligence": {"bounty_tier": "B-Tier", "is_hidden_gem": False}},
        ])
        assert tiers == {"S-Tier": 0, "A-Tier": 0, "B-Tier": 1, "Unpriced": 1}
        assert hidden_gems == 1

    def test_legacy_custom_tier_does_not_crash_summary(self):
        tiers, _ = summarize_enriched_issues([
            {"hunter_intelligence": {"bounty_tier": "Legacy", "is_hidden_gem": False}},
        ])
        assert tiers["Legacy"] == 1


class TestIsHiddenGem:
    def test_is_gem_zero_prs(self):
        assert is_hidden_gem({"open_pr_count": 0, "comment_count": 5}) is True

    def test_is_gem_boundary(self):
        assert is_hidden_gem({"open_pr_count": 3, "comment_count": 10}) is True

    def test_not_gem_high_prs(self):
        assert is_hidden_gem({"open_pr_count": 5, "comment_count": 5}) is False

    def test_not_gem_high_comments(self):
        assert is_hidden_gem({"open_pr_count": 1, "comment_count": 15}) is False

    def test_not_gem_both_high(self):
        assert is_hidden_gem({"open_pr_count": 10, "comment_count": 50}) is False

    def test_missing_fields_defaults(self):
        assert is_hidden_gem({}) is True  # defaults to 0


class TestGetBountyAmount:
    def test_from_label(self):
        issue = {"labels": ["$500 bounty"], "title": "Some title", "body": "body"}
        assert get_bounty_amount(issue) == 500

    def test_from_title(self):
        issue = {"labels": ["bounty"], "title": "[$100 bounty] Fix bug", "body": ""}
        assert get_bounty_amount(issue) == 100

    def test_from_body(self):
        issue = {"labels": ["bounty"], "title": "Fix bug", "body": "This has a $200 bounty."}
        assert get_bounty_amount(issue) == 200

    def test_label_priority_over_title(self):
        issue = {"labels": ["$500 bounty"], "title": "[$100 bounty] Fix", "body": ""}
        assert get_bounty_amount(issue) == 500

    def test_no_amount_anywhere(self):
        issue = {"labels": ["bounty"], "title": "Fix bug", "body": "No amount."}
        assert get_bounty_amount(issue) == 0

    def test_empty_issue(self):
        issue = {"labels": [], "title": "", "body": ""}
        assert get_bounty_amount(issue) == 0

    def test_explicit_body_command_overrides_generic_amount_label(self):
        issue = {
            "labels": ["💎 Bounty", "$1"],
            "title": "Add is_extended_promotional column",
            "body": "Some details\n\n/bounty $75",
        }

        reward = get_bounty_reward(issue)

        assert reward is not None
        assert reward.source == "body"
        assert reward.explicit is True
        assert get_bounty_amount(issue) == 75

    def test_explicit_bounty_label_still_has_priority(self):
        issue = {
            "labels": ["$500 bounty"],
            "title": "[$100 bounty] Fix",
            "body": "/bounty $75",
        }

        assert get_bounty_amount(issue) == 500

    def test_token_title_overrides_placeholder_dollar_label(self):
        issue = {
            "labels": ["💎 Bounty", "$1"],
            "title": "[50 MRG] Implement the feature",
            "body": "Details",
        }

        reward = get_bounty_reward(issue)

        assert reward is not None
        assert reward.currency == "MRG"
        assert reward.amount == 50

    def test_token_body_overrides_placeholder_dollar_label(self):
        issue = {
            "labels": ["💎 Bounty", "$1"],
            "title": "Implement the feature",
            "body": "Reward is 50 MRG",
        }

        reward = get_bounty_reward(issue)

        assert reward is not None
        assert reward.currency == "MRG"
        assert reward.amount == 50


class TestTokenBountyRewards:
    @pytest.mark.parametrize(
        ("issue", "amount", "currency"),
        [
            (
                {
                    "labels": ["bounty", "reward:50-mrg"],
                    "title": "[50 MRG] Implement the feature",
                    "body": "## 50 MRG",
                },
                50,
                "MRG",
            ),
            (
                {
                    "labels": ["bounty", "bounty-M"],
                    "title": "Fix CPU mining",
                    "body": "## Bounty\n\n**Tier:** M — 60,000 XTM",
                },
                60000,
                "XTM",
            ),
            (
                {
                    "labels": ["bounty"],
                    "title": "Undo Move mechanic [bounty: 222 XTR]",
                    "body": "",
                },
                222,
                "XTR",
            ),
        ],
    )
    def test_native_reward_is_preserved_without_becoming_usd(
        self, issue, amount, currency
    ):
        reward = get_bounty_reward(issue)
        fields = get_bounty_fields(issue)

        assert reward is not None
        assert reward.amount == amount
        assert reward.currency == currency
        assert fields == {
            "bounty_amount": 0,
            "bounty_currency": currency,
            "bounty_native_amount": amount,
        }
        assert get_bounty_amount(issue) == 0

    def test_usd_reward_retains_legacy_amount_and_currency_metadata(self):
        issue = {
            "labels": ["bounty"],
            "title": "Implement feature",
            "body": "/bounty $1.2k",
        }

        assert get_bounty_fields(issue) == {
            "bounty_amount": 1200,
            "bounty_currency": "USD",
            "bounty_native_amount": 1200,
        }

    def test_unknown_reward_has_no_assumed_currency(self):
        issue = {"labels": ["bounty"], "title": "Fix bug", "body": "No amount"}

        assert get_bounty_fields(issue) == {
            "bounty_amount": 0,
            "bounty_currency": None,
            "bounty_native_amount": 0,
        }


class TestCalculateBountyScore:
    def test_high_value_low_competition(self):
        issue = {"open_pr_count": 0, "comment_count": 2, "updated_at": "2026-04-01T00:00:00Z"}
        intel = {"bounty_amount": 4000, "friction_level": "Low"}
        score, breakdown = calculate_bounty_score(issue, intel)
        assert 60 <= score <= 100
        assert breakdown["competition"] > 90
        assert breakdown["amount"] > 70

    def test_zero_amount_high_competition(self):
        issue = {"open_pr_count": 10, "comment_count": 50, "updated_at": "2024-01-01T00:00:00Z"}
        intel = {"bounty_amount": 0, "friction_level": "High"}
        score, breakdown = calculate_bounty_score(issue, intel)
        assert score < 30
        assert breakdown["amount"] == 0
        assert breakdown["competition"] == 0

    def test_score_in_valid_range(self):
        issue = {"open_pr_count": 1, "comment_count": 5, "updated_at": "2026-03-01T00:00:00Z"}
        intel = {"bounty_amount": 500, "friction_level": "Medium"}
        score, _ = calculate_bounty_score(issue, intel)
        assert 0 <= score <= 100

    def test_breakdown_keys(self):
        issue = {"open_pr_count": 0, "comment_count": 0, "updated_at": "2026-04-01T00:00:00Z"}
        intel = {"bounty_amount": 100, "friction_level": "Low"}
        _, breakdown = calculate_bounty_score(issue, intel)
        assert "amount" in breakdown
        assert "feasibility" in breakdown
        assert "competition" in breakdown
        assert "freshness" in breakdown

    def test_missing_updated_at(self):
        issue = {"open_pr_count": 0, "comment_count": 0, "updated_at": ""}
        intel = {"bounty_amount": 100, "friction_level": "Low"}
        score, breakdown = calculate_bounty_score(issue, intel)
        assert 0 <= score <= 100
        assert breakdown["freshness"] == 50  # default

    def test_friction_levels(self):
        issue = {"open_pr_count": 0, "comment_count": 0, "updated_at": "2026-04-01T00:00:00Z"}
        
        _, low_bd = calculate_bounty_score(issue, {"bounty_amount": 0, "friction_level": "Low"})
        _, med_bd = calculate_bounty_score(issue, {"bounty_amount": 0, "friction_level": "Medium"})
        _, high_bd = calculate_bounty_score(issue, {"bounty_amount": 0, "friction_level": "High"})
        
        assert low_bd["feasibility"] > med_bd["feasibility"] > high_bd["feasibility"]

    def test_competition_decreases_with_prs(self):
        intel = {"bounty_amount": 100, "friction_level": "Low"}
        
        _, bd0 = calculate_bounty_score({"open_pr_count": 0, "comment_count": 0, "updated_at": "2026-04-01T00:00:00Z"}, intel)
        _, bd5 = calculate_bounty_score({"open_pr_count": 5, "comment_count": 0, "updated_at": "2026-04-01T00:00:00Z"}, intel)
        
        assert bd0["competition"] > bd5["competition"]


class TestManualRiskOverrides:
    def test_openpilot_mainline_kernel_bounty_is_downgraded(self):
        issue = {
            "number": 32386,
            "repository": "commaai/openpilot",
            "title": "Ship Ubuntu 24.04 + mainline kernel to master",
        }
        intel = {
            "friction_level": "Low",
            "technical_hint": "Old hint",
            "bounty_tier": "S-Tier",
            "bounty_amount": 1000,
            "is_hidden_gem": True,
            "bounty_score": 80,
            "score_breakdown": {
                "amount": 20,
                "feasibility": 90,
                "competition": 80,
                "freshness": 60,
            },
        }

        adjusted = apply_manual_risk_override(issue, intel)

        assert adjusted["risk_level"] == "Critical"
        assert "vamOS" in adjusted["risk_warning"]
        assert adjusted["friction_level"] == "High"
        assert adjusted["bounty_amount"] == 2000
        assert adjusted["bounty_currency"] == "USD"
        assert adjusted["bounty_native_amount"] == 2000
        assert adjusted["is_hidden_gem"] is False
        assert adjusted["bounty_score"] == 25
        assert adjusted["score_breakdown"]["feasibility"] == 30

    def test_other_bounties_are_unchanged(self):
        issue = {"number": 1, "repository": "org/repo"}
        intel = {"technical_hint": "Keep this"}

        assert apply_manual_risk_override(issue, intel) == intel


class TestLoadExistingIntelligence:
    """Test that load_existing_intelligence uses composite keys to avoid
    cross-repo issue number collisions."""

    def test_same_number_different_repos_preserved(self, tmp_path, monkeypatch):
        """Two issues with the same number but different repos should not
        overwrite each other's intelligence."""
        data = [
            {
                "number": 42,
                "repository": "org/repo-a",
                "title": "Issue A",
                "hunter_intelligence": {
                    "friction_level": "Low",
                    "technical_hint": "Hint A",
                },
            },
            {
                "number": 42,
                "repository": "org/repo-b",
                "title": "Issue B",
                "hunter_intelligence": {
                    "friction_level": "High",
                    "technical_hint": "Hint B",
                },
            },
        ]
        enriched_file = tmp_path / "enriched_bounties.json"
        enriched_file.write_text(json.dumps(data))

        import enrich_bounties
        monkeypatch.setattr(enrich_bounties, "EXISTING_FILE", str(enriched_file))

        intel = load_existing_intelligence()

        assert len(intel) == 2
        assert "org/repo-a#42" in intel
        assert "org/repo-b#42" in intel
        assert intel["org/repo-a#42"]["technical_hint"] == "Hint A"
        assert intel["org/repo-b#42"]["technical_hint"] == "Hint B"

    def test_unique_numbers_still_work(self, tmp_path, monkeypatch):
        """Issues with unique numbers across repos load correctly."""
        data = [
            {
                "number": 1,
                "repository": "org/repo-a",
                "title": "Issue 1",
                "hunter_intelligence": {"friction_level": "Low", "technical_hint": "H1"},
            },
            {
                "number": 2,
                "repository": "org/repo-b",
                "title": "Issue 2",
                "hunter_intelligence": {"friction_level": "Medium", "technical_hint": "H2"},
            },
        ]
        enriched_file = tmp_path / "enriched_bounties.json"
        enriched_file.write_text(json.dumps(data))

        import enrich_bounties
        monkeypatch.setattr(enrich_bounties, "EXISTING_FILE", str(enriched_file))

        intel = load_existing_intelligence()

        assert len(intel) == 2
        assert intel["org/repo-a#1"]["technical_hint"] == "H1"
        assert intel["org/repo-b#2"]["technical_hint"] == "H2"

    def test_no_existing_file(self, tmp_path, monkeypatch):
        """Returns empty dict when enriched file does not exist."""
        import enrich_bounties
        monkeypatch.setattr(
            enrich_bounties, "EXISTING_FILE", str(tmp_path / "nonexistent.json")
        )

        intel = load_existing_intelligence()

        assert intel == {}
