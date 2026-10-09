"use client";
import { useCallback, useEffect, useState } from "react";
import { mutate, request } from "@/lib/client-api";
import type { ContentDoc } from "@/lib/types";
export function TrashPanel() {
  const [entries, setEntries] = useState<
      { id: string; document: ContentDoc; deletedAt: string }[]
    >([]),
    [error, setError] = useState("");
  const load = useCallback(
    async () =>
      setEntries(
        (await request<{ entries: typeof entries }>("/v1/admin/trash")).entries,
      ),
    [],
  );
  useEffect(() => {
    void load().catch((e) => setError(e.message));
  }, [load]);
  return (
    <div className="admin-panel">
      <h2>Recoverable trash</h2>
      <p className="form-help my-4">
        Restored entries return as private drafts.
      </p>
      {error && (
        <p role="alert" className="form-error">
          {error}
        </p>
      )}
      {entries.map((e) => (
        <div className="admin-record" key={e.id}>
          <h3>{e.document.title}</h3>
          <button
            className="text-link mt-4"
            onClick={async () => {
              try {
                await mutate(`/v1/admin/content/${e.id}/organize`, {
                  action: "restore",
                });
                await load();
              } catch (err) {
                setError((err as Error).message);
              }
            }}
          >
            Restore draft
          </button>
        </div>
      ))}
      {!entries.length && <p className="form-help">Trash is empty.</p>}
    </div>
  );
}
