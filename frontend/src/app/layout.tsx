import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "PD-Hunter | Open Source Bounty Intelligence",
  description:
    "Find high-value open source bounties matched to your skills, powered by AI.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="font-sans text-hacker-text antialiased min-h-screen flex flex-col">
        {children}
        <footer className="border-t border-hacker-border bg-hacker-card/30 mt-auto">
          <div className="max-w-7xl mx-auto px-6 py-6">
            <div className="flex flex-col md:flex-row items-center justify-between gap-4">
              <div className="text-hacker-muted text-sm font-mono">
                <span className="text-hacker-green">$</span> PD-Hunter Bounty
                Intelligence Platform
              </div>
              <div className="text-hacker-muted text-xs font-mono">
                Data sourced from{" "}
                <span className="text-hacker-cyan">enriched_bounties.json</span>{" "}
                &bull; Built with Expert Intelligence
              </div>
            </div>
          </div>
        </footer>
      </body>
    </html>
  );
}
