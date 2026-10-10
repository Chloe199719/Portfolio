import { getContent } from "@/lib/content";
import { metadata as makeMetadata } from "@/lib/seo";
import { PageHeading, ProjectCard } from "@/components/site/ui";
export const metadata = makeMetadata(
  "Work",
  "Software projects, experiments, and the thinking behind them.",
  "/work",
);
export default async function Work() {
  const { projects } = await getContent();
  return (
    <div className="shell page-bottom">
      <PageHeading
        eyebrow="A collection of projects"
        title="Built with curiosity."
      >
        <p>
          From useful tools to small games. A selection of things I’ve brought
          to life on the web.
        </p>
      </PageHeading>
      <div className="project-grid">
        {projects.map((p, i) => (
          <ProjectCard key={p._id} project={p} index={i} />
        ))}
      </div>
    </div>
  );
}
