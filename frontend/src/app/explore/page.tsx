"use client";

import { useBounties } from "@/hooks/useBounties";
import BountyCard from "@/components/BountyCard";
import Link from "next/link";
import { motion } from "framer-motion";
import { useMemo, useState } from "react";

interface OrgGroup {
  name: string;
  bounties: ReturnType<typeof useBounties>["bounties"];
  totalValue: number;
}

export default function ExplorePage() {
  const { bounties, loading } = useBounties();
  const [expandedOrg, setExpandedOrg] = useState<string | null>(null);

  const orgGroups = useMemo(() => {
    const groups: Record<string, OrgGroup> = {};
    bounties.forEach((b) => {
      const org = b.repository.split("/")[0];
      if (!groups[org]) {
        groups[org] = { name: org, bounties: [], totalValue: 0 };
      }
      groups[org].bounties.push(b);
      groups[org].totalValue += b.hunter_intelligence.bounty_amount;
    });
    return Object.values(groups).sort((a, b) => b.totalValue - a.totalValue);
  }, [bounties]);

  const tierDistribution = useMemo(() => {
    const dist = { "S-Tier": 0, "A-Tier": 0, "B-Tier": 0 };
    bounties.forEach((b) => {
      dist[b.hunter_intelligence.bounty_tier]++;
    });
    return dist;
  }, [bounties]);

  const frictionDistribution = useMemo(() => {
    const dist = { Low: 0, Medium: 0, High: 0 };
    bounties.forEach((b) => {
      dist[b.hunter_intelligence.friction_level]++;
    });
    return dist;
  }, [bounties]);

  const amountRanges = useMemo(() => {
    const ranges = [
      { label: "$0", count: 0 },
      { label: "$1-100", count: 0 },
      { label: "$100-500", count: 0 },
      { label: "$500-1K", count: 0 },
      { label: "$1K+", count: 0 },
    ];
    bounties.forEach((b) => {
      const amt = b.hunter_intelligence.bounty_amount;
      if (amt === 0) ranges[0].count++;
      else if (amt <= 100) ranges[1].count++;
      else if (amt <= 500) ranges[2].count++;
      else if (amt <= 1000) ranges[3].count++;
      else ranges[4].count++;
    });
    return ranges;
  }, [bounties]);

  const maxBarCount = Math.max(...amountRanges.map((r) => r.count), 1);

  if (loading) {
    return (
      <div className="text-center py-20">
        <div className="text-hacker-green font-mono text-lg">
          Loading bounties<span className="animate-blink">{"\u2588"}</span>
        </div>
      </div>
    );
  }

  return (
    <>
      {/* Header */}
      <header className="border-b border-hacker-border bg-hacker-card/50 backdrop-blur-sm sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center gap-4">
            <span className="text-2xl">{"\uD83C\uDFAF"}</span>
            <h1 className="text-xl font-mono font-bold text-hacker-green glow-green">PD-HUNTER</h1>
            <span className="text-hacker-muted font-mono text-sm">v2.0.0</span>
            <nav className="hidden sm:flex items-center gap-4 ml-6">
              <Link href="/" className="text-hacker-muted font-mono text-sm hover:text-hacker-green transition-colors">
                Dashboard
              </Link>
              <Link href="/explore" className="text-hacker-green font-mono text-sm border-b border-hacker-green pb-0.5">
                Explore
              </Link>
            </nav>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-8">
        <motion.h2
          className="text-2xl font-mono font-bold text-hacker-text mb-8"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
        >
          {"\uD83D\uDCCA"} Explore Bounties
        </motion.h2>

        {/* Stats Charts Grid */}
        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6 mb-12">
          {/* Tier Distribution */}
          <motion.div
            className="p-6 rounded-xl border border-hacker-border bg-hacker-card"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.1 }}
          >
            <h3 className="text-sm font-mono text-hacker-muted uppercase mb-4">Tier Distribution</h3>
            <div className="space-y-3">
              {Object.entries(tierDistribution).map(([tier, count]) => {
                const colors: Record<string, string> = {
                  "S-Tier": "bg-hacker-yellow",
                  "A-Tier": "bg-hacker-purple",
                  "B-Tier": "bg-hacker-cyan",
                };
                const textColors: Record<string, string> = {
                  "S-Tier": "text-hacker-yellow",
                  "A-Tier": "text-hacker-purple",
                  "B-Tier": "text-hacker-cyan",
                };
                return (
                  <div key={tier}>
                    <div className="flex justify-between text-sm font-mono mb-1">
                      <span className={textColors[tier]}>{tier}</span>
                      <span className="text-hacker-text">{count}</span>
                    </div>
                    <div className="w-full bg-hacker-border rounded-full h-3">
                      <motion.div
                        className={`${colors[tier]} rounded-full h-3`}
                        initial={{ width: 0 }}
                        animate={{ width: `${bounties.length ? (count / bounties.length) * 100 : 0}%` }}
                        transition={{ duration: 0.8, delay: 0.3 }}
                      />
                    </div>
                  </div>
                );
              })}
            </div>
          </motion.div>

          {/* Friction Distribution */}
          <motion.div
            className="p-6 rounded-xl border border-hacker-border bg-hacker-card"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
          >
            <h3 className="text-sm font-mono text-hacker-muted uppercase mb-4">Friction Levels</h3>
            <div className="space-y-3">
              {Object.entries(frictionDistribution).map(([level, count]) => {
                const colors: Record<string, string> = {
                  Low: "bg-hacker-green",
                  Medium: "bg-hacker-yellow",
                  High: "bg-hacker-red",
                };
                const textColors: Record<string, string> = {
                  Low: "text-hacker-green",
                  Medium: "text-hacker-yellow",
                  High: "text-hacker-red",
                };
                return (
                  <div key={level}>
                    <div className="flex justify-between text-sm font-mono mb-1">
                      <span className={textColors[level]}>{level}</span>
                      <span className="text-hacker-text">{count}</span>
                    </div>
                    <div className="w-full bg-hacker-border rounded-full h-3">
                      <motion.div
                        className={`${colors[level]} rounded-full h-3`}
                        initial={{ width: 0 }}
                        animate={{ width: `${bounties.length ? (count / bounties.length) * 100 : 0}%` }}
                        transition={{ duration: 0.8, delay: 0.4 }}
                      />
                    </div>
                  </div>
                );
              })}
            </div>
          </motion.div>

          {/* Amount Distribution */}
          <motion.div
            className="p-6 rounded-xl border border-hacker-border bg-hacker-card"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.3 }}
          >
            <h3 className="text-sm font-mono text-hacker-muted uppercase mb-4">Bounty Amounts</h3>
            <div className="space-y-2">
              {amountRanges.map((range) => (
                <div key={range.label} className="flex items-center gap-3">
                  <span className="text-xs font-mono text-hacker-muted w-16 text-right">{range.label}</span>
                  <div className="flex-1 bg-hacker-border rounded-full h-3">
                    <motion.div
                      className="bg-hacker-green rounded-full h-3"
                      initial={{ width: 0 }}
                      animate={{ width: `${(range.count / maxBarCount) * 100}%` }}
                      transition={{ duration: 0.8, delay: 0.5 }}
                    />
                  </div>
                  <span className="text-xs font-mono text-hacker-text w-6">{range.count}</span>
                </div>
              ))}
            </div>
          </motion.div>
        </div>

        {/* Organizations */}
        <h2 className="text-xl font-mono font-bold text-hacker-text mb-6">
          {"\uD83C\uDFE2"} By Organization ({orgGroups.length})
        </h2>
        <div className="space-y-4">
          {orgGroups.map((org, orgIdx) => (
            <motion.div
              key={org.name}
              className="rounded-xl border border-hacker-border bg-hacker-card overflow-hidden"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: orgIdx * 0.05 }}
            >
              <button
                onClick={() => setExpandedOrg(expandedOrg === org.name ? null : org.name)}
                className="w-full px-6 py-4 flex items-center justify-between hover:bg-hacker-bg/50 transition-colors"
              >
                <div className="flex items-center gap-4">
                  <span className="text-lg font-mono font-bold text-hacker-cyan">{org.name}</span>
                  <span className="text-sm font-mono text-hacker-muted">
                    {org.bounties.length} bounties
                  </span>
                </div>
                <div className="flex items-center gap-4">
                  <span className="text-hacker-green font-mono font-bold">
                    ${org.totalValue.toLocaleString()}
                  </span>
                  <span className="text-hacker-muted text-lg">
                    {expandedOrg === org.name ? "▼" : "▶"}
                  </span>
                </div>
              </button>
              {expandedOrg === org.name && (
                <div className="px-6 pb-6">
                  <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-4 pt-4 border-t border-hacker-border">
                    {org.bounties.map((b, i) => (
                      <BountyCard key={b.url} bounty={b} index={i} />
                    ))}
                  </div>
                </div>
              )}
            </motion.div>
          ))}
        </div>
      </main>
    </>
  );
}
