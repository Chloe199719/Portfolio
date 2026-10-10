"use client";
import Image from "next/image";
import { useEffect, useRef, useState } from "react";
import {
  ArrowLeftIcon,
  ArrowRightIcon,
  Cross1Icon,
} from "@radix-ui/react-icons";
import type { ContentDoc } from "@/lib/types";
import { imageUrl } from "@/lib/utils";
export function Gallery({ photos }: { photos: ContentDoc[] }) {
  const [active, setActive] = useState<number | null>(null);
  const dialog = useRef<HTMLDialogElement>(null);
  const photo = active === null ? null : photos[active];
  useEffect(() => {
    if (active !== null && !dialog.current?.open) dialog.current?.showModal();
    if (active === null && dialog.current?.open) dialog.current?.close();
  }, [active]);
  function step(direction: number) {
    setActive((i) =>
      i === null ? null : (i + direction + photos.length) % photos.length,
    );
  }
  return (
    <>
      <div className="photo-gallery">
        {photos.map((p, i) => (
          <figure key={p._id}>
            <button
              className="gallery-image"
              aria-label={`Open ${p.title}`}
              onClick={() => setActive(i)}
            >
              <Image
                src={imageUrl(p.image)!}
                alt={p.image?.alt || p.title || ""}
                width={1000}
                height={1200}
                sizes="(max-width:767px) 90vw, 45vw"
              />
            </button>
            <figcaption>
              <span>{p.title}</span>
              <span>{p.location || p.date?.slice(0, 4)}</span>
            </figcaption>
          </figure>
        ))}
      </div>
      <dialog
        ref={dialog}
        className="lightbox"
        aria-label={photo?.title || "Photograph viewer"}
        onClose={() => setActive(null)}
        onClick={(e) => {
          if (e.target === dialog.current) setActive(null);
        }}
        onKeyDown={(e) => {
          if (e.key === "ArrowLeft") step(-1);
          if (e.key === "ArrowRight") step(1);
        }}
      >
        <button
          className="lightbox-close icon-button"
          aria-label="Close photograph"
          onClick={() => setActive(null)}
        >
          <Cross1Icon />
        </button>
        {photo && (
          <div className="lightbox-inner">
            <Image
              src={imageUrl(photo.image)!}
              alt={photo.image?.alt || photo.title || ""}
              width={1800}
              height={1400}
              sizes="90vw"
            />
            <div className="lightbox-caption">
              <button
                className="icon-button"
                aria-label="Previous photograph"
                onClick={() => step(-1)}
              >
                <ArrowLeftIcon />
              </button>
              <div>
                <h2>{photo.title}</h2>
                <p>{photo.image?.caption || photo.summary}</p>
                <span>
                  {active! + 1} / {photos.length}
                </span>
              </div>
              <button
                className="icon-button"
                aria-label="Next photograph"
                onClick={() => step(1)}
              >
                <ArrowRightIcon />
              </button>
            </div>
          </div>
        )}
      </dialog>
    </>
  );
}
