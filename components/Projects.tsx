import Image from "next/image";
import React from "react";
import project1img from "./images/project1.png";
import { motion } from "framer-motion";
import { Project } from "@/typings";
import { urlFor } from "@/sanity";

type Props = {
  projects: Project[];
};

function Projects({ projects }: Props) {
  // const projects = [1, 2, 3, 4, 5];
  return (
    <motion.div
      initial={{ opacity: 0 }}
      whileInView={{ opacity: 1 }}
      transition={{ duration: 1.5 }}
      className="h-screen relative flex overflow-hidden flex-col text-left md:flex-row max-w-full justify-evenly mx-auto items-center z-0"
    >
      <motion.h3
        initial={{
          y: 300,
          opacity: 0,
        }}
        whileInView={{ x: 0, opacity: 1, y: 0 }}
        transition={{
          duration: 1,
        }}
        viewport={{ once: true }}
        className="h3"
      >
        Projects
      </motion.h3>
      <div className=" relative w-full flex overflow-x-scroll overflow-y-hidden snap-x snap-mandatory z20 scrollbar-thin scrollbar-track-gray-400/20 scrollbar-thumb-rose-500/80 ">
        {projects?.map((p, i) => (
          <div
            className="w-screen flex-shrink-0 snap-center flex flex-col space-y-5 items-center justify-center p-20 md:p-44 h-screen"
            key={i}
          >
            <motion.div
              initial={{
                y: -300,
                opacity: 0,
              }}
              transition={{ duration: 1.2 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
            >
              <Image
                className="w-[660px] h-[375px]"
                src={urlFor(p?.image).url()}
                alt="project"
                width={400}
                height={500}
              />
            </motion.div>
            <div className="space-y-10 px-0 md:px10 max-w-6xl">
              <h4 className="text-4xl font-semibold text-center">
                <span className="underline decoration-rose-400/50">
                  {`Case study ${i + 1} of ${projects.length}: `}
                </span>
                <a href={p.linkToBuild}> {p.title}</a>
              </h4>
              <div className="flex items-center space-x-2 justify-center">
                {p?.technologies.map((tech) => {
                  return (
                    <Image
                      key={tech._id}
                      alt={tech.title}
                      src={urlFor(tech.image).url()}
                      width={400}
                      height={500}
                      className="h-10 w-10"
                    />
                  );
                })}
              </div>
              <p className="text-lg text-center md:text-left">{p.summary}</p>
            </div>
          </div>
        ))}
        {/* Projects */}
        {/* Projects */}
        {/* Projects */}
      </div>
      <div className="w-full absolute top-[30] bg-rose-500/5 left-0 h-[500px] -skew-y-[10deg] z-[-1]"></div>
      <div className="w-full absolute top-[30] bg-rose-500/5 left-0 h-[500px] -skew-y-[-10deg] z-[-1]"></div>
    </motion.div>
  );
}

export default Projects;
