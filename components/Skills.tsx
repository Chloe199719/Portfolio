import React from "react";
import { motion } from "framer-motion";
import Skill from "./Skill";
import { Skill as Skil } from "@/typings";

type Props = {
  skillsdata: Skil[];
};

function Skills({ skillsdata }: Props) {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      whileInView={{ opacity: 1 }}
      transition={{ duration: 1.5 }}
      className="h-screen flex relative flex-col text-center md:text-left xl:flex-row max-w-[2000px] xl:px-10 min-h-screen justify-center xl:space-y-0 mx-auto items-center"
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
        {" "}
        Skills
      </motion.h3>
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
        className="absolute top-36 uppercase tracking-[3px] text-gray-500 text-sm "
      >
        Hover over a skill for a currency profieciency
      </motion.h3>

      <div className="grid grid-cols-4 gap-5">
        {skillsdata?.slice(0, skillsdata.length / 2).map((e) => {
          return <Skill key={e._id} skill={e} directionLeft={true} />;
        })}
        {skillsdata
          ?.slice(skillsdata.length / 2, skillsdata.length)
          .map((e) => {
            return <Skill key={e._id} skill={e} />;
          })}
      </div>
    </motion.div>
  );
}

export default Skills;
