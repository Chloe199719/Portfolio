import type { Metadata } from "next";
export const metadata: Metadata = {
  title: "Dashboard",
  robots: { index: false, follow: false },
  alternates: { canonical: null },
};
export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
