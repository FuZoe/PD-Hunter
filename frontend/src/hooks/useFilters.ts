"use client";

import { useState, useMemo, useCallback } from "react";
import Fuse from "fuse.js";
import { BountyIssue, TierFilter, SortOption } from "@/lib/types";

interface GemThresholds {
  maxPR: number;
  maxComments: number;
}

export function useFilters(bounties: BountyIssue[]) {
  const [tierFilter, setTierFilter] = useState<TierFilter>("all");
  const [sortOption, setSortOption] = useState<SortOption>("score-desc");
  const [hiddenGemsMode, setHiddenGemsMode] = useState(false);
  const [gemThresholds, setGemThresholds] = useState<GemThresholds>({
    maxPR: 3,
    maxComments: 10,
  });
  const [searchQuery, setSearchQuery] = useState("");

  const toggleHiddenGems = useCallback(() => {
    setHiddenGemsMode((prev) => !prev);
  }, []);

  const fuse = useMemo(() => {
    return new Fuse(bounties, {
      keys: [
        { name: "title", weight: 0.3 },
        { name: "repository", weight: 0.2 },
        { name: "labels", weight: 0.15 },
        { name: "hunter_intelligence.technical_hint", weight: 0.2 },
        { name: "body", weight: 0.15 },
      ],
      threshold: 0.4,
      includeScore: true,
    });
  }, [bounties]);

  const filtered = useMemo(() => {
    let result = [...bounties];

    // Fuse.js fuzzy search
    if (searchQuery.trim()) {
      const fuseResults = fuse.search(searchQuery);
      result = fuseResults.map((r) => r.item);
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
      case "score-desc":
        result.sort(
          (a, b) =>
            (b.hunter_intelligence.bounty_score ?? 0) -
            (a.hunter_intelligence.bounty_score ?? 0)
        );
        break;
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
  }, [bounties, tierFilter, sortOption, hiddenGemsMode, gemThresholds, searchQuery, fuse]);

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
