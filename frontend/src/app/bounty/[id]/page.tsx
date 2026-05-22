import { fetchBountiesServer } from "@/lib/api-server";
import BountyDetailClient from "@/components/BountyDetailClient";
import { notFound } from "next/navigation";
import { bountySlug } from "@/lib/utils";

interface BountyPageProps {
  params: { id: string };
}

export async function generateStaticParams() {
  const bounties = fetchBountiesServer();
  return bounties.map((b) => ({
    id: bountySlug(b.repository, b.number),
  }));
}

export default function BountyPage({ params }: BountyPageProps) {
  const bounties = fetchBountiesServer();
  const bounty = bounties.find(
    (b) => bountySlug(b.repository, b.number) === params.id
  );

  if (!bounty) {
    notFound();
  }

  return <BountyDetailClient bounty={bounty} />;
}
