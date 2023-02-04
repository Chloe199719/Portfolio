import React from "react";
import Image from "next/image";
import { motion } from "framer-motion";
import { PageInfo } from "@/typings";
import { urlFor } from "@/sanity";
type Props = {
  pageInfo: PageInfo;
};
//https://cdn.discordapp.com/attachments/1024956110249148456/1069756792676036729/IMG_0723.jpg

function About({ pageInfo }: Props) {
  return (
    <motion.div
      initial={{ opacity: 0 }}
      whileInView={{ opacity: 1 }}
      transition={{ duration: 1.5 }}
      className="h-screen flex flex-col relative text-center md:text-left md:flex-row max-w-7xl justify-evenly mx-auto items-center"
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
        className="absolute top-24 uppercase tracking-[20px] text-gray-500 text-2xl "
      >
        {" "}
        About
      </motion.h3>
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
        src={urlFor(pageInfo?.profilePic).url()}
        className="-mb-20 md:mb-0 flex-shrink-0 w-56 h-56 rounded-full object-cover md:rounded-lg md:w-64 md:h-95 xl:w-[500px] xl:h-[600px] "
      />
      <motion.div
        initial={{
          x: 200,
          opacity: 0,
        }}
        whileInView={{ x: 0, opacity: 1, rotate: [360, 0] }}
        transition={{
          duration: 1.2,
        }}
        viewport={{ once: true }}
        className="space-y-10 px-0 md:px-10"
      >
        <h4 className="text-4xl font-semibold">
          Here is a{" "}
          <span className="underline decoration-rose-400"> little</span>{" "}
          background
        </h4>
        <p className="text-base">{pageInfo?.backgroundInformation}</p>
      </motion.div>
    </motion.div>
  );
}

export default About;
