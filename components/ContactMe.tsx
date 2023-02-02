import React from "react";
import { PhoneIcon, MapPinIcon, EnvelopeIcon } from "@heroicons/react/24/solid";
import { motion } from "framer-motion";
import { useForm, SubmitHandler } from "react-hook-form";

type Props = {};

type Inputs = {
  name: string;
  email: string;
  subject: string;
  message: string;
};

function ContactMe({}: Props) {
  const { register, handleSubmit } = useForm<Inputs>();
  const onSubmit: SubmitHandler<Inputs> = (data) => console.log(data);
  return (
    <motion.div
      initial={{ opacity: 0 }}
      whileInView={{ opacity: 1 }}
      transition={{ duration: 1.5 }}
      className="h-screen flex relative flex-col text-center md:text-left md:flex-row max-w-7xl px10 justify-evenly mx-auto items-center"
    >
      <motion.h3
        initial={{ x: 500, y: 1000, scale: 0.5 }}
        whileInView={{
          x: 0,
          y: 0,
          scale: 1,
        }}
        transition={{ duration: 2 }}
        className="h3"
      >
        Contact
      </motion.h3>
      <div className="flex flex-col space-y-10">
        <h4 className="text-4xl font-semibold text-center">
          Like what you see?{" "}
          <span className="decoration-[#f7ab0a]/50 underline">Contact Me</span>
        </h4>
        <div className="space-y-10">
          <div className="flex items-center  space-x-5 justify-center">
            <PhoneIcon className="text-[#f7ab0a] h-7 w-7 animate-pulse" />
            <p className="text-2xl">+12312412341</p>
          </div>
          <div className="flex items-center  space-x-5 justify-center">
            <MapPinIcon className="text-[#f7ab0a] h-7 w-7 animate-pulse" />
            <p className="text-2xl">123 deve 51341</p>
          </div>
          <div className="flex items-center  space-x-5 justify-center">
            <EnvelopeIcon className="text-[#f7ab0a] h-7 w-7 animate-pulse" />
            <p className="text-2xl">chloe@chloevision.com</p>
          </div>
        </div>
        <form
          onSubmit={handleSubmit(onSubmit)}
          className="flex flex-col space-y-2 w-fit mx-auto"
        >
          <div className="flex space-x-2">
            <input
              {...register("name")}
              className="contactInput"
              placeholder="Name"
              type="text"
              name="name"
              id="name"
            />
            <input
              {...register("email")}
              className="contactInput"
              placeholder="Email"
              type="email"
              name="email"
              id="email"
            />
          </div>
          <input
            {...register("subject")}
            placeholder="Subject"
            className="contactInput"
            type="text"
            name="subject"
            id="subject"
          />
          <textarea
            {...register("message")}
            placeholder="Message"
            className="contactInput"
            name="content"
            id="content"
          />
          <button
            type="submit"
            className="bg-[#f7ab0a] py-5 px-10 rounded-sm text-black font-bold uppercase text-lg"
          >
            Submit
          </button>
        </form>
      </div>
    </motion.div>
  );
}

export default ContactMe;
