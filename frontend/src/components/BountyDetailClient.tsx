"use client";

import Link from "next/link";
import { BountyIssue, ScoreBreakdown } from "@/lib/types";
import { formatBounty, formatDate, tierColors, frictionConfig, cn } from "@/lib/utils";
import { motion } from "framer-motion";

interface BountyDetailClientProps {
  bounty: BountyIssue;
}

function ScoreRadar({ breakdown }: { breakdown: ScoreBreakdown }) {
  const size = 200;
  const center = size / 2;
  const radius = 70;
  const labels = [
    { key: "amount" as const, label: "Amount", angle: -90 },
    { key: "feasibility" as const, label: "Feasibility", angle: 0 },
    { key: "competition" as const, label: "Competition", angle: 90 },
    { key: "freshness" as const, label: "Freshness", angle: 180 },
  ];

  const getPoint = (angle: number, value: number) => {
    const r = (value / 100) * radius;
    const rad = (angle * Math.PI) / 180;
    return {
      x: center + r * Math.cos(rad),
      y: center + r * Math.sin(rad),
    };
  };

  const dataPoints = labels.map((l) => getPoint(l.angle, breakdown[l.key]));
  const dataPath = dataPoints.map((p, i) => `${i === 0 ? "M" : "L"} ${p.x} ${p.y}`).join(" ") + " Z";

  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} className="mx-auto">
      {/* Grid lines */}
      {[25, 50, 75, 100].map((v) => {
        const pts = labels.map((l) => getPoint(l.angle, v));
        const gridPath = pts.map((p, i) => `${i === 0 ? "M" : "L"} ${p.x} ${p.y}`).join(" ") + " Z";
        return <path key={v} d={gridPath} fill="none" stroke="#1e1e2e" strokeWidth="1" />;
      })}
      {/* Axes */}
      {labels.map((l) => {
        const end = getPoint(l.angle, 100);
        return <line key={l.key} x1={center} y1={center} x2={end.x} y2={end.y} stroke="#2a2a3e" strokeWidth="1" />;
      })}
      {/* Data polygon */}
      <motion.path
        d={dataPath}
        fill="rgba(0, 255, 136, 0.15)"
        stroke="#00ff88"
        strokeWidth="2"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.8 }}
      />
      {/* Data points */}
      {dataPoints.map((p, i) => (
        <circle key={i} cx={p.x} cy={p.y} r="4" fill="#00ff88" />
      ))}
      {/* Labels */}
      {labels.map((l) => {
        const labelPoint = getPoint(l.angle, 130);
        return (
          <text
            key={l.key}
            x={labelPoint.x}
            y={labelPoint.y}
            textAnchor="middle"
            dominantBaseline="middle"
            className="fill-hacker-muted text-[10px] font-mono"
          >
            {l.label}
          </text>
        );
      })}
      {/* Values */}
      {labels.map((l) => {
        const vp = getPoint(l.angle, 115);
        return (
          <text
            key={`v-${l.key}`}
            x={vp.x}
            y={vp.y + 12}
            textAnchor="middle"
            dominantBaseline="middle"
            className="fill-hacker-green text-[9px] font-mono font-bold"
          >
            {breakdown[l.key]}
          </text>
        );
      })}
    </svg>
  );
}

