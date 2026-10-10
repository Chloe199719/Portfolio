import Link from "next/link";
import Image from "next/image";
import { ArrowRightIcon, ArrowTopRightIcon } from "@radix-ui/react-icons";
import type { ReactNode } from "react";
import type { Project } from "@/lib/types";
import { imageUrl } from "@/lib/utils";
export function TextLink({
  href,
  children,
  external = false,
}: {
  href: string;
  children: ReactNode;
  external?: boolean;
}) {
  const content = (
    <>
      {children}
      {external ? <ArrowTopRightIcon /> : <ArrowRightIcon />}
    </>
  );
  return external ? (
    <a
      className="text-link"
      href={href}
      target="_blank"
      rel="noopener noreferrer"
    >
      {content}
    </a>
  ) : (
    <Link className="text-link" href={href}>
      {content}
    </Link>
  );
}
export function PageHeading({
  eyebrow,
  title,
  children,
}: {
  eyebrow: string;
  title: string;
  children?: ReactNode;
}) {
  return (
    <div className="page-heading">
      <p className="eyebrow">{eyebrow}</p>
      <h1>{title}</h1>
      {children && <div className="page-intro">{children}</div>}
    </div>
  );
}
export function SectionHeading({
  number,
  title,
  href,
  link,
}: {
  number: string;
  title: string;
  href?: string;
  link?: string;
}) {
  return (
    <div className="section-heading">
      <div className="flex items-baseline gap-5">
        <span className="section-number">{number}</span>
        <h2>{title}</h2>
      </div>
      {href && <TextLink href={href}>{link || "Explore"}</TextLink>}
    </div>
  );
}
export function EmptyState({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <div className="empty-state">
      <svg
        className="small-flower"
        width="32"
        height="32"
        viewBox="0 0 80 80"
        fill="none"
        aria-hidden="true"
      >
        <path
          d="M40 3v74M3 40h74M14 14l52 52M14 66l52-52"
          stroke="currentColor"
          strokeWidth="3"
        />
      </svg>
      <h2>{title}</h2>
      <div className="max-w-lg text-muted leading-relaxed">{children}</div>
    </div>
  );
}
export function ProjectCard({
  project,
  index = 0,
}: {
  project: Project;
  index?: number;
}) {
  const src = imageUrl(project.image);
  return (
    <article className="project-card">
      <Link
        href={`/work/${project.slug.current}`}
        className={`project-visual project-tone-${index % 3}`}
        aria-label={`Explore ${project.title}`}
      >
        <span className="project-index">0{index + 1} / SELECTED WORK</span>
        {src ? (
          <Image
            src={src}
            alt={
              project.image?.alt || `${project.title} application screenshot`
            }
            width={1200}
            height={800}
            className="project-shot"
            sizes="(max-width: 767px) 90vw, 45vw"
          />
        ) : (
          <div className="project-placeholder" aria-hidden="true">
            {project.title.slice(0, 1)}
          </div>
        )}
        <span className="project-arrow">
          <ArrowTopRightIcon />
        </span>
      </Link>
      <div className="project-caption">
        <div>
          <p className="eyebrow">
            {project.technologyNames.slice(0, 3).join(" · ") ||
              "Independent project"}
          </p>
          <h3>
            <Link href={`/work/${project.slug.current}`}>{project.title}</Link>
          </h3>
        </div>
        <span className="text-muted text-sm">
          {project._createdAt?.slice(0, 4)}
        </span>
      </div>
      <p className="project-description">{project.summary}</p>
    </article>
  );
}
