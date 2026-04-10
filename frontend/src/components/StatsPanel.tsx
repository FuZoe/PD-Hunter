"use client";

import { motion, useMotionValue, useTransform, animate } from "framer-motion";
import { useEffect } from "react";

interface StatsPanelProps {
  stats: {
    total: number;
    totalValue: number;
    sTier: number;
    aTier: number;
    bTier: number;
    lowFriction: number;
    hiddenGems: number;
  };
}

function AnimatedNumber({ value, prefix = "" }: { value: number; prefix?: string }) {
  const count = useMotionValue(0);
  const rounded = useTransform(count, (v) => `${prefix}${Math.round(v).toLocaleString()}`);

  useEffect(() => {
    const controls = animate(count, value, {
      duration: 1.2,
      ease: "easeOut",
    });
    return controls.stop;
  }, [value, count]);

  return <motion.span>{rounded}</motion.span>;
}

const statItems = [
  { key: "sTier", label: "S-TIER", color: "text-hacker-yellow" },
  { key: "aTier", label: "A-TIER", color: "text-hacker-purple" },
  { key: "bTier", label: "B-TIER", color: "text-hacker-cyan" },
  { key: "lowFriction", label: "LOW FRICTION", color: "text-hacker-green" },
  { key: "hiddenGems", label: "HIDDEN GEMS", color: "text-hacker-orange" },
] as const;

export default function StatsPanel({ stats }: StatsPanelProps) {
  return (
    <section className="border-b border-hacker-border bg-hacker-card/30">
      <div className="max-w-7xl mx-auto px-6 py-4">
        <div className="grid grid-cols-2 sm:grid-cols-5 gap-4">
          {statItems.map((item, i) => (
            <motion.div
              key={item.key}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.4, delay: i * 0.1 }}
              className="text-center"
            >
              <div className={`text-2xl font-mono font-bold ${item.color}`}>
                <AnimatedNumber value={stats[item.key]} />
              </div>
              <div className="text-hacker-muted text-xs font-mono mt-1">
                {item.label}
              </div>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}
