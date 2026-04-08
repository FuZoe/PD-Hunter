import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        "hacker-bg": "#0a0a0f",
        "hacker-card": "#12121a",
        "hacker-border": "#1e1e2e",
        "hacker-green": "#00ff88",
        "hacker-cyan": "#00d4ff",
        "hacker-purple": "#a855f7",
        "hacker-yellow": "#fbbf24",
        "hacker-red": "#ef4444",
        "hacker-orange": "#f97316",
        "hacker-text": "#e2e8f0",
        "hacker-muted": "#64748b",
      },
      fontFamily: {
        mono: ["JetBrains Mono", "monospace"],
        sans: ["Inter", "sans-serif"],
      },
      keyframes: {
        "pulse-glow": {
          "0%, 100%": { boxShadow: "0 0 40px rgba(251, 191, 36, 0.3)" },
          "50%": { boxShadow: "0 0 60px rgba(251, 191, 36, 0.5)" },
        },
        blink: {
          "0%, 100%": { opacity: "1" },
          "50%": { opacity: "0" },
        },
      },
      animation: {
        "pulse-glow": "pulse-glow 2s ease-in-out infinite",
        blink: "blink 1s step-end infinite",
      },
    },
  },
  plugins: [],
};
export default config;
