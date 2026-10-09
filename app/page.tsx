import Image from "next/image";
import Link from "next/link";
import { ArrowTopRightIcon, PlusIcon } from "@radix-ui/react-icons";
import { getContent } from "@/lib/content";
import { metadata } from "@/lib/seo";
import { formatDate, imageUrl, safeJson, siteUrl } from "@/lib/utils";
import { ProjectCard, SectionHeading, TextLink } from "@/components/site/ui";
import type { ContentDoc } from "@/lib/types";
export async function generateMetadata() {
  const { settings } = await getContent();
  return metadata(
    settings.homepageTitle || "Software. Side quests. Me.",
    settings.homepageSubtitle ||
      "Software projects, photography, and life outside the browser. By Chloe Pratas.",
    "/",
  );
}
export default async function Home() {
  const { profile, settings, projects, photos, notes, updates } =
    await getContent();
  const portrait = imageUrl(profile.heroImage);
  const select = <T extends ContentDoc>(items: T[], max: number) =>
    (items.some((v) => v.featured) ? items.filter((v) => v.featured) : items)
      .toSorted((a, b) => (a.order ?? 99) - (b.order ?? 99))
      .slice(0, max);
  const selected = select(projects, 4),
    picks = select(photos, 3),
    recent = select(notes, 3),
    now = select(updates, 4);
  const modules = settings.modules || [
    { id: "work", visible: true },
    { id: "photos", visible: true },
    { id: "now", visible: true },
    { id: "notes", visible: true },
    { id: "guestbook", visible: true },
  ];
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: safeJson({
            "@context": "https://schema.org",
            "@type": "Person",
            name: profile.name,
            jobTitle: profile.role,
            url: siteUrl,
          }),
        }}
      />
      <div className="shell">
        <section className="play-hero" id="hero">
          <div className="play-hero-copy">
            <p className="eyebrow">
              <span className="status-dot" /> Chloe Pratas / {profile.role}
            </p>
            <h1>
              {!settings.homepageTitle ||
              settings.homepageTitle ===
                "Software, side quests, and everything in between." ? (
                <>
                  <span>Software.</span>
                  <span>Side quests.</span>
                  <span className="text-rose">
                    Me
                    <span className="hero-cursor" aria-hidden="true">
                      _
                    </span>
                  </span>
                </>
              ) : (
                settings.homepageTitle
              )}
            </h1>
            <p>{profile.intro}</p>
            <div className="hero-actions">
              <Link href="/work" className="button">
                Explore my work <ArrowTopRightIcon />
              </Link>
              <TextLink href="/about">Meet the person</TextLink>
            </div>
          </div>
          <div className="play-portrait">
            {portrait && (
              <Image
                src={portrait}
                alt={`Portrait of ${profile.name}`}
                width={700}
                height={850}
                priority
                sizes="(max-width: 767px) 75vw, 400px"
              />
            )}
            <span className="portrait-coordinate">
              A PERSON BEHIND THE PIXELS
            </span>
            <div className="orbit-mark" aria-hidden="true">
              <span />
              <span />
              <span />
            </div>
            <Link href="/playground" className="portrait-game">
              <span className="game-symbol" aria-hidden="true">
                ↗
              </span>
              <span>
                A little detour.<small>Play something</small>
              </span>
            </Link>
          </div>
        </section>
        <div className="intro-strip" id="skills">
          <span>Curiosity goes beyond the keyboard.</span>
          <div>
            <span>Photography</span>
            <PlusIcon />
            <span>Fitness</span>
            <PlusIcon />
            <span>Gaming</span>
          </div>
        </div>
        {modules
          .filter((m) => m.visible)
          .map((m) => {
            if (m.id === "work" && selected.length)
              return (
                <section className="section-space" id="projects" key={m.id}>
                  <SectionHeading
                    number="01"
                    title="Built with curiosity."
                    href="/work"
                    link="All projects"
                  />
                  <div className="project-grid home-projects">
                    {selected.map((p, i) => (
                      <ProjectCard key={p._id} project={p} index={i} />
                    ))}
                  </div>
                </section>
              );
            if (m.id === "photos" && picks.length)
              return (
                <section className="section-space" key={m.id}>
                  <SectionHeading
                    number="02"
                    title="Outside the browser."
                    href="/photography"
                    link="Through my lens"
                  />
                  <div className="home-photos">
                    {picks.map((p) => (
                      <Link key={p._id} href="/photography">
                        <Image
                          unoptimized
                          src={imageUrl(p.image)!}
                          alt={p.image?.alt || p.title || ""}
                          width={900}
                          height={1100}
                        />
                        <span>{p.title}</span>
                      </Link>
                    ))}
                  </div>
                </section>
              );
            if (m.id === "now" && now.length)
              return (
                <section className="section-space" key={m.id} id="about">
                  <SectionHeading
                    number="03"
                    title="In progress."
                    href="/now"
                    link="Life, lately"
                  />
                  <div className="now-grid">
                    {now.map((u) => (
                      <Link key={u._id} href="/now" className="life-item">
                        <p className="eyebrow">{u.category || "An update"}</p>
                        <h3>{u.title}</h3>
                        <p>{u.summary}</p>
                        <span className="life-date">{formatDate(u.date)}</span>
                      </Link>
                    ))}
                  </div>
                </section>
              );
            if (m.id === "notes" && recent.length)
              return (
                <section className="section-space" key={m.id}>
                  <SectionHeading
                    number="04"
                    title="Notes to keep."
                    href="/notes"
                    link="All notes"
                  />
                  <div className="note-list">
                    {recent.map((n) => (
                      <Link
                        href={`/notes/${n.slug?.current}`}
                        key={n._id}
                        className="note-row"
                      >
                        <span>{formatDate(n.date)}</span>
                        <h3>{n.title}</h3>
                        <ArrowTopRightIcon />
                      </Link>
                    ))}
                  </div>
                </section>
              );
            if (m.id === "guestbook")
              return (
                <section className="guestbook-invite" id="contact" key={m.id}>
                  <p className="eyebrow">
                    A small corner of a very big internet
                  </p>
                  <div>
                    <h2>
                      You made it here.
                      <br />
                      <em>Leave a little hello.</em>
                    </h2>
                    <TextLink href="/guestbook">Open the guestbook</TextLink>
                  </div>
                  <span className="invite-cross" aria-hidden="true">
                    ✳
                  </span>
                </section>
              );
            return null;
          })}
        {!now.length && <span id="about" />}
      </div>
    </>
  );
}
