import { PortableText } from "@portabletext/react";
import Image from "next/image";
import type { ImageAsset, RichBlock } from "@/lib/types";
import { imageUrl } from "@/lib/utils";
export function RichText({
  value,
  preview = false,
}: {
  value?: RichBlock[];
  preview?: boolean;
}) {
  if (!value?.length) return null;
  return (
    <div className="prose">
      <PortableText
        value={value}
        components={{
          types: {
            image: ({ value }: { value: ImageAsset }) => {
              const src = imageUrl(value);
              return src ? (
                <figure>
                  <Image
                    unoptimized={preview}
                    src={src}
                    alt={value.alt || ""}
                    width={1400}
                    height={1000}
                    sizes="(max-width: 768px) 90vw, 760px"
                  />
                  {value.caption && <figcaption>{value.caption}</figcaption>}
                </figure>
              ) : null;
            },
          },
          marks: {
            link: ({ children, value }) => {
              const href =
                typeof value?.href === "string" &&
                /^(https?:\/\/|\/(?!\/))/.test(value.href)
                  ? value.href
                  : undefined;
              return href ? (
                <a
                  href={href}
                  target={href.startsWith("/") ? undefined : "_blank"}
                  rel="noopener noreferrer"
                >
                  {children}
                </a>
              ) : (
                <>{children}</>
              );
            },
          },
        }}
      />
    </div>
  );
}
