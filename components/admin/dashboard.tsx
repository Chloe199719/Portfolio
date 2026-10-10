"use client";
import { useCallback, useEffect, useState } from "react";
import { request, mutate, type useSession } from "@/lib/client-api";
import type { ContentType, GuestEntry } from "@/lib/types";
import { formatDate } from "@/lib/utils";
import { AuthControls } from "@/components/interactive/auth-controls";
import { MediaLibrary } from "./media-library";
import { ApplicationsPanel } from "./applications-panel";
import { TrashPanel } from "./trash-panel";
import { ContentEditor, type Editable } from "./content-editor";
const tabs = [
  ["content", "Content"],
  ["guestbook", "Guestbook"],
  ["inbox", "Inbox"],
  ["media", "Media"],
  ["applications", "Applications"],
  ["trash", "Trash"],
] as const;
const contentLabels: Record<ContentType, string> = {
  pageInfo: "Profile",
  projects: "Project",
  note: "Note",
  photograph: "Photograph",
  nowUpdate: "Now update",
  siteSettings: "Homepage",
};
type Message = {
  id: string;
  name: string;
  email: string;
  subject: string;
  message: string;
  created_at: string;
  is_read: boolean;
};
export function Dashboard({
  session,
}: {
  session: ReturnType<typeof useSession>;
}) {
  const [tab, setTab] = useState("content");
  const [docs, setDocs] = useState<Editable[]>([]);
  const [selected, setSelected] = useState<Editable | null>(null);
  const [newType, setNewType] = useState<ContentType>("note");
  const [dirty, setDirty] = useState(false);
  const [pending, setPending] = useState<(() => void) | null>(null);
  const [entries, setEntries] = useState<GuestEntry[]>([]);
  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [search, setSearch] = useState(""),
    [filterType, setFilterType] = useState(""),
    [filterStatus, setFilterStatus] = useState(""),
    [page, setPage] = useState(1),
    [total, setTotal] = useState(0),
    [confirmTrash, setConfirmTrash] = useState(false);
  const reload = useCallback(async () => {
    setError("");
    setLoading(true);
    try {
      if (tab === "content") {
        const data = await request<{ documents: Editable[]; total: number }>(
          `/v1/admin/content?q=${encodeURIComponent(search)}&type=${filterType}&status=${filterStatus}&page=${page}`,
        );
        setDocs(data.documents);
        setTotal(data.total);
      } else if (tab === "guestbook")
        setEntries(
          (await request<{ entries: GuestEntry[] }>("/v1/admin/guestbook"))
            .entries,
        );
      else if (tab === "inbox")
        setMessages(
          (await request<{ messages: Message[] }>("/v1/admin/inbox")).messages,
        );
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [tab, search, filterType, filterStatus, page]);
  useEffect(() => {
    const timer = setTimeout(() => {
      void reload();
    }, 200);
    return () => clearTimeout(timer);
  }, [reload]);
  function navigate(action: () => void) {
    if (dirty) setPending(() => action);
    else action();
  }
  async function action(task: () => Promise<unknown>) {
    setBusy(true);
    setError("");
    try {
      await task();
      await reload();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="admin shell page-bottom">
      <div className="admin-header">
        <div>
          <p className="eyebrow">Your personal home</p>
          <h1>Behind the scenes.</h1>
          <p>Write something. Share a moment. Make this space yours.</p>
        </div>
        <AuthControls
          user={session.user}
          configured={session.configured}
          onChange={session.refresh}
        />
      </div>
      <div
        className="admin-tabs"
        role="tablist"
        aria-label="Dashboard sections"
      >
        {tabs.map(([id, label]) => (
          <button
            key={id}
            role="tab"
            aria-selected={tab === id}
            onClick={() =>
              navigate(() => {
                setTab(id);
                setSelected(null);
              })
            }
          >
            {label}
          </button>
        ))}
      </div>
      {pending && (
        <div className="inline-confirm">
          <span>You have unsaved changes.</span>
          <button
            onClick={() => {
              setDirty(false);
              pending();
              setPending(null);
            }}
          >
            Discard changes and continue
          </button>
          <button onClick={() => setPending(null)}>Keep editing</button>
        </div>
      )}
      {error && (
        <p role="alert" className="form-error">
          {error}{" "}
          <button className="text-link" onClick={reload}>
            Try again
          </button>
        </p>
      )}
      {tab === "media" ? (
        <div className="admin-panel">
          <MediaLibrary />
        </div>
      ) : tab === "applications" ? (
        <ApplicationsPanel />
      ) : tab === "trash" ? (
        <TrashPanel />
      ) : loading ? (
        <div className="grid gap-4 py-10">
          <div className="skeleton h-16" />
          <div className="skeleton h-16" />
        </div>
      ) : tab === "content" ? (
        <div className="admin-layout">
          <aside className="admin-list">
            <div className="space-y-3 mb-5">
              <label>
                Search content
                <input
                  type="search"
                  value={search}
                  onChange={(e) => {
                    setSearch(e.target.value);
                    setPage(1);
                  }}
                />
              </label>
              <label>
                Type
                <select
                  value={filterType}
                  onChange={(e) => {
                    setFilterType(e.target.value);
                    setPage(1);
                  }}
                >
                  <option value="">All types</option>
                  {Object.entries(contentLabels).map(([id, name]) => (
                    <option key={id} value={id}>
                      {name}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Status
                <select
                  value={filterStatus}
                  onChange={(e) => {
                    setFilterStatus(e.target.value);
                    setPage(1);
                  }}
                >
                  <option value="">All statuses</option>
                  <option value="draft">Draft</option>
                  <option value="published">Published</option>
                </select>
              </label>
            </div>
            <div className="mb-5 space-y-3">
              <label>
                New entry
                <select
                  value={newType}
                  onChange={(e) => setNewType(e.target.value as ContentType)}
                >
                  {(
                    [
                      "note",
                      "projects",
                      "photograph",
                      "nowUpdate",
                    ] as ContentType[]
                  ).map((t) => (
                    <option key={t} value={t}>
                      {contentLabels[t]}
                    </option>
                  ))}
                </select>
              </label>
              <button
                className="button w-full"
                onClick={() =>
                  navigate(() => {
                    setSelected({
                      _id: crypto.randomUUID(),
                      _type: newType,
                      isDraft: true,
                      date: new Date().toISOString().slice(0, 10),
                      order: 99,
                    });
                    setDirty(false);
                  })
                }
              >
                Create {contentLabels[newType].toLowerCase()}
              </button>
            </div>
            {docs.map((d) => (
              <button
                key={d._id}
                aria-current={selected?._id === d._id ? "true" : undefined}
                onClick={() =>
                  navigate(() => {
                    setSelected(d);
                    setDirty(false);
                  })
                }
              >
                <strong>{d.title || d.name || contentLabels[d._type]}</strong>
                <small>
                  {contentLabels[d._type]} ·{" "}
                  {d.isDraft
                    ? "Draft"
                    : d.hasPublished
                      ? "Published"
                      : "Unpublished"}
                </small>
              </button>
            ))}
            <div className="flex items-center justify-between py-5 text-xs">
              <button
                disabled={page === 1}
                onClick={() => setPage((v) => v - 1)}
              >
                Previous
              </button>
              <span>
                {page} / {Math.max(1, Math.ceil(total / 25))}
              </span>
              <button
                disabled={page * 25 >= total}
                onClick={() => setPage((v) => v + 1)}
              >
                Next
              </button>
            </div>
          </aside>
          {selected ? (
            <div className="min-w-0">
              {!["pageInfo", "siteSettings"].includes(selected._type) &&
                selected._rev && (
                  <div className="mb-6 flex flex-wrap gap-5">
                    <button
                      className="text-link text-xs"
                      disabled={busy || dirty}
                      onClick={() =>
                        action(async () => {
                          const current =
                            docs.find((d) => d._id === selected._id) ||
                            selected;
                          const result = await mutate<{ document: Editable }>(
                            `/v1/admin/content/${selected._id}/organize`,
                            { action: "duplicate", revision: current._rev },
                          );
                          setSelected(result.document);
                        })
                      }
                    >
                      Duplicate into draft
                    </button>
                    <button
                      className="text-link text-xs"
                      disabled={busy || dirty}
                      onClick={() => setConfirmTrash(true)}
                    >
                      Move to trash
                    </button>
                    {confirmTrash && (
                      <div className="inline-confirm">
                        <span>
                          This removes the entry from the public website.
                        </span>
                        <button
                          disabled={busy || dirty}
                          onClick={() =>
                            action(async () => {
                              const current =
                                docs.find((d) => d._id === selected._id) ||
                                selected;
                              await mutate(
                                `/v1/admin/content/${selected._id}/organize`,
                                { action: "trash", revision: current._rev },
                              );
                              setSelected(null);
                              setConfirmTrash(false);
                            })
                          }
                        >
                          Confirm trash
                        </button>
                        <button onClick={() => setConfirmTrash(false)}>
                          Keep entry
                        </button>
                      </div>
                    )}
                  </div>
                )}
              <ContentEditor
                key={selected._id}
                initial={selected}
                onDirty={setDirty}
                onSaved={(d) => {
                  setDocs((prev) => [
                    d,
                    ...prev.filter((v) => v._id !== d._id),
                  ]);
                }}
              />
            </div>
          ) : (
            <div className="setup-note">
              <h2>A space that grows with you.</h2>
              <p>
                Select an entry to edit, or create something new. Drafts stay
                private until you publish them.
              </p>
            </div>
          )}
        </div>
      ) : tab === "guestbook" ? (
        <div className="admin-panel">
          {entries.length ? (
            entries.map((e) => (
              <article className="admin-record" key={e.id}>
                <div className="flex justify-between gap-4">
                  <h3>{e.name}</h3>
                  <span className="text-xs text-muted">
                    {e.status} · {formatDate(e.createdAt)}
                  </span>
                </div>
                <p>{e.message}</p>
                <div className="flex gap-3">
                  <button
                    className="button"
                    disabled={busy || e.status === "approved"}
                    onClick={() =>
                      action(() =>
                        mutate(
                          `/v1/guestbook/${e.id}`,
                          { status: "approved" },
                          "PATCH",
                        ),
                      )
                    }
                  >
                    Approve
                  </button>
                  <button
                    className="button button-outline"
                    disabled={busy || e.status === "rejected"}
                    onClick={() =>
                      action(() =>
                        mutate(
                          `/v1/guestbook/${e.id}`,
                          { status: "rejected" },
                          "PATCH",
                        ),
                      )
                    }
                  >
                    Hide / reject
                  </button>
                </div>
              </article>
            ))
          ) : (
            <p className="text-sm text-muted">
              No guestbook messages to review yet.
            </p>
          )}
        </div>
      ) : (
        <div className="admin-panel">
          {messages.length ? (
            messages.map((m) => (
              <article className="admin-record" key={m.id}>
                <div className="flex justify-between gap-4">
                  <h3>{m.subject}</h3>
                  <span className="text-xs text-muted">
                    {m.is_read ? "Read" : "Unread"} · {formatDate(m.created_at)}
                  </span>
                </div>
                <p className="text-muted">
                  {m.name} · <a href={`mailto:${m.email}`}>{m.email}</a>
                </p>
                <p>{m.message}</p>
                <button
                  className="button button-outline"
                  disabled={busy}
                  onClick={() =>
                    action(() =>
                      mutate(
                        "/v1/admin/inbox",
                        { id: m.id, read: !m.is_read },
                        "PATCH",
                      ),
                    )
                  }
                >
                  {m.is_read ? "Mark unread" : "Mark read"}
                </button>
              </article>
            ))
          ) : (
            <p className="text-sm text-muted">Your inbox is clear.</p>
          )}
        </div>
      )}
    </div>
  );
}
