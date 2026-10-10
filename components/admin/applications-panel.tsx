"use client";
import { useCallback, useEffect, useState } from "react";
import { mutate, request } from "@/lib/client-api";
type Application = {
  id: string;
  name: string;
  disabled: boolean;
  client: { public: boolean; redirect_uris: string[]; scopes: string[] };
  origins: string[];
};
export function ApplicationsPanel() {
  const [apps, setApps] = useState<Application[]>([]),
    [error, setError] = useState(""),
    [credential, setCredential] = useState<{
      id: string;
      secret: string;
    } | null>(null),
    [busy, setBusy] = useState(false);
  const load = useCallback(
    async () =>
      setApps(
        (
          await request<{ applications: Application[] }>(
            "/v1/admin/applications",
          )
        ).applications,
      ),
    [],
  );
  useEffect(() => {
    void load().catch((e) => setError(e.message));
  }, [load]);
  async function action(fn: () => Promise<void>) {
    setBusy(true);
    setError("");
    try {
      await fn();
      await load();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="admin-panel">
      <h2>Connected projects</h2>
      <p className="form-help my-4">
        Register applications that can use Chloe ID. All clients use
        Authorization Code with PKCE S256. Register every callback exactly.
      </p>
      {error && (
        <p role="alert" className="form-error">
          {error}
        </p>
      )}
      {credential && (
        <div className="setup-note">
          <h3>Application credentials</h3>
          <p>
            Client ID: <code className="break-all">{credential.id}</code>
          </p>
          {credential.secret ? (
            <>
              <p>
                Secret: <code className="break-all">{credential.secret}</code>
              </p>
              <p>
                Store this in the application server. It will not be shown
                again.
              </p>
            </>
          ) : (
            <p>This public client has no secret.</p>
          )}
          <button
            className="text-link mt-4"
            onClick={() => setCredential(null)}
          >
            Dismiss
          </button>
        </div>
      )}
      <details>
        <summary className="cursor-pointer py-4 text-sm">
          Register an application
        </summary>
        <form
          className="editor-fields max-w-2xl"
          onSubmit={(e) => {
            e.preventDefault();
            const form = e.currentTarget,
              fd = new FormData(form);
            void action(async () => {
              const result = await mutate<{ id: string; secret: string }>(
                "/v1/admin/applications",
                {
                  name: fd.get("name"),
                  public: fd.get("kind") === "public",
                  redirectUris: String(fd.get("redirectUris"))
                    .split("\n")
                    .map((v) => v.trim())
                    .filter(Boolean),
                  origins: String(fd.get("origins"))
                    .split("\n")
                    .map((v) => v.trim())
                    .filter(Boolean),
                  scopes: [
                    "openid",
                    "profile",
                    "email",
                    ...(fd.has("offline") ? ["offline_access"] : []),
                  ],
                },
              );
              setCredential(result);
              form.reset();
            });
          }}
        >
          <label>
            Name
            <input name="name" required maxLength={100} />
          </label>
          <label>
            Client type
            <select name="kind">
              <option value="public">Browser / native · no secret</option>
              <option value="confidential">Web server · secret required</option>
            </select>
          </label>
          <label>
            Callback URLs · one per line
            <textarea name="redirectUris" required rows={3} />
          </label>
          <label>
            Allowed browser origins · one per line
            <textarea name="origins" rows={2} />
          </label>
          <label className="check-label">
            <input type="checkbox" name="offline" />
            Allow refresh tokens
          </label>
          <button className="button justify-self-start" disabled={busy}>
            Register application
          </button>
        </form>
      </details>
      {apps.map((a) => (
        <article className="admin-record" key={a.id}>
          <h3>
            {a.name} ·{" "}
            {a.disabled
              ? "Disabled"
              : a.client.public
                ? "Public client"
                : "Confidential client"}
          </h3>
          <p className="break-all text-muted">
            {a.id}
            <br />
            {a.client.redirect_uris.join("\n")}
          </p>
          <div className="flex flex-wrap gap-5">
            <button
              className="text-link"
              disabled={busy}
              onClick={() =>
                action(async () => {
                  await mutate(
                    `/v1/admin/applications/${a.id}`,
                    { action: a.disabled ? "enable" : "disable" },
                    "PATCH",
                  );
                })
              }
            >
              {a.disabled ? "Enable" : "Disable & revoke tokens"}
            </button>
            {!a.client.public && (
              <button
                className="text-link"
                disabled={busy}
                onClick={() =>
                  action(async () => {
                    const result = await mutate<{ secret: string }>(
                      `/v1/admin/applications/${a.id}`,
                      { action: "rotate" },
                      "PATCH",
                    );
                    setCredential({ id: a.id, secret: result.secret });
                  })
                }
              >
                Rotate secret & revoke tokens
              </button>
            )}
          </div>
        </article>
      ))}
    </div>
  );
}
