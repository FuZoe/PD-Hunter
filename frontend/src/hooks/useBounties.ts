"use client";

import { useState, useEffect } from "react";
import { BountyIssue } from "@/lib/types";
import { fetchBounties } from "@/lib/api";

export function useBounties() {
  const [bounties, setBounties] = useState<BountyIssue[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchBounties().then((data) => {
      setBounties(data);
      setLoading(false);
    });
  }, []);

  const stats = {
    total: bounties.length,
    totalValue: bounties.reduce(
      (sum, b) => sum + b.hunter_intelligence.bounty_amount,
      0
    ),
    sTier: bounties.filter((b) => b.hunter_intelligence.bounty_tier === "S-Tier")
      .length,
    aTier: bounties.filter((b) => b.hunter_intelligence.bounty_tier === "A-Tier")
      .length,
    bTier: bounties.filter((b) => b.hunter_intelligence.bounty_tier === "B-Tier")
      .length,
    lowFriction: bounties.filter(
      (b) => b.hunter_intelligence.friction_level === "Low"
    ).length,
    hiddenGems: bounties.filter(
      (b) => b.hunter_intelligence.is_hidden_gem === true
    ).length,
  };

  return { bounties, loading, stats };
}
