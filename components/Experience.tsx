import React from "react";
import { motion } from "framer-motion";
import Experirencecard from "./Experirencecard";

type Props = {};

function Experience({}: Props) {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      whileInView={{ opacity: 1 }}
      transition={{ duration: 1.5 }}
      className="h-screen flex relative overflow-hidden flex-col text-left md:flex-row  px-10 justify-evenly mx-auto items-center max-w-7xl "
    >
      <h3 className="absolute top-24 uppercase tracking-[20px] text-gray-500 text-2xl ">
        {" "}
        Experience
      </h3>
      <div className="w-full flex space-x-5 overflow-x-scroll p-10 snap-x snap-mandatory scrollbar scrollbar-track-gray-400/20 scrollbar-thumb-rose-500/80 ">
        <Experirencecard />
        <Experirencecard />
        <Experirencecard />
        <Experirencecard />
      </div>
    </motion.div>
  );
}

export default Experience;
