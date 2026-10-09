import type { Metadata } from "next";
import { siteUrl } from "./utils";
export function metadata(
  title: string,
  description: string,
  path: string,
): Metadata {
  return {
    title,
    description,
    alternates: { canonical: path },
    openGraph: {
      title: `${title} — Chloe Pratas`,
      description,
      url: `${siteUrl}${path}`,
      type: "website",
      images: [`${siteUrl}/opengraph-image`],
    },
    twitter: {
      card: "summary_large_image",
      title,
      description,
      images: [`${siteUrl}/opengraph-image`],
    },
  };
}
