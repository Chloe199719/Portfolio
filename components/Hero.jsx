import Image from "next/image";
import Link from "next/link";
import { type } from "os";
import React from "react";
import { Cursor, useTypewriter } from "react-simple-typewriter";
import BackgroundCircles from "./BackgroundCircles";

function Hero() {
  const myLoader = ({ src, width, quality }) => {
    return `https://cdn.discordapp.com/attachments/1024956110249148456/1069756792676036729/${src}?w=${width}&q=${
      quality || 75
    }`;
  };
  const [text, count] = useTypewriter({
    words: ["Hi , Im Chloe", "Coding is Awsome", "#HireME"],
    loop: true,
    delaySpeed: 2000,
  });
  return (
    <div className="h-screen flex flex-col space-y-8 items-center justify-center text-center overflow-hidden">
      <BackgroundCircles />
      <Image
        className="relative rounded-full h-32 w-32 mx-auto object-cover"
        loader={myLoader}
        src="IMG_0723.jpg"
        alt="chloe"
        width={200}
        height={200}
      />
      <div className="z-20">
        <h2 className="text-sm uppercase text-gray-500 pb-2 tracking-[15px]">
          Software Engineer
        </h2>
        <h1 className="text-5xl lg:text-6xl font-semibold px-10 ">
          <span className="mr-3  overflow-clip ">{text}</span>
          <Cursor cursorColor="#f7ab0a" />
        </h1>
        <div className="pt-5">
          <Link href="#about">
            <button className="heroButton">About</button>
          </Link>
          <Link href="#experience">
            <button className="heroButton">Experience</button>
          </Link>
          <Link href="#skills">
            <button className="heroButton">Skills</button>
          </Link>
          <Link href="projects">
            <button className="heroButton">Projects</button>
          </Link>
        </div>
      </div>
    </div>
  );
}

export default Hero;
