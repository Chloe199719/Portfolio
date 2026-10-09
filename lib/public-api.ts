import "server-only";
import { apiBase } from "./api-config";
import snapshot from "./content-snapshot.json";
// Explicit build-only previews can use the versioned public export. Configured APIs fail closed.
export async function publicAPI<T>(path: string): Promise<T> {
  if (!apiBase) {
    if (path === "/v1/content") return { documents: snapshot } as T;
    if (path === "/v1/guestbook") return { entries: [] } as T;
    throw new Error("The backend URL is not configured.");
  }
  const response = await fetch(`${apiBase}${path}`, {
    cache: "no-store",
    signal: AbortSignal.timeout(10000),
  });
  if (!response.ok)
    throw new Error("The content server is temporarily unavailable.");
  return response.json() as Promise<T>;
}
