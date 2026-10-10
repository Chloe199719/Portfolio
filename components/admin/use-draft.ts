"use client";
import { useCallback, useEffect, useRef, useState } from "react";
import { mutate } from "@/lib/client-api";
import type { ContentDoc } from "@/lib/types";
export type Editable = ContentDoc & {
  isDraft?: boolean;
  hasPublished?: boolean;
};
export type SaveAction = "save" | "autosave" | "publish" | "unpublish";
type Recovery = { document: Editable; unset: string[]; baseRevision?: string };
export function useDraft(
  initial: Editable,
  onSaved: (doc: Editable) => void,
  onDirty: (dirty: boolean) => void,
) {
  const [doc, setDoc] = useState(initial),
    [dirty, setDirty] = useState(false),
    [busy, setBusy] = useState(""),
    [error, setError] = useState(""),
    [notice, setNotice] = useState("");
  const [editorVersion, setEditorVersion] = useState(0);
  const [recovery, setRecovery] = useState<Recovery | null>(null);
  const latest = useRef(initial),
    version = useRef(0),
    queue = useRef<Promise<unknown>>(Promise.resolve()),
    failed = useRef(false);
  const callbacks = useRef({ onSaved, onDirty });
  useEffect(() => {
    callbacks.current = { onSaved, onDirty };
  }, [onSaved, onDirty]);
  const storageKey = `chloe-draft:${initial._id}`;
  useEffect(() => {
    try {
      const raw = localStorage.getItem(storageKey);
      if (raw) {
        const value = JSON.parse(raw) as Recovery;
        if (
          value.document?._id === initial._id &&
          value.document?._type === initial._type
        )
          setRecovery(value);
      }
    } catch {
      /* An unavailable browser store does not block editing. */
    }
  }, [storageKey, initial._id, initial._type]);
  const persist = useCallback(
    (value: Editable) => {
      try {
        localStorage.setItem(
          storageKey,
          JSON.stringify({
            document: value,
            baseRevision: value._rev,
            unset: Object.keys(value).filter(
              (k) =>
                value[k as keyof Editable] === undefined && !k.startsWith("_"),
            ),
          }),
        );
      } catch {
        setNotice(
          "Browser recovery storage is unavailable. Keep this tab open until the draft saves.",
        );
      }
    },
    [storageKey],
  );
  const change = useCallback(
    (patch: Partial<ContentDoc>) => {
      const value = { ...latest.current, ...patch };
      latest.current = value;
      version.current++;
      failed.current = false;
      setDoc(value);
      setDirty(true);
      callbacks.current.onDirty(true);
      setNotice("");
      persist(value);
    },
    [persist],
  );
  const replace = useCallback(
    (value: Editable) => {
      latest.current = value;
      version.current++;
      setDoc(value);
      setEditorVersion((v) => v + 1);
      setError("");
      setNotice("Revision restored as a new draft.");
      failed.current = false;
      setDirty(false);
      callbacks.current.onDirty(false);
      callbacks.current.onSaved(value);
      try {
        localStorage.removeItem(storageKey);
      } catch {}
    },
    [storageKey],
  );
  const save = useCallback(
    (action: SaveAction): Promise<Editable> => {
      const operation = queue.current
        .catch(() => undefined)
        .then(async () => {
          const snapshot = latest.current,
            sequence = version.current;
          setBusy(action);
          setError("");
          setNotice("");
          try {
            const payload = { ...snapshot };
            delete payload._rev;
            delete payload.isDraft;
            delete payload.hasPublished;
            const result = await mutate<{ document: Editable }>(
              `/v1/admin/content/${snapshot._id}`,
              {
                action,
                revision: snapshot._rev,
                document: payload,
                unset: Object.keys(snapshot).filter(
                  (k) =>
                    snapshot[k as keyof Editable] === undefined &&
                    !k.startsWith("_"),
                ),
              },
            );
            const saved = result.document;
            if (
              !saved ||
              saved._id !== snapshot._id ||
              typeof saved._rev !== "string" ||
              saved._type !== snapshot._type
            )
              throw new Error(
                "The server did not confirm this save. Your edits are kept locally.",
              );
            const newer = version.current !== sequence;
            const next = newer
              ? {
                  ...latest.current,
                  _rev: saved._rev,
                  _updatedAt: saved._updatedAt,
                  _publishedSlug: saved._publishedSlug,
                  isDraft: saved.isDraft,
                  hasPublished: saved.hasPublished,
                }
              : saved;
            latest.current = next;
            setDoc(next);
            setDirty(newer);
            callbacks.current.onDirty(newer);
            callbacks.current.onSaved(saved);
            failed.current = false;
            if (newer) persist(next);
            else
              try {
                localStorage.removeItem(storageKey);
              } catch {}
            setNotice(
              action === "publish"
                ? `Published.${newer ? " Newer edits are still a draft." : " The public site is updated."}`
                : action === "unpublish"
                  ? "Unpublished. Your draft is safe."
                  : newer
                    ? "Saved. Newer edits are waiting to save."
                    : action === "autosave"
                      ? "Draft saved automatically."
                      : "Revision saved.",
            );
            return saved;
          } catch (e) {
            failed.current = true;
            setError((e as Error).message);
            persist(latest.current);
            throw e;
          } finally {
            setBusy("");
          }
        });
      queue.current = operation;
      return operation;
    },
    [persist, storageKey],
  );
  useEffect(() => {
    if (!dirty || busy || failed.current || recovery) return;
    const timer = setTimeout(() => {
      void save("autosave").catch(() => {});
    }, 1800);
    return () => clearTimeout(timer);
  }, [doc, dirty, busy, recovery, save]);
  useEffect(() => {
    const handler = (e: BeforeUnloadEvent) => {
      if (dirty) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", handler);
    return () => window.removeEventListener("beforeunload", handler);
  }, [dirty]);
  function recover() {
    if (!recovery) return;
    const value = { ...recovery.document, _rev: latest.current._rev };
    for (const key of recovery.unset || [])
      (value as Record<string, unknown>)[key] = undefined;
    setRecovery(null);
    change(value);
    setEditorVersion((v) => v + 1);
    setNotice(
      "Recovered browser edits. Review them against the saved version before publishing.",
    );
  }
  function discardRecovery() {
    setRecovery(null);
    try {
      localStorage.removeItem(storageKey);
    } catch {}
  }
  return {
    doc,
    editorVersion,
    dirty,
    busy,
    error,
    notice,
    change,
    save,
    replace,
    recovery,
    recover,
    discardRecovery,
  };
}
