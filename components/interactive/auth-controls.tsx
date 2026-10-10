"use client";
import { useState } from "react";
import { ExitIcon, ArrowTopRightIcon } from "@radix-ui/react-icons";
import { mutate } from "@/lib/client-api";
import { apiBase, authBase, readOnly } from "@/lib/api-config";
import type { SessionUser } from "@/lib/types";
export function AuthControls({
  user,
  configured,
  onChange,
}: {
  user: SessionUser | null;
  configured: boolean;
  onChange: () => Promise<void>;
  onComplete?: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  return (
    <div className="auth-controls">
      {user ? (
        <div className="signed-in">
          <span>
            Signed in as <strong>{user.name}</strong>
          </span>
          <a className="text-link" href={`${authBase}/account`}>
            Account <ArrowTopRightIcon />
          </a>
          <button
            className="text-link"
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              setError("");
              try {
                await mutate("/v1/session", undefined, "DELETE");
                await onChange();
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            Sign out of website <ExitIcon />
          </button>
        </div>
      ) : (
        <>
          <button
            className="button"
            disabled={!configured || readOnly}
            onClick={() =>
              window.location.assign(
                new URL(
                  `/v1/session/start?returnTo=${encodeURIComponent(window.location.href)}`,
                  apiBase,
                ).href,
              )
            }
          >
            Sign in with Chloe ID <ArrowTopRightIcon />
          </button>
          <p className="form-help">
            {readOnly
              ? "This preview is read-only."
              : !configured
                ? "Sign-in is currently unavailable. You can still explore the site."
                : "Use your account, Google, GitHub, or a passkey."}
          </p>
        </>
      )}
      {configured && !readOnly && (
        <div className="mt-4">
          <a className="text-link text-xs" href={`${authBase}/logout`}>
            Sign out of Chloe ID <ExitIcon />
          </a>
          <p className="form-help">
            Sign out of Chloe ID and this website in this browser.
          </p>
        </div>
      )}
      {error && (
        <p role="alert" className="form-error">
          {error}
        </p>
      )}
    </div>
  );
}
