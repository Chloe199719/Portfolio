import { getContent } from "@/lib/content";
import { formatDate } from "@/lib/utils";
import { metadata as makeMetadata } from "@/lib/seo";
import { PageHeading, EmptyState, TextLink } from "@/components/site/ui";
import { RichText } from "@/components/site/rich-text";
export const metadata = makeMetadata(
  "Now",
  "A snapshot of what Chloe is building, learning, training, and playing.",
  "/now",
);
export default async function Now() {
  const { updates } = await getContent();
  return (
    <div className="shell page-bottom">
      <PageHeading
        eyebrow="A snapshot, not a schedule"
        title="What’s happening now."
      >
        <p>Things I’m building, learning, playing, and making time for.</p>
      </PageHeading>
      {updates.length ? (
        <div className="updates-list">
          {updates.map((u) => (
            <article key={u._id}>
              <div>
                <p className="eyebrow">{u.category || "Life lately"}</p>
                <span className="text-sm text-muted">
                  Updated {formatDate(u.date)}
                </span>
              </div>
              <div>
                <h2>{u.title}</h2>
                <p>{u.summary}</p>
                <RichText value={u.body} />
              </div>
            </article>
          ))}
        </div>
      ) : (
        <EmptyState title="A little room for what’s next.">
          <p>Updates about life at the keyboard and beyond will live here.</p>
          <TextLink href="/about">Get to know me in the meantime</TextLink>
        </EmptyState>
      )}
    </div>
  );
}
