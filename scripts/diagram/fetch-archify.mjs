#!/usr/bin/env node
// Downloads the Archify skill zip and extracts it to third_party/archify.

import { execSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const dest = path.join(root, "third_party", "archify");
const url = "https://github.com/tt-a1i/archify/raw/main/archify.zip";
const zip = path.join(root, ".tmp", "archify.zip");

fs.mkdirSync(path.dirname(zip), { recursive: true });
console.log("downloading", url);
execSync(`curl -fsSL "${url}" -o "${zip}"`, { stdio: "inherit" });
fs.mkdirSync(dest, { recursive: true });
const unzip = process.platform === "win32" ? `tar -xf "${zip}" -C "${dest}"` : `unzip -o "${zip}" -d "${dest}"`;
execSync(unzip, { stdio: "inherit" });
const nested = path.join(dest, "archify");
if (fs.existsSync(path.join(nested, "bin", "archify.mjs"))) {
  for (const name of fs.readdirSync(nested)) {
    fs.rmSync(path.join(dest, name), { recursive: true, force: true });
    fs.renameSync(path.join(nested, name), path.join(dest, name));
  }
  fs.rmSync(nested, { recursive: true, force: true });
}
if (!fs.existsSync(path.join(dest, "bin", "archify.mjs"))) {
  console.error("archify.bin missing after extract");
  process.exit(1);
}
console.log("installed", dest);
