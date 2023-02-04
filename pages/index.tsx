import Head from "next/head";
import Header from "@/components/Header";
import Hero from "@/components/Hero";
import About from "@/components/About";
// import Experience from "@/components/Experience";
import Skills from "@/components/Skills";
import Projects from "@/components/Projects";
import ContactMe from "@/components/ContactMe";
import Link from "next/link";
import { ArrowUpIcon } from "@heroicons/react/24/solid";
import { GetStaticProps } from "next";
import { PageInfo, Project, Skill, Social } from "@/typings";

import { sanityClient } from "@/sanity";
import { groq } from "next-sanity";

type Props = {
  pageInfo: PageInfo;
  // experiences: Exp[];
  skills: Skill[];
  projects: Project[];
  socials: Social[];
};

export default function Home({
  pageInfo,
  // experiences,
  projects,
  skills,
  socials,
}: Props) {
  return (
    <div className="bg-[rgb(36,36,36)] text-white h-screen snap-y snap-mandatory overflow-y-scroll z-0 overflow-x-hidden scrollbar scrollbar-track-gray-400/20 scrollbar-thumb-rose-500/80 ">
      <Head>
        <title>Chloe`s Portfolio</title>
        <meta
          name="description"
          content="Portfolio that contains chloe`s personal Projects "
        />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="icon" href="/favicon.ico" />
      </Head>
      <Header socials={socials} />
      <section id="hero" className="snap-start">
        <Hero pageInfo={pageInfo} />
      </section>
      <section id="about" className="snap-center">
        <About pageInfo={pageInfo} />
      </section>
      {/* Experience */}
      {/* <section id="experience" className="snap-center">
        <Experience />
      </section> */}
      <section id="skills" className="snap-start">
        <Skills skillsdata={skills} />
      </section>
      <section id="projects" className="snap-start">
        <Projects projects={projects} />
      </section>
      <section id="contact" className="snap-start">
        <ContactMe pageInfo={pageInfo} />
      </section>

      <footer className="sticky bottom-5 w-full cursor-pointer">
        <Link href="#hero">
          <div className="flex items-center justify-center">
            <ArrowUpIcon className="h-10 w-10 rounded-full" />
          </div>
        </Link>
      </footer>
    </div>
  );
}

export const getStaticProps: GetStaticProps<Props> = async () => {
  const pageInfo: PageInfo = await sanityClient.fetch(groq`
  *[_type == "pageInfo"][0]
  `);
  // const experiences: Exp[] = await fetchExperience();
  const skills: Skill[] = await sanityClient.fetch(groq`
  *[_type == "Skill"]
  `);
  const projects: Project[] = await sanityClient.fetch(groq`
  *[_type == "projects"]{
    ...,
    technologies[]->
  }
  `);
  const socials: Social[] = await sanityClient.fetch(groq`
  *[_type == "social"]
  `);
  return {
    props: {
      pageInfo,
      // experiences,
      skills,
      projects,
      socials,
    },
    revalidate: 100,
  };
};
