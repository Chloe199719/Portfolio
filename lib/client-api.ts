"use client";
import { useCallback, useEffect, useState } from "react";
import type { SessionUser } from "./types";
import { apiClient } from "./generated-client";
import { apiBase, readOnly } from "./api-config";
export class APIError extends Error {
  constructor(
    message: string,
    public status: number,
  ) {
    super(message);
  }
}
export async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  if (!apiBase)
    throw new APIError("The backend is not connected in this preview.", 503);
  if (!path.startsWith("/v1/")) throw new Error("Invalid API path.");
  const response = await fetch(`${apiBase}${path}`, {
    ...options,
    credentials: "include",
    cache: "no-store",
  });
  let data: unknown;
  try {
    data = await response.json();
  } catch {
    throw new APIError(
      "The server returned an invalid response. Your changes have not been confirmed.",
      response.status,
    );
  }
  if (!data || typeof data !== "object" || Array.isArray(data))
    throw new APIError(
      "The server returned an invalid response.",
      response.status,
    );
  if (!response.ok)
    throw new APIError(
      "error" in data && typeof data.error === "string"
        ? data.error
        : "The request failed. Please try again.",
      response.status,
    );
  if ("document" in data && data.document !== null) {
    const doc = data.document;
    if (
      !doc ||
      typeof doc !== "object" ||
      !("_id" in doc) ||
      typeof doc._id !== "string" ||
      !("_rev" in doc) ||
      typeof doc._rev !== "string" ||
      !("_type" in doc) ||
      typeof doc._type !== "string"
    )
      throw new APIError(
        "The server returned an invalid document. Your changes have not been confirmed.",
        response.status,
      );
  }
  const method = options.method || "GET";
  if (
    method !== "GET" &&
    !["document", "image", "id", "schedules"].some((key) => key in data) &&
    !("ok" in data && data.ok === true)
  )
    throw new APIError(
      "The server returned an invalid response. Your changes have not been confirmed.",
      response.status,
    );
  return data as T;
}
export async function mutate<T>(
  path: string,
  body: unknown,
  method = "POST",
): Promise<T> {
  if (readOnly)
    throw new APIError(
      "This preview is read-only. Use the configured staging site to make changes.",
      403,
    );
  const result = await apiClient.GET("/v1/session");
  if (!result.response.ok || !result.data)
    throw new APIError(
      result.error?.error || "Could not verify your form session.",
      result.response.status,
    );
  const session = result.data;
  if (typeof session.csrf !== "string" || session.csrf.length < 32)
    throw new APIError("Could not verify your form session.", 403);
  const form = body instanceof FormData;
  return request<T>(path, {
    method,
    headers: {
      "X-CSRF-Token": session.csrf,
      ...(!form ? { "Content-Type": "application/json" } : {}),
    },
    body: body === undefined ? undefined : form ? body : JSON.stringify(body),
  });
}
export function useSession() {
  const [user, setUser] = useState<SessionUser | null>(null);
  const [loading, setLoading] = useState(true);
  const [configured, setConfigured] = useState(false);
  const [error, setError] = useState("");
  const refresh = useCallback(async () => {
    if (!apiBase || readOnly) {
      setConfigured(false);
      setLoading(false);
      return;
    }
    try {
      const data = await request<{
        user: SessionUser | null;
        configured: boolean;
      }>("/v1/session");
      setUser(data.user);
      setConfigured(data.configured);
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not check sign-in.");
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => {
    void refresh();
  }, [refresh]);
  return { user, loading, configured, error, refresh };
}
