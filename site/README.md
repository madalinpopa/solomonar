# Solomonar website

A static, Markdown-based documentation site for **https://solomonar.dev**. Eleventy renders the pages; Nginx serves the production files in Docker. The browser gets no client framework, just a small script for themes, the illustrated story, navigation, and code copying.

## Local development

Use Node.js 24 or later. From `site/`:

```sh
npm ci --ignore-scripts --no-audit --no-fund
npm run dev
```

Open <http://localhost:4173/>. The preview rebuilds and refreshes on source changes.

```sh
npm run build
npm run check
```

Output is `_site/`, ignored alongside `node_modules/` and `.cache/`. Checks cover local links and fragments, assets, metadata, unique IDs, heading structure, JavaScript syntax, and an 8 KB browser-script budget. Visual and interaction checks are recorded in `design-qa.md`.

## Docker

From the repository root:

```sh
docker build -f site/Dockerfile -t solomonar-site:local .
docker run --rm --read-only --tmpfs /tmp:rw,size=16m,mode=1777 -p 127.0.0.1:4180:8080 solomonar-site:local
```

Open <http://localhost:4180/>. Nginx runs as its unprivileged `nginx` user on port 8080, with writable temporary files only under `/tmp`. `/healthz` returns 200. Missing pages return the site's custom 404 with an actual 404 status. CSS and JavaScript are revalidated so deployments do not retain stale unversioned assets.

The multi-stage Dockerfile pins the official Node and Nginx images by digest, installs the locked build dependencies, runs the static checks, and copies only the generated site into the serving image. The Dockerfile-specific ignore file limits build context to the site and its supplied logo/diagram sources.

## Dokploy Compose

1. Create a **Docker Compose** service from this repository and set its Compose path to `site/compose.yml`.
2. Use `ghcr.io/madalinpopa/solomonar-site:latest`, the Compose default. Optionally set `SITE_IMAGE` to an immutable published `sha-<full commit>` tag for a controlled rollout.
3. If the GHCR package is private, configure registry pull credentials in Dokploy; alternatively make the website package public in GitHub. Keep credentials outside the Compose file.
4. Add `solomonar.dev` in Dokploy's **Domains** tab, select service `site`, container port **8080**, and enable HTTPS. Point the domain's DNS to the Dokploy server.
5. Deploy. Dokploy supplies the routing labels and network configuration. The Compose service exposes port 8080 internally and does not bind a host port.

This uses [Dokploy's native domain management](https://docs.dokploy.com/docs/core/docker-compose/domains), without hardcoded Traefik router names or a `container_name`.

## Container CI

[ci-site.yml](../.github/workflows/ci-site.yml) builds pull requests without publishing. For site changes on `main`, or a manual run, it:

1. Builds the Docker image; the build validates the static output.
2. Runs the Nginx configuration check and HTTP checks against the image, including a real 404.
3. Logs in to GHCR with the workflow's `GITHUB_TOKEN` and publishes the tested image as `latest` and `sha-<full commit>`.

The workflow has `packages: write`; no registry password secret is needed for publishing. Dokploy deployment remains a separate action. The workflow targets the runner's Linux amd64 platform. The Dockerfile also builds natively on arm64 for local testing.

## Content and sources

- `src/index.njk`: the compact illustrated story in four chapters.
- `src/docs/*.md`: tutorial, how-to, reference, and explanation pages.
- `src/_includes/`: shared layouts; `src/_data/site.js`: shared navigation and metadata.
- `src/assets/site.css` and `site.js`: themes, animation, responsive layout, and progressive enhancement.
- `eleventy.config.js`: rendering, heading anchors, collections, logo reuse, and diagram embedding.

`assets/framework/` owns framework behavior. `docs/cli/interface.md` owns the CLI contract. `internal/cli/run.go` establishes command availability. Update the site availability tables with implementation changes. The root README and historical notes can lag behind the code.

All five README diagrams are embedded directly from `docs/images/`, preserving their geometry and labels. The site replaces their embedded palette/font styles with scoped theme tokens; manual light/dark selection updates the diagrams immediately. Original source files are not rewritten. Diagrams scroll horizontally at small widths so their labels remain readable.

## URLs

`SITE_ORIGIN` defaults to `https://solomonar.dev`; `SITE_BASE_PATH` defaults to `/`. Both are optional Docker build arguments. They determine canonical URLs, internal paths, and the sitemap. The domain root is the production target. Subpath generation remains available:

```sh
SITE_BASE_PATH=/solomonar/ npm run build
SITE_BASE_PATH=/solomonar/ npm run check
```

A container built with a subpath needs a matching proxy mount; the supplied Nginx configuration serves the domain root.

## Design and asset provenance

The visual direction retains the selected Stormkeeper palette and editorial typography. The updated landing page tells the requested book → spells → dragon → direction story, with automatic playback, chapter buttons, and pause/play. Manual chapter selection pauses playback; reduced-motion preferences start the story paused and disable animation. Theme choice is retained when local storage is available. Content and navigation remain usable without JavaScript.

- The supplied dragon and favicon come from `docs/logo/`.
- Newsreader is self-hosted under SIL OFL; see `src/assets/fonts/OFL.txt` and [the source](https://github.com/productiontype/Newsreader).
- Tabler Icons v3.35.0 retain their original paths and MIT license in `src/assets/icons-LICENSE.txt`.
- The embroidery band, stitch field, reading scholar, and dragon rider were generated with the built-in image-generation tool for this site, then resized and compressed to WebP. They are contemporary illustrations, not reproductions of a named regional pattern.
- Cultural context is linked in `src/docs/story.md`: [Solomonari](https://en.wikipedia.org/wiki/Solomonari) and [UNESCO's altiță entry](https://ich.unesco.org/en/RL/the-art-of-the-traditional-blouse-with-embroidery-on-the-shoulder-alti-an-element-of-cultural-identity-in-romania-and-the-republic-of-moldova-01861).
