import { fetchBountiesServer } from "@/lib/api-server";
import BountyDetailClient from "@/components/BountyDetailClient";
import BountyDisambiguation from "@/components/BountyDisambiguation";
import { notFound } from "next/navigation";
import { bountySlug } from "@/lib/utils";

interface BountyPageProps {
  params: { id: string };
}

export async function generateStaticParams() {
  const bounties = fetchBountiesServer();

  // Primary: slug-based params for every bounty
  const slugParams = bounties.map((b) => ({
    id: bountySlug(b.repository, b.number),
  }));

  // Backward compat: bare-number params for all unique issue numbers
  const seenNumbers = new Set<number>();
  const bareNumberParams: { id: string }[] = [];
  for (const b of bounties) {
    if (!seenNumbers.has(b.number)) {
      seenNumbers.add(b.number);
      bareNumberParams.push({ id: String(b.number) });
    }
  }

  return [...slugParams, ...bareNumberParams];
}

export default function BountyPage({ params }: BountyPageProps) {
  const bounties = fetchBountiesServer();

  // Primary lookup: composite slug
  const bounty = bounties.find(
    (b) => bountySlug(b.repository, b.number) === params.id
  );

  if (bounty) {
    return <BountyDetailClient bounty={bounty} />;
  }

  // Backward compat: bare number lookup
  if (/^\d+$/.test(params.id)) {
    const matches = bounties.filter((b) => b.number === Number(params.id));
    if (matches.length === 1) {
      return <BountyDetailClient bounty={matches[0]} />;
    }
    if (matches.length > 1) {
      return <BountyDisambiguation matches={matches} number={params.id} />;
    }
  }

  notFound();
}
