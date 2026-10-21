const root = document.documentElement;
const announce = (text) => { document.querySelector("#announcement").textContent = text; };

document.querySelector(".theme-switch").hidden = false;
const themeButtons = document.querySelectorAll("[data-set-theme]");
function setTheme(theme, persist = false) {
  root.dataset.theme = theme;
  themeButtons.forEach((button) => button.setAttribute("aria-pressed", String(button.dataset.setTheme === theme)));
  document.querySelector('meta[name="theme-color"]').content = theme === "light" ? "#F6F1E7" : "#1A1614";
  if (persist) {
    try { localStorage.setItem("solomonar-theme", theme); } catch { /* The switch also works without storage. */ }
  }
}
setTheme(root.dataset.theme);
themeButtons.forEach((button) => button.addEventListener("click", () => setTheme(button.dataset.setTheme, true)));

const menuButton = document.querySelector(".menu-toggle");
const menu = document.querySelector("#primary-nav");
const mobile = matchMedia("(max-width: 900px)");
function closeMenu() {
  menu.dataset.open = "false";
  menuButton.setAttribute("aria-expanded", "false");
}
function syncMenu() {
  menuButton.hidden = !mobile.matches;
  menu.dataset.collapsible = String(mobile.matches);
  closeMenu();
}
syncMenu();
mobile.addEventListener("change", syncMenu);
menuButton.addEventListener("click", () => {
  const open = menu.dataset.open !== "true";
  menu.dataset.open = String(open);
  menuButton.setAttribute("aria-expanded", String(open));
});
menu.addEventListener("click", (event) => { if (event.target.closest("a")) closeMenu(); });
document.addEventListener("keydown", (event) => {
  if (event.key === "Escape" && menu.dataset.open === "true") { closeMenu(); menuButton.focus(); }
});

const story = document.querySelector(".story-hero");
if (story) {
  const reducedMotion = matchMedia("(prefers-reduced-motion: reduce)");
  const motionButton = story.querySelector(".story-motion");
  const buttons = [...story.querySelectorAll("[data-chapter-button]")];
  const captions = [...story.querySelectorAll("[data-chapter-caption]")];
  let chapter = 0;
  let paused = reducedMotion.matches;
  let timer;
  function showChapter(index) {
    chapter = index;
    story.dataset.chapter = String(index);
    buttons.forEach((button, i) => button.setAttribute("aria-pressed", String(i === index)));
    captions.forEach((caption, i) => { caption.hidden = i !== index; });
  }
  function syncPlayback() {
    clearInterval(timer);
    root.dataset.motion = paused ? "paused" : "running";
    motionButton.setAttribute("aria-pressed", String(paused));
    motionButton.textContent = paused ? "Play story" : "Pause story";
    if (!paused && !document.hidden) timer = setInterval(() => showChapter((chapter + 1) % buttons.length), 6500);
  }
  motionButton.hidden = false;
  story.querySelector(".chapter-buttons").hidden = false;
  motionButton.addEventListener("click", () => { paused = !paused; syncPlayback(); });
  buttons.forEach((button, index) => button.addEventListener("click", () => {
    paused = true;
    showChapter(index);
    syncPlayback();
    announce(`Chapter ${index + 1}. ${captions[index].textContent.trim()}`);
  }));
  reducedMotion.addEventListener("change", () => { paused = reducedMotion.matches; syncPlayback(); });
  document.addEventListener("visibilitychange", syncPlayback);
  syncPlayback();
}

// Code examples remain readable when clipboard access or JavaScript is unavailable.
if (navigator.clipboard) document.querySelectorAll("pre:has(code)").forEach((pre) => {
  const button = document.createElement("button");
  button.className = "copy-button";
  button.type = "button";
  button.setAttribute("aria-label", "Copy code example");
  const iconURL = new URL("icons.svg", document.currentScript?.src || [...document.scripts].find((script) => script.src.endsWith("/site.js")).src);
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("aria-hidden", "true");
  const use = document.createElementNS("http://www.w3.org/2000/svg", "use");
  use.setAttribute("href", `${iconURL}#copy`);
  svg.append(use);
  button.append(svg);
  button.addEventListener("click", async () => {
    try {
      await navigator.clipboard.writeText(pre.querySelector("code").textContent.trimEnd());
      use.setAttribute("href", `${iconURL}#check`);
      announce("Code example copied.");
      setTimeout(() => use.setAttribute("href", `${iconURL}#copy`), 1800);
    } catch { announce("Copy unavailable. Select the code to copy it manually."); }
  });
  pre.append(button);
});
