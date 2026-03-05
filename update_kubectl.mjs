// While waiting for renovate to have custom datasource management
import fs from "fs";

const platforms = [
  "darwin/amd64",
  "linux/amd64",
  "darwin/arm64",
  "linux/arm64",
  "windows/amd64",
];

async function getVersionMap(version) {
  let map = {};
  for (let platform of platforms) {
    const bin = `kubectl${platform.startsWith("windows") ? ".exe" : ""}`;
    let digest = (
      await (
        await fetch(
          `https://dl.k8s.io/release/v${version}/bin/${platform}/${bin}.sha256`
        )
      ).text()
    ).trim();
    map[platform.replace("/", "_")] = [
      `https://dl.k8s.io/release/v${version}/bin/${platform}/${bin}`,
      digest,
    ];
  }
  return map;
}

const FILE = "lib/private/kubectl_toolchain.bzl";

let fileContent = fs.readFileSync(FILE, "utf8").toString();

// Get latest stable version
let stable = (
  await (await fetch("https://dl.k8s.io/release/stable.txt")).text()
)
  .trim()
  .substring(1);

// Extract the _binaries block using brace counting
function extractBinariesBlock(content) {
  const startMatch = /_binaries\s*=\s*\{/.exec(content);
  if (!startMatch) {
    throw new Error("Could not find _binaries in kubectl.bzl");
  }
  const startIdx = startMatch.index + startMatch[0].length;
  let depth = 1;
  let i = startIdx;
  while (i < content.length && depth > 0) {
    if (content[i] === "{") depth++;
    else if (content[i] === "}") depth--;
    i++;
  }
  // Return the full match including `_binaries = { ... }`
  return {
    fullMatch: content.substring(startMatch.index, i),
    inner: content.substring(startIdx, i - 1),
  };
}

// Parse existing versions from the inner content of _binaries
function parseExistingVersions(inner) {
  const info = {};
  // Match version entries like: "1.30.0": { ... }
  const versionRe = /"([^"]+)"\s*:\s*\{/g;
  let match;
  while ((match = versionRe.exec(inner)) !== null) {
    const version = match[1];
    const startIdx = match.index + match[0].length;
    // Find the matching closing brace for this version's dict
    let depth = 1;
    let i = startIdx;
    while (i < inner.length && depth > 0) {
      if (inner[i] === "{") depth++;
      else if (inner[i] === "}") depth--;
      i++;
    }
    const versionInner = inner.substring(startIdx, i - 1);
    // Parse platform entries like: "darwin_amd64": ("url", "sha256"),
    const platformRe = /"([^"]+)"\s*:\s*\(\s*"([^"]+)"\s*,\s*"([^"]+)"\s*\)/g;
    let pmatch;
    info[version] = {};
    while ((pmatch = platformRe.exec(versionInner)) !== null) {
      info[version][pmatch[1]] = [pmatch[2], pmatch[3]];
    }
  }
  return info;
}

// Generate Starlark _binaries block from info object
function generateBinariesBlock(info) {
  // Sort versions in descending order
  const versions = Object.keys(info).sort((a, b) => {
    const pa = a.split(".").map(Number);
    const pb = b.split(".").map(Number);
    for (let i = 0; i < 3; i++) {
      if (pa[i] !== pb[i]) return pb[i] - pa[i];
    }
    return 0;
  });

  let lines = ["_binaries = {"];
  for (const version of versions) {
    lines.push(`    "${version}": {`);
    for (const platform of Object.keys(info[version]).sort()) {
      const [url, sha] = info[version][platform];
      lines.push(`        "${platform}": ("${url}", "${sha}"),`);
    }
    lines.push("    },");
  }
  lines.push("}");
  return lines.join("\n");
}

const { fullMatch, inner } = extractBinariesBlock(fileContent);
let info = parseExistingVersions(inner);

// Read current versions from https://kubernetes.io/releases/
const html = await (await fetch("https://kubernetes.io/releases/")).text();
const re = /CHANGELOG\/CHANGELOG[^>]+>(?<version>\d+\.\d+\.\d+)/g;

let updated = false;
// New stable
if (!info[stable]) {
  console.log("New latest version found:", stable);
  info[stable] = await getVersionMap(stable);
  fileContent = fileContent.replace(
    /DEFAULT_KUBECTL_VERSION = "\d+\.\d+\.\d+"/,
    `DEFAULT_KUBECTL_VERSION = "${stable}"`
  );
  updated = true;
}
for (let match of html.matchAll(re)) {
  let version = match.groups.version;
  if (!info[version]) {
    console.log("New version found:", version);
    info[version] = await getVersionMap(version);
    updated = true;
  }
}

if (updated) {
  const newBlock = generateBinariesBlock(info);
  fileContent = fileContent.replace(fullMatch, newBlock);
  fs.writeFileSync(FILE, fileContent);
  console.log("Updated", FILE);
} else {
  console.log("No new versions found");
}
