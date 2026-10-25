import { execFileSync } from "node:child_process";

const tag = /^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/;

/** The latest stable release tag reachable from HEAD, or "dev" when there is none. */
function latestTag() {
  try {
    return execFileSync("git", ["describe", "--tags", "--abbrev=0", "--match", "v[0-9]*.[0-9]*.[0-9]*", "--exclude", "v*-*"], { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim();
  } catch {
    return "dev";
  }
}

export default function () {
  const version = process.env.SITE_VERSION || latestTag();
  if (version !== "dev" && !tag.test(version)) throw new Error(`SITE_VERSION must be vX.Y.Z or dev, got "${version}"`);
  return { version, number: version.replace(/^v/, "") };
}
