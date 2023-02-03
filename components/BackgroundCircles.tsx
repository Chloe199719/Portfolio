import React from "react";
import { motion } from "framer-motion";
import Image from "next/image";

type Props = {};

function BackgroundCircles({}: Props) {
  return (
    <motion.div
      initial={{
        opacity: 0,
      }}
      animate={{
        scale: [1, 2, 2, 3, 1],
        opacity: [0.1, 0.2, 0.4, 0.8, 0.1, 1.0],
        borderRadius: ["20%", "20%", "50%", "80%", "20%"],
      }}
      transition={{
        duration: 2.5,
      }}
      className="relative flex justify-center items-center"
    >
      <div className="absolute border border-[#333333] rounded-full h-[200px] w-[200px] mt-52 animate-ping" />
      <div className="absolute border border-[#333333] rounded-full h-[300px] w-[300px] mt-52 animate-ping" />
      <div className="absolute border border-[#333333] rounded-full h-[500px] w-[500px] mt-52 animate-ping" />
      <motion.div
        animate={{
          rotateY: [0, 90, 180, 270, 360, 270, 180, 90, 0],
        }}
        transition={{ repeat: Infinity, duration: 10 }}
        className="absolute border border-rose-600 opacity-20 rounded-full h-[650px] w-[650px] mt-52 animate-pulse "
      />{" "}
      <motion.div
        animate={{
          rotateY: [0, 90, 180, 270, 360, 270, 180, 90, 0],
        }}
        transition={{ repeat: Infinity, duration: 10, delay: 2 }}
        className="absolute border border-rose-600 opacity-20 rounded-full h-[650px] w-[650px] mt-52 animate-pulse "
      />
      <motion.div
        animate={{
          rotateX: [0, 90, 180, 270, 360, 270, 180, 90, 0],
        }}
        transition={{ repeat: Infinity, duration: 10 }}
        className="absolute border border-rose-600 opacity-20 rounded-full h-[650px] w-[650px] mt-52 animate-pulse "
      />{" "}
      <motion.div
        animate={{
          rotateX: [0, 90, 180, 270, 360, 270, 180, 90, 0],
        }}
        transition={{ repeat: Infinity, duration: 10, delay: 2 }}
        className="absolute border border-rose-600 opacity-20 rounded-full h-[650px] w-[650px] mt-52 animate-pulse "
      />
      <div className="absolute border border-[#333333] rounded-full h-[800px] w-[800px] mt-52 animate-ping" />
    </motion.div>
  );
}

export default BackgroundCircles;
