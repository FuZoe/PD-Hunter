"use client";

import Link from "next/link";
import { BountyIssue } from "@/lib/types";
import { bountySlug, formatBounty, tierColors, cn } from "@/lib/utils";

interface BountyDisambiguationProps {
  matches: BountyIssue[];
  number: string;
}

export default function BountyDisambiguation({
  matches,
  number,
}: BountyDisambiguationProps) {
  return (
    <div className="max-w-3xl mx-auto px-6 py-8">
      <nav className="text-sm font-mono text-hacker-muted mb-6">
        <Link
          href="/"
          className="hover:text-hacker-green transition-colors"
        >
          Home
        </Link>
        <span className="mx-2">/</span>
        <span className="text-hacker-text">#{number}</span>
      </nav>

      <h1 className="text-xl font-mono font-bold text-hacker-text mb-2">
        Multiple bounties share issue #{number}
      </h1>
      <p className="text-sm font-mono text-hacker-muted mb-6">
        Select the repository you are looking for:
      </p>

      <div className="space-y-3">
        {matches.map((b) => {
          const tier = tierColors[b.hunter_intelligence.bounty_tier] || tierColors["B-Tier"];
          return (
            <Link
              key={bountySlug(b.repository, b.number)}
              href={`/bounty/${bountySlug(b.repository, b.number)}`}
              className={cn(
                "block p-4 rounded-lg border border-hacker-border bg-hacker-card",
                "hover:border-hacker-green transition-colors"
              )}
            >
              <div className="flex items-center justify-between">
                <div>
                  <span className="text-hacker-cyan font-mono text-sm">
                    {b.repository}
                  </span>
                  <h2 className="text-hacker-text font-semibold mt-1 line-clamp-1">
                    {b.title}
                  </h2>
                </div>
                <div className="flex items-center gap-3 shrink-0 ml-4">
                  <span
                    className={`px-2 py-1 rounded text-xs font-mono font-bold ${tier.badge}`}
                  >
                    {b.hunter_intelligence.bounty_tier}
                  </span>
                  <span className="text-hacker-green font-mono font-bold">
                    {formatBounty(b.hunter_intelligence.bounty_amount)}
                  </span>
                </div>
              </div>
            </Link>
          );
        })}
      </div>
    </div>
  );
}
