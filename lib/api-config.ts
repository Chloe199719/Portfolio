export const apiBase = (process.env.NEXT_PUBLIC_API_URL || "").replace(
  /\/$/,
  "",
);
export const authBase = (process.env.NEXT_PUBLIC_AUTH_URL || apiBase).replace(
  /\/$/,
  "",
);
export const readOnly = process.env.NEXT_PUBLIC_READ_ONLY === "true";
