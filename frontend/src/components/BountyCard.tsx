"use client";

import { BountyIssue } from "@/lib/types";
import { formatBounty, formatDate, tierColors, frictionConfig, cn } from "@/lib/utils";

interface BountyCardProps {
  bounty: BountyIssue;
}

export default function BountyCard({ bounty }: BountyCardProps) {
  const intel = bounty.hunter_intelligence;
  const tier = tierColors[intel.bounty_tier] || tierColors["B-Tier"];
  const friction = frictionConfig[intel.friction_level];
  const isSTier = intel.bounty_tier === "S-Tier";
  const repoName = bounty.repository.split("/")[1];
  const updatedDate = formatDate(bounty.updated_at);

  return (
    <div
      className={cn(
        "bg-hacker-card rounded-xl border overflow-hidden card-glow transition-all duration-300",
        isSTier ? "border-hacker-yellow s-tier-glow" : "border-hacker-border",
        friction.class
      )}
    >
      {/* Card Header */}
      <div className="p-5 border-b border-hacker-border">
        <div className="flex items-start justify-between gap-3 mb-3">
          <div className="flex items-center gap-2">
            <span
              className={`px-2 py-1 rounded text-xs font-mono font-bold ${tier.badge}`}
            >
              {intel.bounty_tier}
            </span>
            <span className="px-2 py-1 rounded text-xs font-mono bg-hacker-border text-hacker-muted">
              #{bounty.number}
            </span>
          </div>
          <div className="text-right">
            <div
              className={cn(
                "text-2xl font-mono font-bold",
                isSTier ? "text-hacker-yellow glow-green" : "text-hacker-green"
              )}
            >
              {formatBounty(intel.bounty_amount)}
            </div>
          </div>
        </div>
        <h3 className="font-semibold text-hacker-text leading-tight line-clamp-2 mb-2">
          {bounty.title}
        </h3>
        <div className="flex items-center gap-2 text-sm">
          <span className="text-hacker-cyan font-mono">{repoName}</span>
          <span className="text-hacker-muted">&bull;</span>
          <span className="text-hacker-muted">{updatedDate}</span>
        </div>
      </div>

      {/* Expert Intelligence */}
      <div className="p-5 bg-hacker-bg/50">
        <div className="flex items-center gap-2 mb-3">
          <span className="text-sm">{"\uD83E\uDDE0"}</span>
          <span className="text-xs font-mono text-hacker-purple uppercase">
            Expert Intelligence
          </span>
        </div>
        <p className="text-sm text-hacker-text leading-relaxed">
          {intel.technical_hint}
        </p>
      </div>

      {/* Card Footer */}
      <div className="px-5 py-4 border-t border-hacker-border flex items-center justify-between">
        <div className="flex items-center gap-3 flex-wrap">
          <span className={`text-xs font-mono ${friction.color}`}>
            {friction.icon} {intel.friction_level} Friction
          </span>
          <span className="text-xs text-hacker-muted">&bull;</span>
          <span className="text-xs text-hacker-muted font-mono">
            {bounty.comment_count} comments
          </span>
          <span className="text-xs text-hacker-muted">&bull;</span>
          <span className="text-xs text-hacker-muted font-mono">
            {bounty.open_pr_count || 0} PRs
          </span>
          {intel.is_hidden_gem && (
            <span className="text-xs text-hacker-orange font-mono ml-2">
              {"\uD83D\uDC8E"} GEM
            </span>
          )}
        </div>
        <a
          href={bounty.url}
          target="_blank"
          rel="noopener noreferrer"
          className="px-3 py-1.5 rounded-lg bg-hacker-green/10 text-hacker-green text-xs font-mono font-bold hover:bg-hacker-green hover:text-black transition-colors shrink-0"
        >
          HUNT &rarr;
        </a>
      </div>
    </div>
  );
}
