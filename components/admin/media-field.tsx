"use client";
import Image from "next/image";
import { useState } from "react";
import { mutate } from "@/lib/client-api";
import { imageUrl } from "@/lib/utils";
import { MediaLibrary } from "./media-library";
import type { ImageAsset } from "@/lib/types";
export function MediaField({
  value,
  onChange,
  label = "Image",
}: {
  value?: ImageAsset;
  onChange: (value: ImageAsset | undefined) => void;
  label?: string;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const src = imageUrl(value);
  const [library, setLibrary] = useState(false);
  return (
    <div className="space-y-4">
      <label>
        {label}
        <input
          type="file"
          accept="image/jpeg,image/png,image/webp"
          disabled={busy}
          onChange={async (e) => {
            const file = e.target.files?.[0];
            if (!file) return;
            setError("");
            if (file.size > 10_000_000) {
              setError("Choose an image under 10 MB.");
              return;
            }
            setBusy(true);
            try {
              const data = new FormData();
              data.set("file", file);
              const result = await mutate<{ image: ImageAsset }>(
                "/v1/admin/media",
                data,
              );
              onChange({
                ...result.image,
                alt: value?.alt || "",
                caption: value?.caption || "",
              });
            } catch (e) {
              setError((e as Error).message);
            } finally {
              setBusy(false);
            }
          }}
        />
      </label>
      <button
        type="button"
        className="text-link text-xs"
        onClick={() => setLibrary((v) => !v)}
      >
        {library ? "Close media library" : "Choose from media library"}
      </button>
      {library && (
        <MediaLibrary
          onSelect={(image) => {
            onChange(image);
            setLibrary(false);
          }}
        />
      )}
      {busy && (
        <p role="status" className="text-sm">
          Uploading and optimizing image…
        </p>
      )}
      {src && (
        <>
          <div className="upload-preview">
            <Image
              unoptimized
              src={src}
              alt={value?.alt || "Image being edited"}
              width={500}
              height={300}
            />
          </div>
          <label>
            Alternative text
            <input
              value={value?.alt || ""}
              onChange={(e) => onChange({ ...value!, alt: e.target.value })}
              maxLength={500}
              placeholder="Describe what the photograph shows"
            />
          </label>
          <label>
            Caption
            <input
              value={value?.caption || ""}
              onChange={(e) => onChange({ ...value!, caption: e.target.value })}
              maxLength={1000}
            />
          </label>
          <button
            type="button"
            className="text-link text-xs text-rose"
            onClick={() => onChange(undefined)}
          >
            Remove image
          </button>
        </>
      )}
      {error && (
        <p role="alert" className="form-error">
          {error}
        </p>
      )}
    </div>
  );
}
