export interface HunterIntelligence {
  friction_level: "High" | "Medium" | "Low";
  technical_hint: string;
  bounty_tier: "S-Tier" | "A-Tier" | "B-Tier";
  bounty_amount: number;
  is_hidden_gem: boolean;
}

export interface BountyIssue {
  number: number;
  title: string;
  url: string;
  state: string;
  labels: string[];
  comment_count: number;
  open_pr_count: number;
  repository: string;
  created_at: string;
  updated_at: string;
  author: string;
  body: string;
  hunter_intelligence: HunterIntelligence;
}

export type TierFilter = "all" | "S-Tier" | "A-Tier" | "B-Tier";

export type SortOption =
  | "bounty-desc"
  | "bounty-asc"
  | "friction-asc"
  | "date-desc";
