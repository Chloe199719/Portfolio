import { metadata as makeMetadata } from "@/lib/seo";
import { PageHeading } from "@/components/site/ui";
import { getContent } from "@/lib/content";
export const metadata = makeMetadata(
  "Privacy",
  "How this personal website handles messages, sign-in, and local browser storage.",
  "/privacy",
);
export default async function Privacy() {
  const { profile } = await getContent();
  return (
    <div className="shell page-bottom">
      <PageHeading
        eyebrow="A small site, a clear explanation"
        title="Your privacy."
      />
      <div className="reading-width prose">
        <h2>Messages and guestbook notes</h2>
        <p>
          Contact messages are stored privately on this website’s server and are
          visible only to the site owner. Guestbook notes include the public
          name you choose and become visible after moderation. Your sign-in
          email is not displayed in the guestbook. You can delete your own notes
          after signing in.
        </p>
        <h2>Sign-in</h2>
        <p>
          Accounts, sessions, and sign-in are handled by our own identity
          server. Google and GitHub receive sign-in requests only when you
          choose them. These providers process your sign-in under their own
          privacy policies. This website uses an essential session cookie
          lasting up to five days and a short-lived security cookie to protect
          form submissions.
        </p>
        <h2>What stays in your browser</h2>
        <p>
          The memory game saves your best score in local browser storage. No
          advertising trackers or analytics cookies are included.
        </p>
        <h2>Your choices</h2>
        <p>
          You can browse without an account. Sign out to remove this website’s
          session cookie, and clear your browser data to reset your game score.
          To request removal of a contact message or account data, contact{" "}
          <a href={`mailto:${profile.email}`}>{profile.email}</a>.
        </p>
      </div>
    </div>
  );
}
