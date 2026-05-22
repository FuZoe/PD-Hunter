import fs from "fs";
import path from "path";
import { BountyIssue } from "./types";

export function fetchBountiesServer(): BountyIssue[] {
  const filePath = path.join(process.cwd(), "public", "data", "enriched_bounties.json");
  let bounties: BountyIssue[];
  if (!fs.existsSync(filePath)) {
    // Fallback: try the root-level file
    const rootPath = path.join(process.cwd(), "..", "enriched_bounties.json");
    if (fs.existsSync(rootPath)) {
      const raw = fs.readFileSync(rootPath, "utf-8");
      bounties = JSON.parse(raw);
    } else {
      return [];
    }
  } else {
    const raw = fs.readFileSync(filePath, "utf-8");
    bounties = JSON.parse(raw);
  }
  return bounties.filter((b) => b.state.toLowerCase() === "open");
}
