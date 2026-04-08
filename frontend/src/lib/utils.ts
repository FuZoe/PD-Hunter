import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatBounty(amount: number): string {
  if (amount >= 1000) {
    return `$${(amount / 1000).toFixed(1)}k`;
  }
  return `$${amount}`;
}

export function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
  });
}

export const tierColors = {
  "S-Tier": {
    bg: "bg-hacker-yellow/10",
    border: "border-hacker-yellow",
    text: "text-hacker-yellow",
    badge: "bg-hacker-yellow text-black",
  },
  "A-Tier": {
    bg: "bg-hacker-purple/10",
    border: "border-hacker-purple",
    text: "text-hacker-purple",
    badge: "bg-hacker-purple text-white",
  },
  "B-Tier": {
    bg: "bg-hacker-cyan/10",
    border: "border-hacker-cyan",
    text: "text-hacker-cyan",
    badge: "bg-hacker-cyan text-black",
  },
} as const;

export const frictionConfig = {
  Low: { class: "friction-low", color: "text-hacker-green", icon: "\u26A1" },
  Medium: {
    class: "friction-medium",
    color: "text-hacker-yellow",
    icon: "\u2699\uFE0F",
  },
  High: { class: "friction-high", color: "text-hacker-red", icon: "\uD83D\uDD25" },
} as const;
