"use client";

import Link from "next/link";
import { BountyIssue } from "@/lib/types";
import { formatBounty, formatDate, tierColors, frictionConfig, cn } from "@/lib/utils";
import { motion } from "framer-motion";

interface BountyCardProps {
  bounty: BountyIssue;
  index?: number;
}

function ScoreBadge({ score }: { score: number }) {
  const color =
    score >= 75
      ? "border-hacker-green text-hacker-green"
      : score >= 50
      ? "border-hacker-yellow text-hacker-yellow"
      : "border-hacker-red text-hacker-red";

  return (
    <div
      className={cn(
        "w-10 h-10 rounded-full flex items-center justify-center text-xs font-mono font-bold border-2",
        color
      )}
      title={`BountyScore: ${score}/100`}
    >
      {score}
    </div>
  );
}

export default function BountyCard({ bounty, index = 0 }: BountyCardProps) {
  const intel = bounty.hunter_intelligence;
  const tier = tierColors[intel.bounty_tier] || tierColors["B-Tier"];
  const friction = frictionConfig[intel.friction_level];
  const isSTier = intel.bounty_tier === "S-Tier";
  const repoName = bounty.repository.split("/")[1];
  const updatedDate = formatDate(bounty.updated_at);

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, delay: index * 0.05 }}
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
          <div className="flex items-center gap-3">
            {intel.bounty_score !== undefined && (
              <ScoreBadge score={intel.bounty_score} />
            )}
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
        </div>
        <Link
          href={`/bounty/${bounty.number}`}
          className="block"
        >
          <h3 className="font-semibold text-hacker-text leading-tight line-clamp-2 mb-2 hover:text-hacker-cyan transition-colors cursor-pointer">
            {bounty.title}
          </h3>
        </Link>
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
    </motion.div>
  );
}
