// While waiting for renovate to have custom datasource management

import fs from "fs";
let fileContent = fs.readFileSync("starlark/kubectl.bzl", "utf8").toString();

// Get latest stable version
let stable = (
  await (await fetch("https://dl.k8s.io/release/stable.txt")).text()
)
  .trim()
  .substring(1);

let content = fileContent.match(/_binaries\W+=\W+{([^}])+/gm);
if (!content) {
  throw new Error("Could not find _binaries in kubectl.bzl");
}
content = content[0];
let data = {};
const regex =
  /https:\/\/dl.k8s.io\/release\/v(?<currentVersion>[^\/]+)\/bin\/(?<depType>[^\/]+\/[^\/]+)[^,]*,\W"(?<currentDigest>[^"]+)/gm;
let info;
while ((info = regex.exec(content))) {
  console.log(info);
  data[info.groups.currentVersion] ??= {};
  data[info.groups.currentVersion][info.groups.depType] =
    info.groups.currentDigest;
}

console.log("Latest version:", stable);
console.log("Current version:", Object.keys(data).join(","));

if (stable !== Object.keys(data).join(",")) {
  console.log("Updating kubectl binaries");
  // Gather all archs to replace
  let archs = [];
  let archsDigests = {};
  Object.values(data).forEach((v) =>
    Object.keys(v).forEach((k) => {
      archsDigests[k] ??= [];
      archsDigests[k].push(v[k]);
      archs.push(k);
    })
  );
  // Replace version
  fileContent = fileContent.replace(
    /https:\/\/dl\.k8s\.io\/release\/v\d+\.\d+\.\d+/g,
    "https://dl.k8s.io/release/v" + stable
  );
  // Replace now for all archs
  for (let arch of archs) {
    let newDigest = (
      await (
        await fetch(
          `https://dl.k8s.io/release/v${stable}/bin/${arch}/kubectl.sha256`
        )
      ).text()
    ).trim();
    for (let digest of archsDigests[arch]) {
      fileContent = fileContent.replace(new RegExp(digest, "g"), newDigest);
    }
  }
  fs.writeFileSync("starlark/kubectl.bzl", fileContent);
  //console.log("Data", fileContent);
}
