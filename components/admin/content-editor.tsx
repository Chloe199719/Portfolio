"use client";
import dynamic from "next/dynamic";
import { useState } from "react";
import { ArrowTopRightIcon } from "@radix-ui/react-icons";
import type { ContentDoc } from "@/lib/types";
import { useDraft, type Editable } from "./use-draft";
import { PublishingTools } from "./publishing-tools";
export type { Editable } from "./use-draft";
import { slugify } from "@/lib/utils";
import { MediaField } from "./media-field";
const RichEditor = dynamic(
  () => import("./rich-editor").then((m) => m.RichEditor),
  {
    ssr: false,
    loading: () => (
      <div className="skeleton h-64" aria-label="Loading text editor" />
    ),
  },
);
export function ContentEditor({
  initial,
  onSaved,
  onDirty,
}: {
  initial: Editable;
  onSaved: (doc: Editable) => void;
  onDirty: (dirty: boolean) => void;
}) {
  const draft = useDraft(initial, onSaved, onDirty);
  const { doc, dirty, busy, error, notice, change } = draft;
  const [slugEdited, setSlugEdited] = useState(Boolean(initial.slug?.current));
  const [confirmUnpublish, setConfirmUnpublish] = useState(false);
  async function save(action: "save" | "publish" | "unpublish") {
    try {
      await draft.save(action);
      setConfirmUnpublish(false);
    } catch {
      /* Error and recovery are managed by the draft hook. */
    }
  }
  function field(
    key: keyof ContentDoc,
    label: string,
    textarea = false,
    type = "text",
  ) {
    const value = typeof doc[key] === "string" ? (doc[key] as string) : "";
    return (
      <label key={key}>
        {label}
        {textarea ? (
          <textarea
            rows={4}
            value={value}
            onChange={(e) => change({ [key]: e.target.value })}
          />
        ) : (
          <input
            type={type}
            value={value}
            onChange={(e) => change({ [key]: e.target.value || undefined })}
          />
        )}
      </label>
    );
  }
  const profile = doc._type === "pageInfo";
  const settings = doc._type === "siteSettings";
  const article = doc._type === "note" || doc._type === "projects";
  return (
    <div className="editor">
      <div className="editor-heading">
        <div>
          <h2>
            {profile
              ? "Your profile"
              : settings
                ? "Homepage settings"
                : doc.title || "New entry"}
          </h2>
          <p className="form-help">
            {dirty
              ? "Unsaved changes"
              : doc.isDraft
                ? "Draft"
                : doc.hasPublished
                  ? "Published"
                  : "New draft"}
          </p>
        </div>
        {doc._rev && (
          <a
            href={`/admin/preview/${doc._id}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-link"
          >
            Preview saved version <ArrowTopRightIcon />
          </a>
        )}
      </div>
      {draft.recovery && (
        <div className="inline-confirm">
          <span>
            Browser edits were recovered
            {draft.recovery.baseRevision !== doc._rev
              ? " from an older revision"
              : ""}
            . Review before publishing.
          </span>
          <button onClick={draft.recover}>Restore browser edits</button>
          <button onClick={draft.discardRecovery}>Use server version</button>
        </div>
      )}
      <div className="editor-fields">
        {profile ? (
          <>
            {field("name", "Name")}
            {field("role", "Professional title")}
            {field("email", "Contact email", false, "email")}
            {field("intro", "Homepage introduction", true)}
            {field("bio", "About you", true)}
            <label>
              Interests, separated by commas
              <DelimitedInput
                value={doc.interests}
                onChange={(interests) => change({ interests })}
              />
            </label>
            <MediaField
              label="Homepage portrait"
              value={doc.heroImage}
              onChange={(heroImage) => change({ heroImage })}
            />
            <MediaField
              label="About portrait"
              value={doc.profilePic}
              onChange={(profilePic) => change({ profilePic })}
            />
          </>
        ) : settings ? (
          <>
            {field("homepageTitle", "Homepage headline")}
            {field("homepageSubtitle", "Site description", true)}
            <fieldset className="space-y-3">
              <legend className="mb-4 text-sm">
                Homepage modules · empty collections stay hidden
              </legend>
              {(
                doc.modules ||
                ["work", "photos", "now", "notes", "guestbook"].map((id) => ({
                  id: id as NonNullable<ContentDoc["modules"]>[number]["id"],
                  visible: true,
                }))
              ).map((module, index, all) => (
                <div
                  key={module.id}
                  className="flex flex-wrap items-center gap-4"
                >
                  <label className="check-label">
                    <input
                      type="checkbox"
                      checked={module.visible}
                      onChange={(e) =>
                        change({
                          modules: all.map((m) =>
                            m.id === module.id
                              ? { ...m, visible: e.target.checked }
                              : m,
                          ),
                        })
                      }
                    />
                    {module.id}
                  </label>
                  <button
                    type="button"
                    className="text-link text-xs"
                    disabled={index === 0}
                    onClick={() => {
                      const next = [...all];
                      [next[index - 1], next[index]] = [
                        next[index],
                        next[index - 1],
                      ];
                      change({ modules: next });
                    }}
                  >
                    Move up
                  </button>
                  <button
                    type="button"
                    className="text-link text-xs"
                    disabled={index === all.length - 1}
                    onClick={() => {
                      const next = [...all];
                      [next[index + 1], next[index]] = [
                        next[index],
                        next[index + 1],
                      ];
                      change({ modules: next });
                    }}
                  >
                    Move down
                  </button>
                </div>
              ))}
            </fieldset>
          </>
        ) : (
          <>
            <label>
              Title
              <input
                value={doc.title || ""}
                onChange={(e) =>
                  change({
                    title: e.target.value,
                    ...(article && !doc.hasPublished && !slugEdited
                      ? { slug: { current: slugify(e.target.value) } }
                      : {}),
                  })
                }
              />
            </label>
            {article && (
              <label>
                URL slug
                <input
                  value={doc.slug?.current || ""}
                  onChange={(e) => {
                    setSlugEdited(true);
                    change({ slug: { current: slugify(e.target.value) } });
                  }}
                  readOnly={Boolean(doc.hasPublished || doc._publishedSlug)}
                />
                <span className="form-help">
                  {doc.hasPublished || doc._publishedSlug
                    ? "Published URLs stay fixed so existing links keep working."
                    : "Use lowercase words separated by hyphens."}
                </span>
              </label>
            )}
            {field(
              "summary",
              doc._type === "nowUpdate" ? "Your update" : "Short description",
              true,
            )}
            {["note", "photograph", "nowUpdate"].includes(doc._type) &&
              field("date", "Date", false, "date")}
            {["note", "nowUpdate"].includes(doc._type) &&
              field("category", "Category")}
            {doc._type === "photograph" &&
              field("location", "Location (optional)")}
            {doc._type === "note" && (
              <label>
                Topics, separated by commas
                <DelimitedInput
                  value={doc.tags}
                  onChange={(tags) => change({ tags })}
                />
              </label>
            )}
            {["photograph", "projects", "note"].includes(doc._type) && (
              <MediaField
                value={doc.image}
                onChange={(image) => change({ image })}
              />
            )}
            <label className="check-label">
              <input
                type="checkbox"
                checked={doc.featured || false}
                onChange={(e) => change({ featured: e.target.checked })}
              />
              Feature on the homepage
            </label>
            <label>
              Display order
              <input
                type="number"
                min={0}
                max={9999}
                value={doc.order ?? 99}
                onChange={(e) => change({ order: Number(e.target.value) })}
              />
            </label>
            {doc._type === "projects" && (
              <>
                {field("linkToBuild", "Project or source URL", false, "url")}
                {field("projectRole", "Your role")}
                {field("problem", "The problem", true)}
                {field("decisions", "Decisions along the way", true)}
                {field("outcome", "The outcome", true)}
              </>
            )}
          </>
        )}
        {!settings && doc._type !== "photograph" && (
          <div>
            <span className="mb-3 block text-sm font-medium">
              {profile
                ? "More of your story"
                : "Full story / additional details"}
            </span>
            <RichEditor
              key={`${initial._id}:${draft.editorVersion}`}
              value={doc.body}
              onChange={(body) => change({ body })}
            />
          </div>
        )}
      </div>
      {error && (
        <p role="alert" className="form-error">
          {error} Your edits remain in this form.
        </p>
      )}
      {notice && (
        <p role="status" className="form-success">
          {notice}
        </p>
      )}
      <PublishingTools
        doc={doc}
        dirty={dirty}
        busy={Boolean(busy)}
        save={draft.save}
        onRestore={draft.replace}
      />
      <div className="editor-actions">
        <button
          className="button button-outline"
          disabled={Boolean(busy)}
          onClick={() => save("save")}
        >
          {busy === "save" ? "Saving…" : "Save draft"}
        </button>
        <button
          className="button"
          disabled={Boolean(busy)}
          onClick={() => save("publish")}
        >
          {busy === "publish" ? "Publishing…" : "Publish"}
        </button>
        {doc.hasPublished && !profile && !settings && (
          <button
            className="text-link text-xs text-rose"
            disabled={Boolean(busy)}
            onClick={() => setConfirmUnpublish(true)}
          >
            Unpublish
          </button>
        )}
      </div>
      {confirmUnpublish && (
        <div className="inline-confirm">
          <span>Remove this from the public site? Your draft will remain.</span>
          <button disabled={Boolean(busy)} onClick={() => save("unpublish")}>
            Unpublish
          </button>
          <button onClick={() => setConfirmUnpublish(false)}>
            Keep published
          </button>
        </div>
      )}
    </div>
  );
}

function DelimitedInput({
  value,
  onChange,
}: {
  value?: string[];
  onChange: (value: string[]) => void;
}) {
  const [raw, setRaw] = useState(value?.join(", ") || "");
  return (
    <input
      value={raw}
      onChange={(e) => {
        setRaw(e.target.value);
        onChange(
          e.target.value
            .split(",")
            .map((v) => v.trim())
            .filter(Boolean),
        );
      }}
    />
  );
}
