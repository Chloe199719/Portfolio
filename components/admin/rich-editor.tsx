"use client";
import Image from "next/image";
import { useState } from "react";
import {
  EditorProvider,
  PortableTextEditable,
  defineSchema,
  useEditor,
  defineBlockObject,
} from "@portabletext/editor";
import { EventListenerPlugin, NodePlugin } from "@portabletext/editor/plugins";
import type { RichBlock, ImageAsset } from "@/lib/types";
import { imageUrl } from "@/lib/utils";
import { MediaField } from "./media-field";
const schema = defineSchema({
  styles: [
    { name: "normal" },
    { name: "h2" },
    { name: "h3" },
    { name: "blockquote" },
  ],
  decorators: [{ name: "strong" }, { name: "em" }],
  lists: [{ name: "bullet" }, { name: "number" }],
  annotations: [{ name: "link", fields: [{ name: "href", type: "string" }] }],
  blockObjects: [
    {
      name: "image",
      fields: [
        { name: "asset", type: "object" },
        { name: "alt", type: "string" },
        { name: "caption", type: "string" },
      ],
    },
  ],
});
const nodes = [
  defineBlockObject({
    type: "image",
    render: ({ node, children, attributes }) => {
      const value = node as unknown as ImageAsset;
      const src = imageUrl(value);
      return (
        <div {...attributes}>
          {children}
          <figure contentEditable={false}>
            {src && (
              <Image
                unoptimized
                src={src}
                alt={value.alt || ""}
                width={650}
                height={400}
              />
            )}
            <figcaption>{value.caption || value.alt}</figcaption>
          </figure>
        </div>
      );
    },
  }),
];
function Toolbar() {
  const editor = useEditor();
  const [imageOpen, setImageOpen] = useState(false);
  const [image, setImage] = useState<ImageAsset | undefined>();
  const [linkOpen, setLinkOpen] = useState(false);
  const [href, setHref] = useState("");
  return (
    <>
      <div className="rich-toolbar" role="toolbar" aria-label="Text formatting">
        {[
          ["strong", "Bold"],
          ["em", "Italic"],
        ].map(([decorator, label]) => (
          <button
            key={decorator}
            type="button"
            onMouseDown={(e) => e.preventDefault()}
            onClick={() => editor.send({ type: "decorator.toggle", decorator })}
          >
            {label}
          </button>
        ))}
        {[
          ["h2", "Heading"],
          ["h3", "Subheading"],
          ["blockquote", "Quote"],
        ].map(([style, label]) => (
          <button
            key={style}
            type="button"
            onMouseDown={(e) => e.preventDefault()}
            onClick={() => editor.send({ type: "style.toggle", style })}
          >
            {label}
          </button>
        ))}
        <button
          type="button"
          onMouseDown={(e) => e.preventDefault()}
          onClick={() =>
            editor.send({ type: "list item.toggle", listItem: "bullet" })
          }
        >
          List
        </button>
        <button
          type="button"
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => setLinkOpen(!linkOpen)}
        >
          Link
        </button>
        <button type="button" onClick={() => setImageOpen(!imageOpen)}>
          Image
        </button>
      </div>
      {linkOpen && (
        <div className="border-b border-line p-4">
          <label>
            Link URL
            <input
              type="url"
              value={href}
              onChange={(e) => setHref(e.target.value)}
              placeholder="https://…"
            />
          </label>
          <button
            type="button"
            className="text-link mt-3"
            disabled={!/^https?:\/\//.test(href)}
            onClick={() => {
              editor.send({
                type: "annotation.toggle",
                annotation: { name: "link", value: { href } },
              });
              setLinkOpen(false);
              setHref("");
            }}
          >
            Apply to selected text
          </button>
        </div>
      )}
      {imageOpen && (
        <div className="border-b border-line p-4">
          <MediaField value={image} onChange={setImage} />
          <button
            type="button"
            className="button mt-4"
            disabled={!image?.alt}
            onClick={() => {
              editor.send({
                type: "insert.blocks",
                blocks: [{ ...image!, _key: crypto.randomUUID() }],
                placement: "after",
                select: "end",
              });
              setImageOpen(false);
              setImage(undefined);
            }}
          >
            Insert image
          </button>
        </div>
      )}
    </>
  );
}
export function RichEditor({
  value,
  onChange,
}: {
  value?: RichBlock[];
  onChange: (value: RichBlock[]) => void;
}) {
  return (
    <div className="rich-editor">
      <EditorProvider
        initialConfig={{
          schemaDefinition: schema,
          initialValue:
            value as import("@portabletext/editor").PortableTextBlock[],
        }}
      >
        <NodePlugin nodes={nodes} />
        <ImmediateValuePlugin onChange={onChange} />
        <Toolbar />
        <PortableTextEditable
          className="rich-editable"
          role="textbox"
          aria-label="Article content"
          aria-multiline
          spellCheck
          renderPlaceholder={() => (
            <span className="text-muted">Start writing here…</span>
          )}
        />
      </EditorProvider>
    </div>
  );
}

function ImmediateValuePlugin({
  onChange,
}: {
  onChange: (value: RichBlock[]) => void;
}) {
  const editor = useEditor();
  return (
    <EventListenerPlugin
      on={(event) => {
        // Mutation events are batched. Read synchronously so an immediate Save
        // includes the final keystroke instead of the preceding batch.
        if (event.type === "operation" && event.origin === "local") {
          onChange(editor.getSnapshot().context.value as RichBlock[]);
        }
      }}
    />
  );
}
