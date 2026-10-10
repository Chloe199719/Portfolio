import { defineConfig, globalIgnores } from "eslint/config";
import next from "eslint-config-next/core-web-vitals";
import ts from "eslint-config-next/typescript";
export default defineConfig([
  ...next,
  ...ts,
  globalIgnores([
    ".next/**",
    "backend/web/**",
    "examples/**",
    "lib/generated/**",
    "chloe-portfolio/**",
    "next-env.d.ts",
    ".backups/**",
    "playwright-report/**",
    "test-results/**",
  ]),
  {
    rules: {
      "react-hooks/set-state-in-effect": "off",
      "react-hooks/purity": "off",
    },
  },
]);
