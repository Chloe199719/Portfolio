import Image from "next/image";
import type { ContentDoc } from "@/lib/types";
import { formatDate, imageUrl } from "@/lib/utils";
import { PageHeading, TextLink } from "./ui";
import { RichText } from "./rich-text";
export function ContentDetail({
  doc,
  preview = false,
}: {
  doc: ContentDoc;
  preview?: boolean;
}) {
  const isProject = doc._type === "projects";
  const src = imageUrl(doc.image);
  return (
    <article className="shell page-bottom">
      <div className="pt-10">
        <TextLink href={isProject ? "/work" : "/notes"}>
          {isProject ? "Back to work" : "All notes"}
        </TextLink>
      </div>
      <PageHeading
        eyebrow={
          isProject
            ? "Project archive"
            : `${doc.category || "A note"}${doc.date ? ` / ${formatDate(doc.date)}` : ""}`
        }
        title={doc.title || "Untitled"}
      >
        <p>{doc.summary}</p>
      </PageHeading>
      {src && (
        <figure className="detail-cover">
          <Image
            unoptimized={preview}
            src={src}
            alt={doc.image?.alt || `${doc.title} screenshot`}
            width={1500}
            height={950}
            sizes="(max-width: 1200px) 90vw, 1200px"
          />
          {doc.image?.caption && <figcaption>{doc.image.caption}</figcaption>}
        </figure>
      )}
      <div className="reading-width">
        {isProject && (
          <div className="project-meta">
            <div>
              <span className="eyebrow">Project</span>
              <p>{doc.projectRole || "Independent software project"}</p>
            </div>
            {doc.linkToBuild && (
              <TextLink href={doc.linkToBuild} external>
                {doc.linkToBuild.includes("github.com")
                  ? "View source code"
                  : "Visit project"}
              </TextLink>
            )}
          </div>
        )}
        {[
          ["The problem", doc.problem],
          ["Decisions along the way", doc.decisions],
          ["The outcome", doc.outcome],
        ].map(([label, value]) =>
          value ? (
            <section className="prose" key={label}>
              <h2>{label}</h2>
              <p>{value}</p>
            </section>
          ) : null,
        )}
        <RichText value={doc.body} preview={preview} />
      </div>
    </article>
  );
}
