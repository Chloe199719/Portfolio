import { CameraIcon } from "@radix-ui/react-icons";
import { getContent } from "@/lib/content";
import { metadata as makeMetadata } from "@/lib/seo";
import { PageHeading } from "@/components/site/ui";
import { Gallery } from "@/components/interactive/gallery";
export const metadata = makeMetadata(
  "Photography",
  "A collection of photographs, details, and everyday moments by Chloe Pratas.",
  "/photography",
);
export default async function Photography() {
  const { photos } = await getContent();
  return (
    <div className="shell page-bottom">
      <PageHeading eyebrow="Outside the browser" title="Through my lens.">
        <p>
          A place for moments I notice. Some ordinary, some unexpected, all
          worth keeping.
        </p>
      </PageHeading>
      {photos.length ? (
        <Gallery photos={photos.filter((p) => p.image)} />
      ) : (
        <div className="gallery-empty">
          <CameraIcon />
          <div>
            <p className="eyebrow">The photo journal</p>
            <h2>
              A collection,
              <br />
              <em>taking shape.</em>
            </h2>
            <p>
              Photographs and the stories behind them will find a home here.
            </p>
          </div>
          <span className="gallery-empty-index">FRAME 001 / OPEN</span>
        </div>
      )}
    </div>
  );
}
