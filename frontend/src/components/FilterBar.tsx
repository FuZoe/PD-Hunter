"use client";

import { TierFilter, SortOption } from "@/lib/types";
import { cn } from "@/lib/utils";
import { Search } from "lucide-react";

interface FilterBarProps {
  tierFilter: TierFilter;
  setTierFilter: (filter: TierFilter) => void;
  sortOption: SortOption;
  setSortOption: (option: SortOption) => void;
  hiddenGemsMode: boolean;
  toggleHiddenGems: () => void;
  gemThresholds: { maxPR: number; maxComments: number };
  setGemThresholds: (t: { maxPR: number; maxComments: number }) => void;
  searchQuery: string;
  setSearchQuery: (q: string) => void;
  visibleCount: number;
}

const tierButtons: { filter: TierFilter; label: string; hoverBorder: string }[] = [
  { filter: "all", label: "ALL", hoverBorder: "hover:border-hacker-green" },
  { filter: "S-Tier", label: "S-TIER", hoverBorder: "hover:border-hacker-yellow" },
  { filter: "A-Tier", label: "A-TIER", hoverBorder: "hover:border-hacker-purple" },
  { filter: "B-Tier", label: "B-TIER", hoverBorder: "hover:border-hacker-cyan" },
];

export default function FilterBar({
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
  visibleCount,
}: FilterBarProps) {
  return (
    <section className="border-b border-hacker-border bg-hacker-bg">
      <div className="max-w-7xl mx-auto px-6 py-4 space-y-4">
        {/* Search Bar */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-hacker-muted" />
          <input
            type="text"
            placeholder="Search bounties by title, repo, label, or hint..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2.5 rounded-lg bg-hacker-card border border-hacker-border text-sm font-mono text-hacker-text placeholder:text-hacker-muted focus:border-hacker-green focus:outline-none transition-colors"
          />
          {searchQuery && (
            <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs font-mono text-hacker-muted">
              {visibleCount} results
            </span>
          )}
        </div>

        {/* Filter Row */}
        <div className="flex flex-wrap items-center gap-4">
          <span className="text-hacker-muted text-sm font-mono">FILTER:</span>
          {tierButtons.map((btn) => (
            <button
              key={btn.filter}
              onClick={() => setTierFilter(btn.filter)}
              className={cn(
                "px-4 py-2 rounded-lg bg-hacker-card border border-hacker-border text-sm font-mono transition-colors",
                btn.hoverBorder,
                tierFilter === btn.filter &&
                  "border-hacker-green text-hacker-green"
              )}
            >
              {btn.label}
            </button>
          ))}

          <div className="h-6 w-px bg-hacker-border" />

          <button
            onClick={toggleHiddenGems}
            className={cn(
              "px-4 py-2 rounded-lg bg-hacker-card border border-hacker-border text-sm font-mono hover:border-hacker-orange transition-colors",
              hiddenGemsMode &&
                "border-hacker-orange text-hacker-orange bg-hacker-orange/10"
            )}
          >
            {"\uD83D\uDC8E"} HIDDEN GEMS
          </button>

          {hiddenGemsMode && (
            <div className="flex items-center gap-2">
              <span className="text-hacker-muted text-xs font-mono">
                PR&le;
              </span>
              <input
                type="number"
                value={gemThresholds.maxPR}
                min={0}
                max={100}
                onChange={(e) =>
                  setGemThresholds({
                    ...gemThresholds,
                    maxPR: parseInt(e.target.value) || 3,
                  })
                }
                className="w-14 px-2 py-1 rounded bg-hacker-card border border-hacker-border text-sm font-mono text-hacker-text focus:border-hacker-orange outline-none"
              />
              <span className="text-hacker-muted text-xs font-mono">
                Comments&le;
              </span>
              <input
                type="number"
                value={gemThresholds.maxComments}
                min={0}
                max={200}
                onChange={(e) =>
                  setGemThresholds({
                    ...gemThresholds,
                    maxComments: parseInt(e.target.value) || 10,
                  })
                }
                className="w-14 px-2 py-1 rounded bg-hacker-card border border-hacker-border text-sm font-mono text-hacker-text focus:border-hacker-orange outline-none"
              />
            </div>
          )}

          <div className="flex-1" />

          <select
            value={sortOption}
            onChange={(e) => setSortOption(e.target.value as SortOption)}
            className="px-4 py-2 rounded-lg bg-hacker-card border border-hacker-border text-sm font-mono text-hacker-text focus:border-hacker-green outline-none"
          >
            <option value="bounty-desc">Bounty: High &rarr; Low</option>
            <option value="bounty-asc">Bounty: Low &rarr; High</option>
            <option value="friction-asc">Friction: Low &rarr; High</option>
            <option value="date-desc">Date: Newest</option>
          </select>
        </div>
      </div>
    </section>
  );
}
