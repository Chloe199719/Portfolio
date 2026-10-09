"use client";
import { useState } from "react";
import { ArrowTopRightIcon, CheckIcon } from "@radix-ui/react-icons";
import { mutate } from "@/lib/client-api";
export function ContactForm({ configured }: { configured: boolean }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [sent, setSent] = useState(false);
  return (
    <form
      className="contact-form"
      onSubmit={async (event) => {
        event.preventDefault();
        const form = event.currentTarget;
        setError("");
        setSent(false);
        setBusy(true);
        try {
          await mutate("/v1/contact", Object.fromEntries(new FormData(form)));
          setSent(true);
          form.reset();
        } catch (e) {
          setError((e as Error).message);
        } finally {
          setBusy(false);
        }
      }}
    >
      <div className="grid gap-6 sm:grid-cols-2">
        <label>
          Your name
          <input
            name="name"
            autoComplete="name"
            maxLength={100}
            required
            placeholder="How should I call you?"
          />
        </label>
        <label>
          Email address
          <input
            name="email"
            type="email"
            autoComplete="email"
            maxLength={254}
            required
            placeholder="you@example.com"
          />
        </label>
      </div>
      <label>
        What’s on your mind?
        <input
          name="subject"
          required
          maxLength={200}
          placeholder="A project, a question, or just a hello"
        />
      </label>
      <label>
        Your message
        <textarea
          name="message"
          rows={6}
          required
          minLength={10}
          maxLength={5000}
          placeholder="Tell me a little about it…"
        />
      </label>
      <div className="honeypot" aria-hidden="true">
        <label>
          Website
          <input name="website" tabIndex={-1} autoComplete="off" />
        </label>
      </div>
      <div className="flex items-center justify-between gap-6 flex-wrap">
        <button className="button" type="submit" disabled={busy || !configured}>
          {busy ? "Sending…" : "Send your message"}
          <ArrowTopRightIcon />
        </button>
        <span className="text-xs text-muted">
          Straight to my inbox. No mailing lists.
        </span>
      </div>
      {!configured && (
        <p className="form-help">
          The form is currently unavailable. Please use the email link instead.
        </p>
      )}
      {sent && (
        <p role="status" className="form-success flex items-center gap-2">
          <CheckIcon />
          Your message is in my inbox. Thanks for reaching out.
        </p>
      )}
      {error && (
        <p role="alert" className="form-error">
          {error} Your message is still here.
        </p>
      )}
    </form>
  );
}
