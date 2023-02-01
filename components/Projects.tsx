import Image from "next/image";
import React from "react";
import project1img from "./images/project1.png";
import { motion } from "framer-motion";

type Props = {};

function Projects({}: Props) {
  const projects = [1, 2, 3, 4, 5];
  return (
    <motion.div
      initial={{ opacity: 0 }}
      whileInView={{ opacity: 1 }}
      transition={{ duration: 1.5 }}
      className="h-screen relative flex overflow-hidden flex-col text-left md:flex-row max-w-full justify-evenly mx-auto items-center z-0"
    >
      <h3 className="h3">Projects</h3>
      <div className=" relative w-full flex overflow-x-scroll overflow-y-hidden snap-x snap-mandatory z20 scrollbar-thin scrollbar-track-gray-400/20 scrollbar-thumb-rose-500/80 ">
        {projects.map((p, i) => (
          <div
            className="w-screen flex-shrink-0 snap-center flex flex-col space-y-5 items-center justify-center p-20 md:p-44 h-screen"
            key={p}
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
                src={project1img}
                alt="project"
              />
            </motion.div>
            <div className="space-y-10 px-0 md:px10 max-w-6xl">
              <h4 className="text-4xl font-semibold text-center">
                <span className="underline decoration-[#f7ab0a]/50">
                  Case study {i + 1}of {projects.length}
                </span>
                Clone
              </h4>
              <p className="text-lg text-center md:text-left">
                Lorem ipsum dolor sit amet consectetur, adipisicing elit. Odit
                perferendis reprehenderit velit ab vel ipsum accusamus et!
                Corrupti possimus quam nesciunt quod, repudiandae quibusdam
                eaque blanditiis necessitatibus earum dolorum. Aut.
              </p>
            </div>
          </div>
        ))}
        {/* Projects */}
        {/* Projects */}
        {/* Projects */}
      </div>
      <div className="w-full absolute top-[30] bg-[#f7ab0a]/10 left-0 h-[500px] -skew-y-[10deg]"></div>
    </motion.div>
  );
}

export default Projects;
