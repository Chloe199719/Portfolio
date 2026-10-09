import type { Metadata } from "next";
import "@fontsource-variable/manrope";
import "@fontsource-variable/space-grotesk";
import "@/styles/globals.css";
import { Header } from "@/components/site/header";
import { Footer } from "@/components/site/footer";
import { siteUrl } from "@/lib/utils";
export const dynamic = "force-dynamic";
export const runtime = "nodejs";
export const metadata: Metadata = {
  metadataBase: new URL(siteUrl),
  title: {
    default: "Chloe Pratas — Software, side quests & life",
    template: "%s — Chloe Pratas",
  },
  description:
    "A personal collection of software projects, photography, notes, and life in between. By Chloe Pratas.",
  alternates: { canonical: "/" },
  openGraph: {
    type: "website",
    siteName: "Chloe Pratas",
    locale: "en_US",
    images: ["/opengraph-image"],
  },
  twitter: { card: "summary_large_image" },
  icons: { icon: "/icon.svg" },
};
export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" data-scroll-behavior="smooth">
      <body>
        <a className="skip-link" href="#main-content">
          Skip to content
        </a>
        <Header />
        <main id="main-content">{children}</main>
        <Footer />
      </body>
    </html>
  );
}
