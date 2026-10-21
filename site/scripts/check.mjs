import { readFile, readdir, stat } from "node:fs/promises";
import { resolve, relative, dirname, join } from "node:path";
import { spawnSync } from "node:child_process";

const root = resolve(process.argv[2] || "_site");
const prefix = `/${(process.env.SITE_BASE_PATH || "").split("/").filter(Boolean).join("/")}`.replace(/\/$/, "");
const failures = [];
async function walk(dir) {
  const files = await readdir(dir, { withFileTypes: true });
  return (await Promise.all(files.map((file) => file.isDirectory() ? walk(join(dir, file.name)) : join(dir, file.name)))).flat();
}
const files = await walk(root);
const htmlFiles = files.filter((file) => file.endsWith(".html"));
const cache = new Map();
async function text(file) {
  if (!cache.has(file)) cache.set(file, await readFile(file, "utf8"));
  return cache.get(file);
}
const fail = (file, message) => failures.push(`${relative(root, file)}: ${message}`);
for (const file of htmlFiles) {
  const html = await text(file);
  if ((html.match(/<h1[ >]/g) || []).length !== 1) fail(file, "expected exactly one h1");
  if (!html.includes('<main id="main"')) fail(file, "missing main landmark and skip-link target");
  if (!/<title>[^<]+<\/title>/.test(html)) fail(file, "missing title");
  if (!/<meta name="description" content="[^"]+">/.test(html)) fail(file, "missing description");
  if (!/<link rel="canonical" href="https:\/\/[^" ]+">/.test(html)) fail(file, "missing HTTPS canonical URL");
  if (!html.startsWith("<!doctype html>")) fail(file, "missing HTML doctype");
  if (/\{\{|\{%/.test(html)) fail(file, "unrendered template syntax");
  for (const svg of html.matchAll(/<svg\b[\s\S]*?<\/svg>/g)) {
    if (/<(?:p|pre|code)\b/.test(svg[0])) fail(file, "Markdown interrupted an embedded SVG");
  }
  const ids = [...html.matchAll(/\bid="([^"]+)"/g)].map((match) => match[1]);
  if (new Set(ids).size !== ids.length) fail(file, "duplicate element ID");
  for (const match of html.matchAll(/\b(?:href|src)="([^"]+)"/g)) {
    const link = match[1].replaceAll("&amp;", "&");
    if (/^(?:https?:|mailto:|data:)/.test(link)) continue;
    let [path, hash] = link.split("#");
    path = decodeURIComponent(path.split("?")[0]);
    let target;
    if (!path) target = file;
    else if (path.startsWith("/")) {
      if (prefix && !path.startsWith(`${prefix}/`)) { fail(file, `link escapes base path: ${link}`); continue; }
      target = resolve(root, `.${prefix ? path.slice(prefix.length) : path}`);
    } else target = resolve(dirname(file), path);
    if (target !== root && !target.startsWith(`${root}/`)) { fail(file, `link escapes output: ${link}`); continue; }
    try {
      if ((await stat(target)).isDirectory()) target = join(target, "index.html");
      await stat(target);
      if (hash && /\.(?:html|svg)$/.test(target) && !(await text(target)).includes(`id="${decodeURIComponent(hash)}"`)) fail(file, `missing fragment: ${link}`);
    } catch { fail(file, `missing local target: ${link}`); }
  }
}
const css = await readFile(join(root, "assets/site.css"), "utf8");
for (const match of css.matchAll(/url\("([^"]+)"\)/g)) {
  try { await stat(resolve(root, "assets", match[1])); } catch { failures.push(`CSS asset missing: ${match[1]}`); }
}
const script = join(root, "assets/site.js");
if (spawnSync(process.execPath, ["--check", script]).status !== 0) failures.push("browser JavaScript syntax failed");
if ((await stat(script)).size > 8_000) failures.push("browser JavaScript exceeds 8 KB budget");
const totalBytes = (await Promise.all(files.map(async (file) => (await stat(file)).size))).reduce((sum, size) => sum + size, 0);
if (failures.length) {
  console.error(failures.join("\n"));
  process.exitCode = 1;
} else console.log(`Checked ${htmlFiles.length} HTML pages: links, fragments, assets, metadata, unique IDs, and JavaScript. ${Math.round(totalBytes / 1024)} KB total static output.`);
