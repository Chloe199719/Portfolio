import React from "react";
import { motion } from "framer-motion";
import Image from "next/image";
import jsimg from "./images/js.png";
type Props = {};

function Experirencecard({}: Props) {
  return (
    <article className="flex flex-col rounded-lg items-center  space-y-7 flex-shrink-0 w-[500px] md:w-[600px] xl:w-[900px] snap-center bg-[#292929] p-10 hover:opacity-100 opacity-40 cursor-pointer transition-opacity duration-200 overflow-hidden">
      <motion.img
        initial={{
          y: -100,
          opacity: 0,
        }}
        transition={{ duration: 1.2 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        className="w32 h-32 rounded-full xl:w-[200px] xl:h-[200px] object-cover object-center"
        alt="computer"
        src="https://cdn.discordapp.com/attachments/292671150373339137/1070196245945126982/pexels-cottonbro-studio-5474282.jpg"
      />
      <div className=" px-0 md:px-10">
        <h4 className="text-4xl font-light">Free Lancer</h4>
        <p className="font-bold text-2xl mt-1">Web Development </p>
        <div className="flex space-x-2 my-2">
          <Image
            className="h-10 w-10 rounded-full"
            src={jsimg}
            alt="javascript"
          ></Image>
          <Image
            className="h-10 w-10 rounded-full"
            src={jsimg}
            alt="javascript"
          ></Image>
          <Image
            className="h-10 w-10 rounded-full"
            src={jsimg}
            alt="javascript"
          ></Image>
        </div>
        <p className="uppercase py-5 text-gray-300 ">
          Started work ... - Ended ...
        </p>
        <ul className=" list-disc space-y-4 ml-5 text-lg ">
          <li>Summary Points</li>
          <li>Summary Points</li>
          <li>Summary Points</li>
          <li>Summary Points</li>
          <li>Summary Points</li>
        </ul>
      </div>
    </article>
  );
}

export default Experirencecard;
