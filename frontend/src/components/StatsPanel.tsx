"use client";

interface StatsPanelProps {
  stats: {
    sTier: number;
    aTier: number;
    bTier: number;
    lowFriction: number;
    hiddenGems: number;
  };
}

const statItems = [
  {
    label: "S-TIER",
    key: "sTier" as const,
    colorBg: "bg-hacker-yellow/20",
    colorText: "text-hacker-yellow",
    display: "S",
  },
  {
    label: "A-TIER",
    key: "aTier" as const,
    colorBg: "bg-hacker-purple/20",
    colorText: "text-hacker-purple",
    display: "A",
  },
  {
    label: "B-TIER",
    key: "bTier" as const,
    colorBg: "bg-hacker-cyan/20",
    colorText: "text-hacker-cyan",
    display: "B",
  },
  {
    label: "LOW FRICTION",
    key: "lowFriction" as const,
    colorBg: "bg-hacker-green/20",
    colorText: "text-hacker-green",
    display: "\u2713",
  },
  {
    label: "HIDDEN GEMS",
    key: "hiddenGems" as const,
    colorBg: "bg-hacker-orange/20",
    colorText: "text-hacker-orange",
    display: "\uD83D\uDC8E",
  },
];

export default function StatsPanel({ stats }: StatsPanelProps) {
  return (
    <section className="border-b border-hacker-border bg-hacker-card/30">
      <div className="max-w-7xl mx-auto px-6 py-4">
        <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
          {statItems.map((item) => (
            <div
              key={item.key}
              className="bg-hacker-card rounded-lg p-4 border border-hacker-border"
            >
              <div className="flex items-center gap-3">
                <div
                  className={`w-10 h-10 rounded-lg ${item.colorBg} flex items-center justify-center`}
                >
                  <span className={`${item.colorText} font-bold`}>
                    {item.display}
                  </span>
                </div>
                <div>
                  <div className="text-hacker-muted text-xs font-mono">
                    {item.label}
                  </div>
                  <div className={`${item.colorText} font-mono font-bold`}>
                    {stats[item.key]}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
