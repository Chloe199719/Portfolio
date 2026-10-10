import { PrivatePreview } from "@/components/admin/private-preview";
export default async function Preview({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  return <PrivatePreview id={(await params).id} />;
}
