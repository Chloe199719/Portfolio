import { readFile } from "node:fs/promises";

const { VERCEL_TOKEN, VERCEL_ORG_ID, NEXT_PUBLIC_SITE_URL } = process.env;
if (!VERCEL_TOKEN || !VERCEL_ORG_ID || !NEXT_PUBLIC_SITE_URL) {
  throw new Error("Missing Vercel token, team ID, or staging URL.");
}
const result = JSON.parse(await readFile(process.argv[2], "utf8"));
const deployment = result.deployment;
if (
  result.status !== "ok" ||
  deployment?.readyState !== "READY" ||
  !/^dpl_[A-Za-z0-9]+$/.test(deployment?.id ?? "")
) {
  throw new Error("Vercel did not return a ready deployment.");
}
const site = new URL(NEXT_PUBLIC_SITE_URL);
if (site.protocol !== "https:" || site.username || site.password || site.port) {
  throw new Error("Staging requires a plain HTTPS hostname.");
}
// Project-scoped tokens cannot use the CLI alias command's account lookup.
// This documented endpoint authorizes the deployment itself.
const endpoint = new URL(
  `https://api.vercel.com/v2/deployments/${deployment.id}/aliases`,
);
endpoint.searchParams.set("teamId", VERCEL_ORG_ID);
const response = await fetch(endpoint, {
  method: "POST",
  headers: {
    Authorization: `Bearer ${VERCEL_TOKEN}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ alias: site.hostname }),
  signal: AbortSignal.timeout(30_000),
});
const alias = await response.json();
if (!response.ok || alias.alias !== site.hostname || !alias.uid) {
  throw new Error(`Staging alias was not confirmed (HTTP ${response.status}).`);
}
console.log(`Staging alias confirmed: ${site.hostname}`);
