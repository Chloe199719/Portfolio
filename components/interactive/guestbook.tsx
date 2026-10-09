"use client";
import { useCallback, useEffect, useState } from "react";
import { ArrowTopRightIcon } from "@radix-ui/react-icons";
import type { GuestEntry } from "@/lib/types";
import { formatDate } from "@/lib/utils";
import { mutate, request, useSession } from "@/lib/client-api";
import { AuthControls } from "./auth-controls";
export function Guestbook({
  initialEntries,
  storageReady,
}: {
  initialEntries: GuestEntry[];
  storageReady: boolean;
}) {
  const session = useSession();
  const [entries, setEntries] = useState(initialEntries);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);
  const refresh = useCallback(async () => {
    const data = await request<{ entries: GuestEntry[] }>("/v1/guestbook");
    setEntries(data.entries);
  }, []);
  useEffect(() => {
    void refresh().catch((e) => setError(e.message));
  }, [session.user, refresh]);
  return (
    <div className="guestbook-layout">
      <aside className="guestbook-compose">
        <h2>Leave a little hello.</h2>
        <p>
          A recommendation, a shared interest, or just your name in the margins.
        </p>
        {session.loading ? (
          <div className="skeleton h-12" aria-label="Checking sign-in" />
        ) : (
          <AuthControls
            user={session.user}
            configured={session.configured}
            onChange={session.refresh}
          />
        )}
        <p className="form-error" role={session.error ? "alert" : undefined}>
          {session.error}
        </p>
        {session.user && (
          <form
            key={session.user.uid}
            onSubmit={async (e) => {
              e.preventDefault();
              const form = e.currentTarget;
              setBusy(true);
              setError("");
              setNotice("");
              try {
                await mutate(
                  "/v1/guestbook",
                  Object.fromEntries(new FormData(form)),
                );
                form.reset();
                await refresh();
                setNotice(
                  "Your note is waiting for approval. Thanks for stopping by.",
                );
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <label>
              Public name
              <input
                name="name"
                defaultValue={session.user.name}
                required
                maxLength={60}
              />
            </label>
            <label>
              Your note
              <textarea
                name="message"
                rows={4}
                maxLength={500}
                minLength={2}
                required
                placeholder="Leave something kind…"
              />
            </label>
            <p className="form-help">
              Your name and note will be public after approval. Your email stays
              private.
            </p>
            <button className="button mt-4" disabled={busy || !storageReady}>
              {busy ? "Saving…" : "Leave a note"}
              <ArrowTopRightIcon />
            </button>
          </form>
        )}
        {notice && (
          <p role="status" className="form-success">
            {notice}
          </p>
        )}
        {error && (
          <p role="alert" className="form-error">
            {error}
          </p>
        )}
      </aside>
      <div className="guestbook-entries">
        {entries.length ? (
          entries.map((entry) => (
            <article key={entry.id}>
              <div className="flex justify-between gap-4">
                <div className="flex items-center gap-3">
                  <span className="guest-avatar" aria-hidden="true">
                    {entry.name.slice(0, 1).toUpperCase()}
                  </span>
                  <div>
                    <h3>{entry.name}</h3>
                    <span className="text-xs text-muted">
                      {formatDate(entry.createdAt)}
                    </span>
                  </div>
                </div>
                {entry.own && (
                  <button
                    className="text-link text-xs"
                    onClick={() => setConfirmDelete(entry.id)}
                  >
                    Delete
                  </button>
                )}
              </div>
              <p>{entry.message}</p>
              {entry.status !== "approved" && (
                <span className="entry-status">
                  {entry.status === "pending"
                    ? "Waiting for approval · only visible to you"
                    : "Not published · only visible to you"}
                </span>
              )}
              {confirmDelete === entry.id && (
                <div className="inline-confirm">
                  <span>Delete your note?</span>
                  <button
                    onClick={async () => {
                      try {
                        await mutate(
                          `/v1/guestbook/${entry.id}`,
                          undefined,
                          "DELETE",
                        );
                        setConfirmDelete(null);
                        await refresh();
                      } catch (e) {
                        setError((e as Error).message);
                      }
                    }}
                  >
                    Delete note
                  </button>
                  <button onClick={() => setConfirmDelete(null)}>
                    Keep it
                  </button>
                </div>
              )}
            </article>
          ))
        ) : (
          <div className="guestbook-empty">
            <span className="empty-quotes" aria-hidden="true">
              “
            </span>
            <h2>
              Every guestbook starts
              <br />
              with one hello.
            </h2>
            <p>There’s a place here for yours.</p>
          </div>
        )}
      </div>
    </div>
  );
}
