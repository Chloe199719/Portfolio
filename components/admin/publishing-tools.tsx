"use client";
import { useCallback, useEffect, useState } from "react";
import { mutate, request } from "@/lib/client-api";
import { berlinToUTC } from "@/lib/scheduling";
import type { Editable, SaveAction } from "./use-draft";
type Revision = {
  id: string;
  action: string;
  createdAt: string;
  document: Editable;
};
type Schedule = {
  id: string;
  revisionId: string;
  runAt: string;
  status: string;
  error: string;
};
const berlin = (date: string) =>
  new Intl.DateTimeFormat("en-GB", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "Europe/Berlin",
  }).format(new Date(date));
export function PublishingTools({
  doc,
  dirty,
  busy,
  save,
  onRestore,
}: {
  doc: Editable;
  dirty: boolean;
  busy: boolean;
  save: (action: SaveAction) => Promise<Editable>;
  onRestore: (d: Editable) => void;
}) {
  const [revisions, setRevisions] = useState<Revision[]>([]),
    [schedules, setSchedules] = useState<Schedule[]>([]),
    [date, setDate] = useState(""),
    [error, setError] = useState(""),
    [working, setWorking] = useState(false),
    [open, setOpen] = useState(false);
  const reload = useCallback(async () => {
    const [a, b] = await Promise.all([
      request<{ revisions: Revision[] }>(
        `/v1/admin/content/${doc._id}/revisions`,
      ),
      request<{ schedules: Schedule[] }>(
        `/v1/admin/content/${doc._id}/schedules`,
      ),
    ]);
    setRevisions(a.revisions);
    setSchedules(b.schedules);
  }, [doc._id]);
  useEffect(() => {
    if (open && doc._rev) void reload().catch((e) => setError(e.message));
  }, [open, doc._rev, reload]);
  async function action(fn: () => Promise<unknown>) {
    setWorking(true);
    setError("");
    try {
      await fn();
      await reload();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setWorking(false);
    }
  }
  return (
    <details
      className="publish-tools"
      onToggle={(e) => setOpen(e.currentTarget.open)}
    >
      <summary>Revision history & scheduled publishing</summary>
      <p className="form-help mb-5">
        Explicit saves create revisions. Restoring a revision makes a new draft;
        publishing is always a separate choice.
      </p>
      {error && (
        <p role="alert" className="form-error">
          {error}
        </p>
      )}
      <label>
        Publish at · Europe/Berlin
        <input
          type="datetime-local"
          value={date}
          onChange={(e) => setDate(e.target.value)}
        />
      </label>
      <button
        className="button button-outline my-4"
        disabled={busy || working || !date}
        onClick={() =>
          action(async () => {
            const runAt = berlinToUTC(date);
            const saved = await save("save");
            await mutate(`/v1/admin/content/${doc._id}/schedules`, {
              revisionId: saved._rev,
              runAt,
            });
          })
        }
      >
        Save a revision & schedule
      </button>
      {schedules.map((s) => (
        <div className="record" key={s.id}>
          <span>
            {berlin(s.runAt)} Berlin · {s.status}
            {s.error && <span className="form-error">{s.error}</span>}
          </span>
          {s.status === "pending" && (
            <button
              disabled={working || busy}
              className="text-link"
              onClick={() =>
                action(() =>
                  mutate(
                    `/v1/admin/content/${doc._id}/schedules`,
                    undefined,
                    "DELETE",
                  ),
                )
              }
            >
              Cancel
            </button>
          )}
        </div>
      ))}
      <h3 className="mt-6">Saved revisions</h3>
      {revisions.length ? (
        revisions.map((r) => (
          <div className="record" key={r.id}>
            <span>
              {berlin(r.createdAt)} · {r.action}
            </span>
            <button
              className="text-link"
              disabled={dirty || busy || working}
              title={
                dirty
                  ? "Save current edits before restoring"
                  : "Restore as a new draft"
              }
              onClick={() =>
                action(async () => {
                  const result = await mutate<{ document: Editable }>(
                    `/v1/admin/content/${doc._id}/revisions`,
                    { revisionId: r.id, revision: doc._rev },
                  );
                  onRestore(result.document);
                })
              }
            >
              Restore draft
            </button>
          </div>
        ))
      ) : (
        <p className="form-help my-4">No explicit saves yet.</p>
      )}
    </details>
  );
}
