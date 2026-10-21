import { readFile, readdir, stat } from "node:fs/promises";
import { resolve, relative, dirname, join } from "node:path";
import { spawnSync } from "node:child_process";

const root = resolve(process.argv[2] || "_site");
const prefix = `/${(process.env.SITE_BASE_PATH || "").split("/").filter(Boolean).join("/")}`.replace(/\/$/, "");
const origin = new URL(process.env.SITE_ORIGIN || "https://solomonar.dev").origin;
const baseUrl = `${origin}${prefix}/`;
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
const titles = new Set();
const descriptions = new Set();
const canonicalUrls = new Set();
for (const file of htmlFiles) {
  const html = await text(file);
  const meta = (name) => html.match(new RegExp(`<meta (?:name|property)="${name}" content="([^"]+)">`))?.[1];
  const title = html.match(/<title>([^<]+)<\/title>/)?.[1];
  const description = meta("description");
  const canonical = html.match(/<link rel="canonical" href="([^"]+)">/)?.[1];
  const is404 = file === join(root, "404.html");
  if ((html.match(/<h1[ >]/g) || []).length !== 1) fail(file, "expected exactly one h1");
  if (!html.includes('<main id="main"')) fail(file, "missing main landmark and skip-link target");
  if (!/<title>[^<]+<\/title>/.test(html)) fail(file, "missing title");
  if (!/<meta name="description" content="[^"]+">/.test(html)) fail(file, "missing description");
  if (title && titles.has(title)) fail(file, "duplicate page title");
  if (description && descriptions.has(description)) fail(file, "duplicate meta description");
  titles.add(title);
  descriptions.add(description);
  if (is404) {
    if (meta("robots") !== "noindex, follow") fail(file, "404 must be noindex, follow");
    if (canonical) fail(file, "404 must not declare a canonical URL");
  } else {
    const expectedUrl = `${baseUrl}${relative(root, file).replace(/index\.html$/, "")}`;
    if (canonical !== expectedUrl || !canonical?.startsWith("https://")) fail(file, "incorrect HTTPS canonical URL");
    if ((html.match(/rel="canonical"/g) || []).length !== 1) fail(file, "expected one canonical URL");
    canonicalUrls.add(canonical);
    if (/noindex|nofollow|none/i.test(meta("robots") || "")) fail(file, "content page must be indexable");
    for (const name of ["og:type", "og:site_name", "og:locale", "og:title", "og:description", "og:url", "og:image", "og:image:type", "og:image:width", "og:image:height", "og:image:alt", "twitter:card", "twitter:title", "twitter:description", "twitter:image", "twitter:image:alt"]) {
      if (!meta(name)) fail(file, `missing social metadata: ${name}`);
    }
    if (meta("og:url") !== canonical) fail(file, "Open Graph URL differs from canonical");
    if (meta("og:title") !== title || meta("twitter:title") !== title) fail(file, "social titles differ from page title");
    if (meta("og:description") !== description || meta("twitter:description") !== description) fail(file, "social descriptions differ from page description");
    if (meta("twitter:card") !== "summary_large_image") fail(file, "expected large-image X card");
    if (meta("og:image") !== `${baseUrl}assets/social-card.png` || meta("twitter:image") !== meta("og:image")) fail(file, "incorrect absolute social image URL");
    if (meta("og:image:type") !== "image/png" || meta("og:image:width") !== "1200" || meta("og:image:height") !== "630") fail(file, "incorrect social image format or dimensions");
  }
  let previousHeading = 0;
  for (const heading of html.matchAll(/<h([1-6])\b/g)) {
    const level = Number(heading[1]);
    if (level > previousHeading + 1) fail(file, `heading skips from h${previousHeading} to h${level}`);
    previousHeading = level;
  }
  for (const match of html.matchAll(/<script type="application\/ld\+json">([\s\S]*?)<\/script>/g)) {
    try { JSON.parse(match[1]); } catch { fail(file, "invalid JSON-LD"); }
  }
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
const sitemap = await readFile(join(root, "sitemap.xml"), "utf8");
const sitemapUrls = [...sitemap.matchAll(/<loc>([^<]+)<\/loc>/g)].map((match) => match[1]);
if (new Set(sitemapUrls).size !== sitemapUrls.length) failures.push("sitemap contains duplicate URLs");
for (const url of canonicalUrls) {
  if (!sitemapUrls.includes(url)) failures.push(`sitemap missing canonical page: ${url}`);
}
for (const url of sitemapUrls) {
  if (!canonicalUrls.has(url)) failures.push(`sitemap contains noncanonical or unindexable URL: ${url}`);
}
const robots = await readFile(join(root, "robots.txt"), "utf8");
if (!robots.includes(`Sitemap: ${baseUrl}sitemap.xml`)) failures.push("robots.txt missing absolute sitemap URL");
if (!/^User-agent: \*\s+Allow: \/\s*$/m.test(robots) || /^Disallow:\s*\/\s*$/m.test(robots)) failures.push("robots.txt must allow crawling");
const home = await text(join(root, "index.html"));
const project = JSON.parse(home.match(/<script type="application\/ld\+json">([\s\S]*?)<\/script>/)?.[1] || "null");
if (project?.["@type"] !== "SoftwareSourceCode" || project?.url !== baseUrl || !project?.codeRepository || !project?.license) failures.push("homepage missing project structured data");
const socialImage = await readFile(join(root, "assets/social-card.png"));
if (socialImage.subarray(0, 8).toString("hex") !== "89504e470d0a1a0a" || socialImage.readUInt32BE(16) !== 1200 || socialImage.readUInt32BE(20) !== 630) failures.push("social image must be a 1200 by 630 PNG");
if (socialImage.length > 300_000) failures.push("social image exceeds 300 KB budget");
if (!files.includes(join(root, ".nojekyll"))) failures.push("missing GitHub Pages .nojekyll marker");
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
} else console.log(`Checked ${htmlFiles.length} HTML pages: links, fragments, assets, unique metadata, social cards, canonicals, sitemap, robots, 404 indexability, JSON-LD, heading order, unique IDs, and JavaScript. ${Math.round(totalBytes / 1024)} KB total static output.`);
