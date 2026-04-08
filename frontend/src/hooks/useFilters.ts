"use client";

import { useState, useMemo, useCallback } from "react";
import { BountyIssue, TierFilter, SortOption } from "@/lib/types";

interface GemThresholds {
  maxPR: number;
  maxComments: number;
}

export function useFilters(bounties: BountyIssue[]) {
  const [tierFilter, setTierFilter] = useState<TierFilter>("all");
  const [sortOption, setSortOption] = useState<SortOption>("bounty-desc");
  const [hiddenGemsMode, setHiddenGemsMode] = useState(false);
  const [gemThresholds, setGemThresholds] = useState<GemThresholds>({
    maxPR: 3,
    maxComments: 10,
  });
  const [searchQuery, setSearchQuery] = useState("");

  const toggleHiddenGems = useCallback(() => {
    setHiddenGemsMode((prev) => !prev);
  }, []);

  const filtered = useMemo(() => {
    let result = [...bounties];

    // Search filter
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      result = result.filter(
        (b) =>
          b.title.toLowerCase().includes(q) ||
          b.repository.toLowerCase().includes(q) ||
          b.labels.some((l) => l.toLowerCase().includes(q)) ||
          b.hunter_intelligence.technical_hint.toLowerCase().includes(q) ||
          b.body?.toLowerCase().includes(q)
      );
    }

    // Hidden gems filter
    if (hiddenGemsMode) {
      result = result.filter(
        (b) =>
          (b.open_pr_count || 0) <= gemThresholds.maxPR &&
          b.comment_count <= gemThresholds.maxComments
      );
    }

    // Tier filter
    if (tierFilter !== "all") {
      result = result.filter(
        (b) => b.hunter_intelligence.bounty_tier === tierFilter
      );
    }

    // Sort
    const frictionOrder = { Low: 1, Medium: 2, High: 3 };
    switch (sortOption) {
      case "bounty-desc":
        result.sort(
          (a, b) =>
            b.hunter_intelligence.bounty_amount -
            a.hunter_intelligence.bounty_amount
        );
        break;
      case "bounty-asc":
        result.sort(
          (a, b) =>
            a.hunter_intelligence.bounty_amount -
            b.hunter_intelligence.bounty_amount
        );
        break;
      case "friction-asc":
        result.sort(
          (a, b) =>
            frictionOrder[a.hunter_intelligence.friction_level] -
            frictionOrder[b.hunter_intelligence.friction_level]
        );
        break;
      case "date-desc":
        result.sort(
          (a, b) =>
            new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
        );
        break;
    }

    return result;
  }, [bounties, tierFilter, sortOption, hiddenGemsMode, gemThresholds, searchQuery]);

  return {
    filtered,
    tierFilter,
    setTierFilter,
    sortOption,
    setSortOption,
    hiddenGemsMode,
    toggleHiddenGems,
    gemThresholds,
    setGemThresholds,
    searchQuery,
    setSearchQuery,
  };
}
