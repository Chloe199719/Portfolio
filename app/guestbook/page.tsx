import { metadata as makeMetadata } from "@/lib/seo";
import { PageHeading } from "@/components/site/ui";
import { Guestbook } from "@/components/interactive/guestbook";
import { publicAPI } from "@/lib/public-api";
import { apiBase } from "@/lib/api-config";
import type { GuestEntry } from "@/lib/types";
export const metadata = makeMetadata(
  "Guestbook",
  "A little collection of hellos from people who stopped by.",
  "/guestbook",
);
export default async function GuestbookPage() {
  return (
    <div className="shell page-bottom">
      <PageHeading
        eyebrow="A little trace of being here"
        title="You were here."
      >
        <p>
          A guestbook for good conversations and small connections. Leave a
          note, find a familiar name, make this corner of the internet a little
          warmer.
        </p>
      </PageHeading>
      <Guestbook
        initialEntries={
          (await publicAPI<{ entries: GuestEntry[] }>("/v1/guestbook")).entries
        }
        storageReady={Boolean(apiBase)}
      />
    </div>
  );
}
