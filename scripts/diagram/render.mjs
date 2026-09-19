#!/usr/bin/env node
// Optional Archify-first generate. The canonical command is:
//   go generate ./internal/diagram
// This script only exists so operators can call Node directly.

import { spawnSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const result = spawnSync("go", ["generate", "./internal/diagram"], {
  cwd: root,
  stdio: "inherit",
  env: process.env,
});
process.exit(result.status ?? 1);
