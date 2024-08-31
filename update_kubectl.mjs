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
  let map;
  for (let platform of platforms) {
    let digest = (
      await (
        await fetch(
          `https://dl.k8s.io/release/v${version}/bin/${platform}/kubectl.sha256`
        )
      ).text()
    ).trim();
    map[platform.replace("/", "_")] = [
      `https://dl.k8s.io/release/v${version}/bin/${platform}/kubectl${
        platform.startsWith("windows") ? ".exe" : ""
      }`,
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

let current = fileContent.match(/DEFAULT_KUBECTL_VERSION = "\d+\.\d+\.\d+"/)[1];
let content = /_binaries\s*=(?<versions>(\s*{[^{]+{[^}]+})+[^}]+})/gm.exec(
  fileContent
);
if (!content) {
  throw new Error("Could not find _binaries in kubectl.bzl");
}
const json = content.groups.versions
  .replace(/\(/g, "[")
  .replace(/\)/g, "]")
  .replace(/},\s*}/g, "}}")
  .replace(/],\s*}/g, "]}");
console.log(json);

let info = JSON.parse(json);

// Read current versions from https://kubernetes.io/releases/
const html = await (await fetch("https://kubernetes.io/releases/")).text();
html.match(/<a href="\/docs\/reference\/kubectl\/versions.html">/g);
const re = /CHANGELOG\/CHANGELOG[^>]+>(?<version>\d+\.\d+\.\d+)/g;

let updated = false;
// New stable
if (!info[stable]) {
  console.log("New latest version found:", stable);
  info[stable] = await getVersionMap(stable);
  updated = true;
}
for (let match of html.matchAll(re)) {
  let version = match.groups.version;
  if (!info[version]) {
    console.log("New version found:", version);
    info[version] = addVersion(version);
    updated = true;
  }
}

if (updated) {
  const newContent = JSON.stringify(info, null, 2)
    .replace(/\[/g, "(")
    .replace(/\]/g, ")")
    .replace(/\)(\s*)}/, "),$1}")
    .replace(/\}(\s*)}/, "},$1}");

  fileContent.replace(
    /_binaries\s*=(\s*{[^{]+{[^}]+})+[^}]+}/gm,
    `_binaries = ${newContent}`
  );
  fs.writeFileSync(FILE, fileContent);
}
