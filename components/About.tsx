import React from "react";
import Image from "next/image";
import { motion } from "framer-motion";
type Props = {};
//https://cdn.discordapp.com/attachments/1024956110249148456/1069756792676036729/IMG_0723.jpg

function About({}: Props) {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      whileInView={{ opacity: 1 }}
      transition={{ duration: 1.5 }}
      className="h-screen flex flex-col relative text-center md:text-left md:flex-row max-w-7xl justify-evenly mx-auto items-center"
    >
      <h3 className="absolute top-24 uppercase tracking-[20px] text-gray-500 text-2xl ">
        {" "}
        About
      </h3>
      <motion.img
        initial={{
          x: -200,
          opacity: 0,
        }}
        whileInView={{ x: 0, opacity: 1 }}
        transition={{
          duration: 1.2,
        }}
        viewport={{ once: true }}
        src="https://cdn.discordapp.com/attachments/1024956110249148456/1069756792676036729/IMG_0723.jpg"
        className="-mb-20 md:mb-0 flex-shrink-0 w-56 h-56 rounded-full object-cover md:rounded-lg md:w-64 md:h-95 xl:w-[500px] xl:h-[600px] "
      />
      <div className="space-y-10 px-0 md:px-10">
        <h4 className="text-4xl font-semibold">
          Here is a{" "}
          <span className="underline decoration-[#f7ab0a]"> little</span>{" "}
          background
        </h4>
        <p className="text-base">
          Im Chloe Self Taught Software Developer and here a bit about me im 26
          years Old with passion to build things becoming a software developer
          has enhanced this passion greatly with many Real Life Projects that i
          was so far able to make and more to come you can find a bit more about
          my skills down below and my projects{" "}
        </p>
      </div>
    </motion.div>
  );
}

export default About;
