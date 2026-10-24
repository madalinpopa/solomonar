import { readFileSync } from "node:fs";

const base = process.env.SITE_BASE_PATH || "/";
const segments = base.split("/").filter(Boolean);
const pathPrefix = segments.length ? `/${segments.join("/")}/` : "/";
const origin = new URL(process.env.SITE_ORIGIN || "https://solomonar.dev").origin;

export default function (config) {
  config.setUseTemplateCache(false);
  config.addPassthroughCopy({ "src/assets": "assets", "../docs/logo/favicon.svg": "favicon.svg" });
  config.addWatchTarget("../docs/logo/mark.svg");
  config.addWatchTarget("../docs/images");
  config.addGlobalData("origin", origin);
  config.addGlobalData("basePath", pathPrefix);
  config.addGlobalData("canonicalBase", `${origin}${pathPrefix}`);
  config.addFilter("absolute", (url) => `${origin}${pathPrefix}${url.replace(/^\//, "")}`);
  config.addFilter("jsonLd", (value) => JSON.stringify(value).replace(/</g, "\\u003c"));
  config.addFilter("source", (path) => `https://github.com/madalinpopa/solomonar/blob/main/${path}`);
  config.addFilter("selectGroup", (items, group) => items.filter((item) => item.data.group === group));
  config.addFilter("toc", (html) => [...html.matchAll(/<h2 id="([^"]+)">([\s\S]*?)<\/h2>/g)].map((match) => ({ id: match[1], title: match[2].replace(/<[^>]*>/g, "") })));
  config.amendLibrary("md", (md) => {
    md.renderer.rules.table_open = () => '<div class="table-scroll" tabindex="0" role="region" aria-label="Scrollable reference table"><table>\n';
    md.renderer.rules.table_close = () => '</table></div>\n';
    md.renderer.rules.heading_open = (tokens, index, options, env, renderer) => {
      const text = tokens[index + 1].content;
      const slug = text.normalize("NFD").replace(/[\u0300-\u036f]/g, "").toLowerCase().replace(/[^a-z0-9\s-]/g, "").trim().replace(/\s+/g, "-");
      env.headingIds ||= {};
      const count = env.headingIds[slug] || 0;
      env.headingIds[slug] = count + 1;
      tokens[index].attrSet("id", count ? `${slug}-${count + 1}` : slug);
      return renderer.renderToken(tokens, index, options);
    };
  });
  config.addCollection("docs", (api) => api.getFilteredByTag("docs").sort((a, b) => a.data.order - b.data.order));
  config.addShortcode("dragon", (className = "", decorative = true) => {
    let svg = readFileSync(new URL("../docs/logo/mark.svg", import.meta.url), "utf8");
    svg = svg.replace(/<style>[\s\S]*?<\/style>/g, "")
      .replace(/<title>[\s\S]*?<\/title>/, "")
      .replace(/role="img" aria-label="[^"]*"/, decorative ? 'aria-hidden="true"' : 'role="img" aria-label="Solomonar: a cross-stitched balaur"')
      .replace('<svg ', `<svg class="dragon-mark ${className}" focusable="false" `);
    return svg;
  });
  const inlineCode = (text) => text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/`([^`]+)`/g, "<code>$1</code>");
  config.addShortcode("cliUsage", (when, who) => {
    const item = (icon, label, text) => `<div class="cli-usage-item"><svg class="cli-usage-icon" aria-hidden="true"><use href="${pathPrefix}assets/icons.svg#${icon}"></use></svg><div><p class="cli-usage-label">${label}</p><p>${inlineCode(text)}</p></div></div>`;
    return `<div class="cli-usage">${item("clock", "When to use it", when)}${item("user", "Who runs it", who)}</div>`;
  });
  const diagrams = {
    blocks: "framework-blocks", cycle: "workflow-cycle", "task-loop": "workflow-task-loop",
    parallel: "workflow-parallel", "status-stage": "workflow-status-stage", closure: "workflow-closure",
    registry: "workflow-task-registry", "session-stop": "workflow-session-stop", takeover: "workflow-session-takeover",
    "extension-lifecycle": "extension-lifecycle",
  };
  config.addShortcode("diagram", (name) => {
    if (!Object.hasOwn(diagrams, name)) throw new Error(`Unknown framework diagram: ${name}`);
    const source = `docs/images/${diagrams[name]}.svg`;
    const svg = readFileSync(new URL(`../${source}`, import.meta.url), "utf8")
      .replace(/<style>[\s\S]*?<\/style>/g, "")
      .replace('class="dg"', 'class="workflow-diagram"')
      .replace(/>\s+</g, "><").trim();
    return `<figure class="framework-figure"><div class="diagram-scroll" tabindex="0" role="region" aria-label="Scrollable framework diagram">${svg}</div><figcaption>Scroll the diagram on smaller screens. Source: <a href="https://github.com/madalinpopa/solomonar/blob/main/${source}">${source}</a></figcaption></figure>`;
  });
  return {
    dir: { input: "src", output: "_site", includes: "_includes", data: "_data" },
    templateFormats: ["md", "njk"],
    markdownTemplateEngine: "njk",
    htmlTemplateEngine: "njk",
    pathPrefix,
  };
}
