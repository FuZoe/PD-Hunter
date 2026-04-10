"""Tests for enrich_bounties.py core functions."""
import sys
import os
import pytest

# Add project root to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from enrich_bounties import (
    extract_amount_from_text,
    get_bounty_amount,
    get_bounty_tier,
    is_hidden_gem,
    calculate_bounty_score,
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

    def test_b_tier_low(self):
        assert get_bounty_tier(50) == "B-Tier"

    def test_b_tier_boundary(self):
        assert get_bounty_tier(199) == "B-Tier"


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
