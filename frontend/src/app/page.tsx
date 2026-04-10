"use client";

import { useBounties } from "@/hooks/useBounties";
import { useFilters } from "@/hooks/useFilters";
import StatsPanel from "@/components/StatsPanel";
import FilterBar from "@/components/FilterBar";
import BountyCard from "@/components/BountyCard";
import Link from "next/link";
import { motion } from "framer-motion";

export default function Home() {
  const { bounties, loading, stats } = useBounties();
  const {
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
  } = useFilters(bounties);

  const sTierBounties = filtered.filter(
    (b) => b.hunter_intelligence.bounty_tier === "S-Tier"
  );
  const otherBounties = filtered.filter(
    (b) => b.hunter_intelligence.bounty_tier !== "S-Tier"
  );
  const showFeatured = tierFilter === "all" && sTierBounties.length > 0;

  return (
    <>
      {/* Header */}
      <header className="border-b border-hacker-border bg-hacker-card/50 backdrop-blur-sm sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <span className="text-2xl">{"\uD83C\uDFAF"}</span>
                <h1 className="text-xl font-mono font-bold text-hacker-green glow-green">
                  PD-HUNTER
                </h1>
              </div>
              <span className="text-hacker-muted font-mono text-sm">v2.0.0</span>
              <nav className="hidden sm:flex items-center gap-4 ml-6">
                <Link href="/" className="text-hacker-green font-mono text-sm border-b border-hacker-green pb-0.5">
                  Dashboard
                </Link>
                <Link href="/explore" className="text-hacker-muted font-mono text-sm hover:text-hacker-green transition-colors">
                  Explore
                </Link>
              </nav>
            </div>
            <div className="flex items-center gap-6">
              <div className="text-right">
                <div className="text-hacker-muted text-xs font-mono">
                  TOTAL BOUNTIES
                </div>
                <div className="text-hacker-cyan font-mono font-bold text-lg">
                  {loading ? "--" : stats.total}
                </div>
              </div>
              <div className="text-right">
                <div className="text-hacker-muted text-xs font-mono">
                  TOTAL VALUE
                </div>
                <div className="text-hacker-green font-mono font-bold text-lg glow-green">
                  {loading ? "$--" : `$${stats.totalValue.toLocaleString()}`}
                </div>
              </div>
            </div>
          </div>
        </div>
      </header>

      {/* Stats */}
      <StatsPanel stats={stats} />

      {/* Filters */}
      <FilterBar
        tierFilter={tierFilter}
        setTierFilter={setTierFilter}
        sortOption={sortOption}
        setSortOption={setSortOption}
        hiddenGemsMode={hiddenGemsMode}
        toggleHiddenGems={toggleHiddenGems}
        gemThresholds={gemThresholds}
        setGemThresholds={setGemThresholds}
        searchQuery={searchQuery}
        setSearchQuery={setSearchQuery}
        visibleCount={filtered.length}
      />

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-6 py-8 flex-1">
        {loading ? (
          <div className="text-center py-20">
            <div className="text-hacker-green font-mono text-lg">
              Loading bounties
              <span className="animate-blink">{"\u2588"}</span>
            </div>
          </div>
        ) : filtered.length === 0 ? (
          <div className="text-center py-20">
            <div className="text-hacker-muted font-mono">
              No bounties found matching your criteria.
            </div>
          </div>
        ) : (
          <>
            {/* Featured S-Tier Section */}
            {showFeatured && (
              <motion.section
                className="mb-10"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ duration: 0.5 }}
              >
                <div className="flex items-center gap-3 mb-6">
                  <span className="text-2xl">{"\u26A1"}</span>
                  <h2 className="text-xl font-mono font-bold text-hacker-yellow">
                    FEATURED: S-TIER OPPORTUNITIES
                  </h2>
                </div>
                <div className="grid gap-6">
                  {sTierBounties.map((b, i) => (
                    <BountyCard key={b.url} bounty={b} index={i} />
                  ))}
                </div>
              </motion.section>
            )}

            {/* All Bounties */}
            <section>
              <div className="flex items-center gap-3 mb-6">
                <span className="text-xl">{"\uD83D\uDC8E"}</span>
                <h2 className="text-lg font-mono font-bold text-hacker-text">
                  HUNTER CARDS
                </h2>
                <span className="text-hacker-muted text-sm font-mono">
                  ({filtered.length})
                </span>
              </div>
              <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
                {(showFeatured ? otherBounties : filtered).map((b, i) => (
                  <BountyCard key={b.url} bounty={b} index={i} />
                ))}
              </div>
            </section>
          </>
        )}
      </main>
    </>
  );
}
