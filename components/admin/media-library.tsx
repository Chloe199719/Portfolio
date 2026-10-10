"use client";
import Image from "next/image";
import { useCallback, useEffect, useState } from "react";
import { mutate, request } from "@/lib/client-api";
import { imageUrl } from "@/lib/utils";
import type { ImageAsset } from "@/lib/types";
type Asset = {
  id: string;
  alt: string;
  caption: string;
  width: number;
  height: number;
  usedBy: { id: string; state: string; title: string }[];
};
const asImage = (a: Asset): ImageAsset => ({
  _type: "image",
  asset: { _type: "reference", _ref: a.id },
  alt: a.alt,
  caption: a.caption,
});
export function MediaLibrary({
  onSelect,
}: {
  onSelect?: (image: ImageAsset) => void;
}) {
  const [assets, setAssets] = useState<Asset[]>([]),
    [search, setSearch] = useState(""),
    [page, setPage] = useState(1),
    [total, setTotal] = useState(0),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [notice, setNotice] = useState("");
  const load = useCallback(async () => {
    const data = await request<{ assets: Asset[]; total: number }>(
      `/v1/admin/media?q=${encodeURIComponent(search)}&page=${page}`,
    );
    setAssets(data.assets);
    setTotal(data.total);
  }, [search, page]);
  useEffect(() => {
    const timer = setTimeout(() => {
      void load().catch((e) => setError(e.message));
    }, 200);
    return () => clearTimeout(timer);
  }, [load]);
  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-end gap-5">
        <label className="flex-1">
          Search media
          <input
            type="search"
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setPage(1);
            }}
          />
        </label>
        <label className="flex-1">
          Upload photographs
          <input
            type="file"
            multiple
            accept="image/jpeg,image/png,image/webp"
            disabled={busy}
            onChange={async (e) => {
              const files = Array.from(e.target.files || []);
              setBusy(true);
              setError("");
              let count = 0;
              try {
                for (const file of files) {
                  if (file.size > 10 * 1024 * 1024)
                    throw new Error(`${file.name} is larger than 10 MB.`);
                  const body = new FormData();
                  body.set("file", file);
                  await mutate("/v1/admin/media", body);
                  count++;
                  setNotice(`Uploaded ${count} of ${files.length}`);
                }
                await load();
              } catch (err) {
                setError((err as Error).message);
                await load();
              } finally {
                setBusy(false);
              }
            }}
          />
        </label>
      </div>
      <p className="form-help">
        Images stay private until a published entry uses them. Library text is
        copied when you select an image; existing articles keep their own
        captions.
      </p>
      {notice && (
        <p role="status" className="form-success">
          {notice}
        </p>
      )}
      {error && (
        <p role="alert" className="form-error">
          {error}
        </p>
      )}
      <div className="media-grid">
        {assets.map((asset) => (
          <MediaRecord
            key={asset.id}
            asset={asset}
            onSelect={onSelect}
            onError={setError}
            onSaved={() => setNotice("Media details saved.")}
          />
        ))}
      </div>
      {!assets.length && (
        <p className="form-help">
          No images match this view. Upload a photograph to begin.
        </p>
      )}
      <div className="flex items-center gap-5">
        <button
          className="text-link"
          disabled={page === 1}
          onClick={() => setPage((v) => v - 1)}
        >
          Previous
        </button>
        <span className="text-xs text-muted">
          Page {page} · {total} images
        </span>
        <button
          className="text-link"
          disabled={page * 30 >= total}
          onClick={() => setPage((v) => v + 1)}
        >
          Next
        </button>
      </div>
    </div>
  );
}
function MediaRecord({
  asset,
  onSelect,
  onError,
  onSaved,
}: {
  asset: Asset;
  onSelect?: (image: ImageAsset) => void;
  onError: (e: string) => void;
  onSaved: () => void;
}) {
  const [alt, setAlt] = useState(asset.alt),
    [caption, setCaption] = useState(asset.caption),
    [busy, setBusy] = useState(false);
  return (
    <div className="media-record">
      <Image
        unoptimized
        src={imageUrl(asImage(asset))!}
        alt={alt || "Media library preview"}
        width={500}
        height={350}
      />
      <small>
        {asset.width} × {asset.height}
      </small>
      {onSelect ? (
        <button
          className="button button-outline"
          onClick={() => onSelect(asImage(asset))}
        >
          Use image
        </button>
      ) : (
        <>
          <label>
            Alternative text
            <input
              value={alt}
              onChange={(e) => setAlt(e.target.value)}
              maxLength={500}
            />
          </label>
          <label>
            Caption
            <input
              value={caption}
              onChange={(e) => setCaption(e.target.value)}
              maxLength={1000}
            />
          </label>
          <button
            className="text-link mt-3"
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              try {
                await mutate(
                  `/v1/admin/media/${asset.id}`,
                  { alt, caption },
                  "PATCH",
                );
                onSaved();
              } catch (e) {
                onError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            {busy ? "Saving…" : "Save details"}
          </button>
        </>
      )}
      <small>
        {asset.usedBy.length
          ? asset.usedBy.map((v) => `${v.title} (${v.state})`).join(" · ")
          : "Not used in content"}
      </small>
    </div>
  );
}
