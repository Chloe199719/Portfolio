import React from "react";
import { PhoneIcon, MapPinIcon, EnvelopeIcon } from "@heroicons/react/24/solid";
import { motion } from "framer-motion";
import { useForm, SubmitHandler } from "react-hook-form";
import { PageInfo } from "@/typings";
import { db } from "../firebase";
import { addDoc, collection, serverTimestamp } from "firebase/firestore";
import { useState } from "react";
import Alert from "./alert";

type Props = {
  pageInfo: PageInfo;
};

type Inputs = {
  name: string;
  email: string;
  subject: string;
  message: string;
};

function ContactMe({ pageInfo }: Props) {
  const { register, handleSubmit, reset } = useForm<Inputs>();
  const [load, setLoad] = useState(false);
  const [error, setError] = useState("");
  const [sent, setSent] = useState("");
  const SendData = async function (data: any) {
    setLoad(true);
    setSent("");
    setError("");
    try {
      await addDoc(collection(db, "messages"), {
        email: data.email,
        message: data.message,
        subject: data.subject,
        name: data.name,
        timestamp: serverTimestamp(),
      });
      reset();
      setSent("Your Message was Sent");
      setLoad(false);
    } catch (e) {
      setError("Error delivering your message to our servers");
      setLoad(false);
    }
  };
  const onSubmit: SubmitHandler<Inputs> = (data) => {
    SendData(data);
  };
  return (
    <motion.div
      initial={{ opacity: 0 }}
      whileInView={{ opacity: 1 }}
      transition={{ duration: 1.5 }}
      className="h-screen flex relative flex-col text-center md:text-left md:flex-row max-w-7xl px10 justify-evenly mx-auto items-center"
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
        Contact
      </motion.h3>
      <div className="flex flex-col space-y-10">
        <motion.h4
          initial={{
            x: -300,
          }}
          whileInView={{
            x: 0,
          }}
          viewport={{ once: true }}
          transition={{ duration: 1.5 }}
          className="text-4xl font-semibold text-center"
        >
          Like what you see?{" "}
          <span className="decoration-rose-300/50 underline">Contact Me</span>
        </motion.h4>
        <div className="space-y-10">
          <motion.div
            initial={{
              x: 300,
            }}
            whileInView={{
              x: 0,
            }}
            viewport={{ once: true }}
            transition={{ duration: 1.5 }}
            className="flex items-center  space-x-5 justify-center"
          >
            <PhoneIcon className="text-rose-300 h-7 w-7 animate-pulse" />
            <p className="text-2xl">{pageInfo?.phoneNumber}</p>
          </motion.div>
          <motion.div
            initial={{
              x: -300,
            }}
            whileInView={{
              x: 0,
            }}
            viewport={{ once: true }}
            transition={{ duration: 1.5 }}
            className="flex items-center  space-x-5 justify-center"
          >
            <MapPinIcon className="text-rose-300 h-7 w-7 animate-pulse" />
            <p className="text-2xl">{pageInfo?.address}</p>
          </motion.div>
          <motion.div
            initial={{
              x: 300,
            }}
            whileInView={{
              x: 0,
            }}
            viewport={{ once: true }}
            transition={{ duration: 1.5 }}
            className="flex items-center  space-x-5 justify-center"
          >
            <EnvelopeIcon className="text-rose-300 h-7 w-7 animate-pulse" />
            <p className="text-2xl">{pageInfo?.email}</p>
          </motion.div>
        </div>
        <form
          onSubmit={handleSubmit(onSubmit)}
          className="flex flex-col space-y-2 w-fit mx-auto"
        >
          <div className="flex space-x-2">
            <motion.input
              initial={{
                x: -300,
              }}
              whileInView={{
                x: 0,
              }}
              viewport={{ once: true }}
              transition={{ duration: 1.0 }}
              {...register("name", { required: true })}
              className="contactInput"
              placeholder="Name"
              type="text"
              name="name"
              id="name"
              required
            />
            <motion.input
              initial={{
                x: 300,
              }}
              whileInView={{
                x: 0,
              }}
              viewport={{ once: true }}
              transition={{ duration: 1.0 }}
              {...register("email", { required: true })}
              className="contactInput"
              placeholder="Email"
              type="email"
              name="email"
              required
              id="email"
            />
          </div>
          <motion.input
            initial={{
              x: -200,
            }}
            whileInView={{
              x: 0,
            }}
            viewport={{ once: true }}
            transition={{ duration: 1.2 }}
            {...register("subject", { required: true })}
            placeholder="Subject"
            className="contactInput"
            required
            type="text"
            name="subject"
            id="subject"
          />
          <motion.textarea
            initial={{
              x: 300,
            }}
            whileInView={{
              x: 0,
            }}
            viewport={{ once: true }}
            transition={{ duration: 1 }}
            {...register("message", { required: true })}
            placeholder="Message"
            className="contactInput"
            name="message"
            id="message"
          />
          <motion.button
            initial={{
              y: 200,
            }}
            whileInView={{
              rotate: [0, 360, 180, 0],
              y: 0,
            }}
            viewport={{ once: true }}
            transition={{ duration: 2 }}
            type="submit"
            disabled={load}
            className="bg-rose-400 py-5 px-10 rounded-sm text-black font-bold uppercase text-lg"
          >
            Submit
          </motion.button>
          {error ? <Alert color="red" message={error} func={setError} /> : null}
          {sent ? <Alert color="green" message={sent} func={setSent} /> : null}
        </form>
      </div>
    </motion.div>
  );
}

export default ContactMe;
