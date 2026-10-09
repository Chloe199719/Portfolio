import { metadata as makeMetadata } from "@/lib/seo";
import { PageHeading } from "@/components/site/ui";
import { MemoryGame } from "@/components/interactive/memory-game";
export const metadata = makeMetadata(
  "Playground",
  "Small experiments and a little room to play. Try a memory matching game.",
  "/playground",
);
export default function Playground() {
  return (
    <div className="shell page-bottom">
      <PageHeading
        eyebrow="A little side quest"
        title="Made for the fun of it."
      >
        <p>
          Small experiments. No big agenda. Just a place to follow an idea and
          see where it goes.
        </p>
      </PageHeading>
      <section className="playground-layout">
        <div>
          <p className="eyebrow">Experiment 001 / In your browser</p>
          <h2>
            Something
            <br />
            to <em>remember.</em>
          </h2>
          <p>
            A small matching game about paying attention. Turn over two tiles
            and find their partner.
          </p>
          <p className="text-sm text-muted mt-5">
            Use your mouse, touch, or Tab and Enter. Your best score stays in
            this browser.
          </p>
        </div>
        <MemoryGame />
      </section>
    </div>
  );
}
