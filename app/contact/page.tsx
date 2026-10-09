import { getContent } from "@/lib/content";
import { apiBase } from "@/lib/api-config";
import { metadata as makeMetadata } from "@/lib/seo";
import { PageHeading, TextLink } from "@/components/site/ui";
import { ContactForm } from "@/components/interactive/contact-form";
export const metadata = makeMetadata(
  "Contact",
  "Say hello to Chloe Pratas — about a project, a question, or a shared interest.",
  "/contact",
);
export default async function Contact() {
  const { profile } = await getContent();
  return (
    <div className="shell page-bottom">
      <PageHeading
        eyebrow="An open invitation"
        title="Good conversations start here."
      >
        <p>
          A project in mind, a question, or something we have in common? I’d
          love to hear it.
        </p>
      </PageHeading>
      <div className="contact-layout">
        <aside>
          <h2>
            Just a hello
            <br />
            is a good <em>start.</em>
          </h2>
          {profile.email && (
            <TextLink href={`mailto:${profile.email}`}>
              {profile.email}
            </TextLink>
          )}
          <p className="mt-8 text-muted leading-relaxed">
            Prefer a little public note?
            <br />
            <TextLink href="/guestbook">There’s a guestbook for that</TextLink>
          </p>
        </aside>
        <ContactForm configured={Boolean(apiBase)} />
      </div>
    </div>
  );
}
