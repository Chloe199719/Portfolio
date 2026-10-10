import Image from "next/image";
import { getContent } from "@/lib/content";
import { imageUrl } from "@/lib/utils";
import { metadata as makeMetadata } from "@/lib/seo";
import { PageHeading, TextLink } from "@/components/site/ui";
import { RichText } from "@/components/site/rich-text";
export const metadata = makeMetadata(
  "About",
  "Meet Chloe: software engineer, photography enthusiast, and a person with interests beyond the keyboard.",
  "/about",
);
export default async function About() {
  const { profile } = await getContent();
  const portrait = imageUrl(profile.profilePic || profile.heroImage);
  return (
    <div className="shell page-bottom">
      <PageHeading
        eyebrow="The person behind the pixels"
        title="Hi, I’m Chloe."
      >
        <p>{profile.role}. And a few other things.</p>
      </PageHeading>
      <div className="about-layout">
        {portrait && (
          <figure className="about-portrait">
            <Image
              src={portrait}
              alt={`Portrait of ${profile.name}`}
              width={650}
              height={850}
              sizes="(max-width:768px) 90vw, 420px"
            />
            <figcaption>A little more than an introduction.</figcaption>
          </figure>
        )}
        <div className="about-copy">
          <h2>
            Work is one part
            <br />
            of the <em>picture.</em>
          </h2>
          <p>{profile.bio}</p>
          <RichText value={profile.body} />
          <div className="interest-list">
            {(profile.interests || ["Photography", "Fitness", "Gaming"]).map(
              (v, i) => (
                <div key={v}>
                  <span>0{i + 1}</span>
                  <h3>{v}</h3>
                </div>
              ),
            )}
          </div>
          <TextLink href="/contact">Say hello</TextLink>
        </div>
      </div>
    </div>
  );
}
