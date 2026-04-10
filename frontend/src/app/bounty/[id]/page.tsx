import { fetchBountiesServer } from "@/lib/api-server";
import BountyDetailClient from "@/components/BountyDetailClient";
import { notFound } from "next/navigation";

interface BountyPageProps {
  params: { id: string };
}

export async function generateStaticParams() {
  const bounties = fetchBountiesServer();
  return bounties.map((b) => ({
    id: String(b.number),
  }));
}

export default function BountyPage({ params }: BountyPageProps) {
  const bounties = fetchBountiesServer();
  const bounty = bounties.find((b) => String(b.number) === params.id);

  if (!bounty) {
    notFound();
  }

  return <BountyDetailClient bounty={bounty} />;
}
