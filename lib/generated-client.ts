import createClient from "openapi-fetch";
import type { paths } from "./generated/api";
import { apiBase } from "./api-config";
export const apiClient = createClient<paths>({
  baseUrl: apiBase,
  credentials: "include",
  cache: "no-store",
});