export default function BountyDetailClient({ bounty }: BountyDetailClientProps) {
  const intel = bounty.hunter_intelligence;
  const tier = tierColors[intel.bounty_tier] || tierColors["B-Tier"];
  const friction = frictionConfig[intel.friction_level];
  const isSTier = intel.bounty_tier === "S-Tier";

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4 }}
      className="max-w-5xl mx-auto px-6 py-8"
    >
      {/* Breadcrumb */}
      <nav className="text-sm font-mono text-hacker-muted mb-6">
        <Link href="/" className="hover:text-hacker-green transition-colors">Home</Link>
        <span className="mx-2">/</span>
        <span className="text-hacker-cyan">{bounty.repository}</span>
        <span className="mx-2">/</span>
        <span className="text-hacker-text">#{bounty.number}</span>
      </nav>

      {/* Title Area */}
      <div className="mb-8">
        <div className="flex items-center gap-3 mb-4 flex-wrap">
          <span className={`px-3 py-1.5 rounded text-sm font-mono font-bold ${tier.badge}`}>
            {intel.bounty_tier}
          </span>
          <span className={`text-sm font-mono ${friction.color}`}>
            {friction.icon} {intel.friction_level} Friction
          </span>
          {intel.is_hidden_gem && (
            <span className="text-sm text-hacker-orange font-mono">
              {"\uD83D\uDC8E"} Hidden Gem
            </span>
          )}
        </div>
        <h1 className="text-2xl md:text-3xl font-bold text-hacker-text leading-tight mb-4">
          {bounty.title}
        </h1>
        <div className="flex items-center gap-4 flex-wrap text-sm font-mono">
          <span className="text-hacker-cyan">{bounty.repository}</span>
          <span className="text-hacker-muted">by {bounty.author}</span>
          <span className="text-hacker-muted">Created {formatDate(bounty.created_at)}</span>
          <span className="text-hacker-muted">Updated {formatDate(bounty.updated_at)}</span>
        </div>
      </div>

      {/* Main Grid */}
      <div className="grid lg:grid-cols-3 gap-8">
        {/* Left: Body + Intelligence */}
        <div className="lg:col-span-2 space-y-6">
          {/* Bounty Amount */}
          <div className={cn(
            "p-6 rounded-xl border",
            isSTier ? "border-hacker-yellow bg-hacker-yellow/5" : "border-hacker-border bg-hacker-card"
          )}>
            <div className="flex items-center justify-between">
              <div>
                <div className="text-hacker-muted text-xs font-mono uppercase mb-1">Bounty Value</div>
                <div className={cn(
                  "text-4xl font-mono font-bold",
                  isSTier ? "text-hacker-yellow" : "text-hacker-green glow-green"
                )}>
                  {formatBounty(intel.bounty_amount)}
                </div>
              </div>
              {intel.bounty_score !== undefined && (
                <div className="text-center">
                  <div className={cn(
                    "w-16 h-16 rounded-full flex items-center justify-center text-xl font-mono font-bold border-2",
                    intel.bounty_score >= 75 ? "border-hacker-green text-hacker-green" :
                    intel.bounty_score >= 50 ? "border-hacker-yellow text-hacker-yellow" :
                    "border-hacker-red text-hacker-red"
                  )}>
                    {intel.bounty_score}
                  </div>
                  <div className="text-hacker-muted text-[10px] font-mono mt-1">SCORE</div>
                </div>
              )}
            </div>
          </div>

          {/* Expert Intelligence */}
          <div className="p-6 rounded-xl border border-hacker-border bg-hacker-card">
            <div className="flex items-center gap-2 mb-4">
              <span className="text-lg">{"\uD83E\uDDE0"}</span>
              <span className="text-sm font-mono text-hacker-purple uppercase font-bold">
                Expert Intelligence
              </span>
            </div>
            <p className="text-hacker-text leading-relaxed">
              {intel.technical_hint}
            </p>
          </div>

          {/* Issue Body */}
          <div className="p-6 rounded-xl border border-hacker-border bg-hacker-card">
            <h3 className="text-sm font-mono text-hacker-muted uppercase mb-4">Issue Description</h3>
            <div className="text-hacker-text leading-relaxed text-sm whitespace-pre-wrap break-words max-h-96 overflow-y-auto scrollbar-thin">
              {bounty.body || "No description provided."}
            </div>
          </div>

          {/* Labels */}
          {bounty.labels.length > 0 && (
            <div className="p-6 rounded-xl border border-hacker-border bg-hacker-card">
              <h3 className="text-sm font-mono text-hacker-muted uppercase mb-3">Labels</h3>
              <div className="flex flex-wrap gap-2">
                {bounty.labels.map((label) => (
                  <span key={label} className="px-3 py-1 rounded-full text-xs font-mono bg-hacker-border text-hacker-text">
                    {label}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Right Sidebar */}
        <div className="space-y-6">
          {/* Score Radar */}
          {intel.score_breakdown && (
            <div className="p-6 rounded-xl border border-hacker-border bg-hacker-card">
              <h3 className="text-sm font-mono text-hacker-muted uppercase mb-4 text-center">Score Breakdown</h3>
              <ScoreRadar breakdown={intel.score_breakdown} />
            </div>
          )}

          {/* Competition Analysis */}
          <div className="p-6 rounded-xl border border-hacker-border bg-hacker-card">
            <h3 className="text-sm font-mono text-hacker-muted uppercase mb-4">Competition</h3>
            <div className="space-y-3">
              <div className="flex justify-between items-center">
                <span className="text-sm font-mono text-hacker-text">Open PRs</span>
                <span className={cn(
                  "text-lg font-mono font-bold",
                  (bounty.open_pr_count || 0) === 0 ? "text-hacker-green" :
                  (bounty.open_pr_count || 0) <= 3 ? "text-hacker-yellow" : "text-hacker-red"
                )}>
                  {bounty.open_pr_count || 0}
                </span>
              </div>
              <div className="w-full bg-hacker-border rounded-full h-2">
                <div
                  className="bg-hacker-cyan rounded-full h-2 transition-all"
                  style={{ width: `${Math.min(100, (bounty.open_pr_count || 0) * 10)}%` }}
                />
              </div>
              <div className="flex justify-between items-center mt-4">
                <span className="text-sm font-mono text-hacker-text">Comments</span>
                <span className="text-lg font-mono font-bold text-hacker-cyan">
                  {bounty.comment_count}
                </span>
              </div>
              <div className="w-full bg-hacker-border rounded-full h-2">
                <div
                  className="bg-hacker-purple rounded-full h-2 transition-all"
                  style={{ width: `${Math.min(100, bounty.comment_count * 2)}%` }}
                />
              </div>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="space-y-3">
            <a
              href={bounty.url}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center justify-center gap-2 w-full px-6 py-3 rounded-xl bg-hacker-green text-black font-mono font-bold text-sm hover:bg-hacker-green/80 transition-colors"
            >
              HUNT THIS BOUNTY &rarr;
            </a>
            <Link
              href="/"
              className="flex items-center justify-center gap-2 w-full px-6 py-3 rounded-xl border border-hacker-border text-hacker-muted font-mono text-sm hover:text-hacker-text hover:border-hacker-green transition-colors"
            >
              &larr; Back to Dashboard
            </Link>
          </div>
        </div>
      </div>
    </motion.div>
  );
}
