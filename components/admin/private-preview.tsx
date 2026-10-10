"use client";
import Image from "next/image";
import { useEffect, useState } from "react";
import { request } from "@/lib/client-api";
import type { ContentDoc } from "@/lib/types";
import { imageUrl } from "@/lib/utils";
import { ContentDetail } from "@/components/site/content-detail";
import { PageHeading } from "@/components/site/ui";
import { RichText } from "@/components/site/rich-text";
export function PrivatePreview({ id }: { id: string }) {
  const [doc, setDoc] = useState<ContentDoc | null>(null);
  const [error, setError] = useState("");
  useEffect(() => {
    let active = true;
    request<{ document: ContentDoc }>(
      `/v1/admin/content/${encodeURIComponent(id)}`,
    )
      .then((d) => {
        if (active) setDoc(d.document);
      })
      .catch((e) => {
        if (active) setError(e.message);
      });
    return () => {
      active = false;
    };
  }, [id]);
  if (error)
    return (
      <div className="shell py-20">
        <p role="alert">{error}</p>
        <a className="text-link mt-6" href="/admin">
          Back to dashboard
        </a>
      </div>
    );
  if (!doc)
    return (
      <div className="shell py-20">
        <div className="skeleton h-40" aria-label="Loading private preview" />
      </div>
    );
  const src = imageUrl(doc.image || doc.heroImage);
  return (
    <>
      <div className="preview-banner">Private preview · Saved changes only</div>
      {["projects", "note"].includes(doc._type) ? (
        <ContentDetail doc={doc} preview />
      ) : (
        <div className="shell page-bottom">
          <PageHeading
            eyebrow="Private preview"
            title={doc.homepageTitle || doc.title || doc.name || "Preview"}
          >
            <p>{doc.summary || doc.intro || doc.homepageSubtitle}</p>
          </PageHeading>
          {src && (
            <Image
              unoptimized
              src={src}
              alt={doc.image?.alt || "Preview image"}
              width={900}
              height={650}
              className="max-h-[650px] w-auto object-contain"
            />
          )}
          <div className="prose reading-width mt-10">
            <p>{doc.bio}</p>
            <RichText value={doc.body} preview />
          </div>
        </div>
      )}
    </>
  );
}
