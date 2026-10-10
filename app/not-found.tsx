import { PageHeading, TextLink } from "@/components/site/ui";
export default function NotFound() {
  return (
    <div className="shell page-bottom">
      <PageHeading
        eyebrow="404 / A little detour"
        title="This page wandered off."
      >
        <p>
          The link may have changed, or this story hasn’t been published yet.
        </p>
      </PageHeading>
      <TextLink href="/">Find your way home</TextLink>
    </div>
  );
}
