import { BountyIssue } from "./types";

const DATA_URL = "./data/enriched_bounties.json";

export async function fetchBounties(): Promise<BountyIssue[]> {
  try {
    const response = await fetch(DATA_URL);
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return await response.json();
  } catch (error) {
    console.error("Failed to load bounties:", error);
    return [];
  }
}
