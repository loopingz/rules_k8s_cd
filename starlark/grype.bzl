load("//starlark:utils.bzl", "download_binary")

# version=https://dl.k8s.io/release/stable.txt
# https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl.sha256

_binaries = {
    "darwin-amd64": ("https://github.com/anchore/grype/releases/download/v0.61.1/grype_0.61.1_darwin_amd64.tar.gz", "ced8fe972cf690cc295e1c1ef334a3b27c85ff27875d4afc214824ce664c8472"),
    "darwin-arm64": ("https://github.com/anchore/grype/releases/download/v0.61.1/grype_0.61.1_darwin_arm64.tar.gz", "6a72f55f3106c9498ec5f5f967c71da754951b61a3d6c9122e08652ec80e5e66"),
    "linux-amd64": ("https://github.com/anchore/grype/releases/download/v0.61.1/grype_0.61.1_linux_amd64.tar.gz", "b5628b37123ae03b85bb6e692f37ba015c60ba92f4ef6a2a874fd015f918a6ef"),
    "linux-arm64": ("https://github.com/anchore/grype/releases/download/v0.61.1/grype_0.61.1_linux_arm64.tar.gz", "b77f592386f9dd48d23b05b7bdb52d92e1c9f1b98403c3b54f7e1280b5931ebd"),
}

def grype_setup(name = "grype_bin", binaries = _binaries, bin = ""):
    if (bin == ""):
        bin = name.replace("_bin", "")
    download_binary(name = name, binaries = binaries, bin = bin)

