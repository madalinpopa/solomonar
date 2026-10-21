# Site design and delivery QA

Date: 2026-10-21. Target: `https://solomonar.dev/`.

final result: passed

## Design reference and approved changes

The selected reference is **Stormkeeper, option 1**: charcoal, linen, coral thread, editorial serif typography, the supplied stitched dragon, and Romanian embroidery. The implementation uses Eleventy, Markdown, Nunjucks, CSS, and a small browser script. This keeps documentation static and avoids a client framework.

The later requests deliberately change the reference's content and deployment: a shorter illustrated book → spells → dragon → direction story replaces the text-heavy landing page; Docker and Dokploy Compose replace GitHub Pages. “Every dragon needs a little direction” is the headline. These changes are part of the accepted direction.

The supplied logo remains authoritative. Generated illustrations are contemporary embroidery-inspired artwork, not claims to reproduce a specific regional pattern.

## Visual evidence

Local screenshots are under the ignored `.cache/qa/` directory. They are review artifacts, not build inputs.

| Evidence | State |
| --- | --- |
| `desktop-dark.png`, `desktop-light.png` | Homepage, book chapter, 1440 × 1120 CSS pixels, matching scroll position and 1× capture scale. |
| `story-dragon-dark.png` | Homepage, dragon chapter paused, 1440 × 1120, header visible. |
| `mobile-light.png` | Homepage, book chapter, 390 × 844. |
| `diagram-dark.png` | Fully rendered workflow diagram, dark theme, 1440 × 1120. |
| `comparison-full.png` | Selected reference and rendered dark homepage side by side, normalized to matching 1440 × 1120 viewports. |
| `comparison-header.png` | Corresponding header crops compared in one image. |

Both combined comparisons were visually inspected. Layout hierarchy, typography, palette, navigation, embroidery, and controls were reviewed together. The story illustrations and shorter copy follow the subsequent user request. No blocking visual findings remain.

## Findings resolved

- Adjusted headline sizing and spacing so text and artwork remain separate across widths.
- Removed decorative overflow and fixed the story layout at narrow widths.
- Fixed Markdown inserting HTML paragraphs into inline SVGs. SVG whitespace is normalized before embedding; the static checker now rejects interrupted SVG markup. All five diagrams were visually inspected and their expected shapes/text confirmed in the browser.
- Scoped diagram styles to shared theme tokens while preserving source geometry and labels. The source SVGs in `docs/images/` are unchanged.
- Restarted the preview after configuration changes and disabled template caching for local authoring.

## Browser and accessibility checks

- Homepage and framework page tested at widths 320, 390, 600, 768, 1024, 1280, and 1440: no document-level horizontal overflow. Large diagrams and tables have their own keyboard-focusable scrolling regions.
- Light/dark controls update the page and inline diagrams; the selection survives reload and navigation.
- All four story chapter controls select one visible caption. Manual selection pauses playback. Automatic advancement was observed, and pause/play controls work.
- Mobile navigation opens with Enter; Escape closes it and returns focus to the menu button.
- The skip link moves focus to the main content. Code-copy feedback announces success.
- Console warnings and errors: none observed during the checks.
- Normal text contrast against the theme background is at least 5.37:1 for the checked ink, muted, quiet, and accent tokens. Semantic headings, landmarks, button names, focus states, and image alternatives were inspected.

Reduced-motion rules disable animation and initialize playback as paused. The static baseline keeps content and navigation available without JavaScript. Both paths were inspected in source; this browser environment did not provide OS reduced-motion or JavaScript-disable emulation. These checks are not a formal WCAG certification or a measured Core Web Vitals report.

## Build and deployment checks

- Locked dependency installation and root production build pass. The static checker validates 10 HTML pages, local links/fragments, assets, metadata, IDs, embedded SVGs, and browser-script syntax/budget. Total static output is approximately 1070 KB; browser JavaScript stays below 8 KB.
- A separate `/solomonar/` build and link check pass. Production canonical URLs, robots, and sitemap use `https://solomonar.dev` at the domain root.
- The final Docker image builds successfully on local Linux arm64. Its built output passes the same static checks.
- Nginx configuration passes. The final container runs as `nginx`, with a read-only root filesystem, dropped capabilities, and temporary writes under `/tmp`; its health check is healthy.
- `/healthz`, `/`, and `/docs/framework/` return 200. An unknown URL returns the custom page with HTTP 404. Canonical domain metadata, CSS gzip, and `no-cache` revalidation were checked over HTTP.
- Compose configuration validates. CI builds and checks the image before publishing SHA and `latest` tags, and skips publishing for pull requests.

GHCR publication, Linux amd64 execution on GitHub's runner, DNS, HTTPS, and Dokploy deployment require the corresponding remote workflow and server; they were configured and documented, not executed locally. Local previews remain available on ports 4173 (authoring) and 4180 (Nginx container).
