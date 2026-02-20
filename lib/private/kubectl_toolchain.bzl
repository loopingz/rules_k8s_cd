load("@aspect_bazel_lib//lib/private:repo_utils.bzl", "repo_utils")
load("//lib:repo_utils.bzl", "download_toolchain_binary")

# version=https://dl.k8s.io/release/stable.txt
# https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl.sha256
# https://dl.k8s.io/release/${version}/bin/darwin/amd64/kubectl https://dl.k8s.io/release/${version}/bin/darwin/amd64/kubectl.sha256
# https://dl.k8s.io/release/${version}/bin/linux/arm64/kubectl https://dl.k8s.io/release/${version}/bin/linux/arm64/kubectl.sha256
# https://dl.k8s.io/release/${version}/bin/linux/amd64/kubectl https://dl.k8s.io/release/${version}/bin/linux/amd64/kubectl.sha256

_binaries = {
  "1.31.0": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.31.0/bin/darwin/amd64/kubectl",
      "fb6e07a69acc4e16885eda55b524c13b84bfbcf78cfac8d6c378d2bad321e105"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.31.0/bin/linux/amd64/kubectl",
      "7c27adc64a84d1c0cc3dcf7bf4b6e916cc00f3f576a2dbac51b318d926032437"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.31.0/bin/darwin/arm64/kubectl",
      "b7472df17a885574ed7273947a8a274c156357db21b981208e8e109b9ed4022d"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.31.0/bin/linux/arm64/kubectl",
      "f42832db7d77897514639c6df38214a6d8ae1262ee34943364ec1ffaee6c009c"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.31.0/bin/windows/amd64/kubectl.exe",
      "a618de26c86421a394de7041f9d0a87752dd4e555894d2278421cf12097fa531"
    ),
  },
  "1.30.0": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.30.0/bin/darwin/amd64/kubectl",
      "bcfa57d020b8d07d0ea77235ce8012c2c28fefdfd7cb9738f33674a7b16cef08"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.30.0/bin/linux/amd64/kubectl",
      "7c3807c0f5c1b30110a2ff1e55da1d112a6d0096201f1beb81b269f582b5d1c5"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.30.0/bin/darwin/arm64/kubectl",
      "45cfa208151320153742062824398f22bb6bfb5a142bf6238476d55dacbd1bdd"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.30.0/bin/linux/arm64/kubectl",
      "669af0cf520757298ea60a8b6eb6b719ba443a9c7d35f36d3fb2fd7513e8c7d2"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.30.0/bin/windows/amd64/kubectl.exe",
      "e0e72bf37bf563fdea4a6070b07e2fbaa818aa02ed38c5d10d9ce146106cab70"
    )
  },
  "1.35.0": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.35.0/bin/darwin/amd64/kubectl",
      "2447cb78911b10a667202b078eeb30541ec78d1280c3682921dc81607e148d96"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.35.0/bin/linux/amd64/kubectl",
      "a2e984a18a0c063279d692533031c1eff93a262afcc0afdc517375432d060989"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.35.0/bin/darwin/arm64/kubectl",
      "cf699c56340dc775230fde4ef84237d27563ea6ef52164c7d078072b586c3918"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.35.0/bin/linux/arm64/kubectl",
      "58f82f9fe796c375c5c4b8439850b0f3f4d401a52434052f2df46035a8789e25"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.35.0/bin/windows/amd64/kubectl.exe",
      "4c5d14b8673bd55f813a8965ad70d5150e3960ee5f274025e2286aea3a0fa8b6"
    )
  },
  "1.34.0": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.34.0/bin/darwin/amd64/kubectl",
      "a5904061dd5c8e57d55e52c78fa23790e76de30924b26ba31be891e75710d7a9"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.34.0/bin/linux/amd64/kubectl",
      "cfda68cba5848bc3b6c6135ae2f20ba2c78de20059f68789c090166d6abc3e2c"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.34.0/bin/darwin/arm64/kubectl",
      "d491f4c47c34856188d38e87a27866bd94a66a57b8db3093a82ae43baf3bb20d"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.34.0/bin/linux/arm64/kubectl",
      "00b182d103a8a73da7a4d11e7526d0543dcf352f06cc63a1fde25ce9243f49a0"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.34.0/bin/windows/amd64/kubectl.exe",
      "856b6a92556452e249db940e7fdb8d8f8f622805d25f67de09a4d4d2da6f6132"
    )
  },
  "1.34.1": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.34.1/bin/darwin/amd64/kubectl",
      "bb211f2b31f2b3bc60562b44cc1e3b712a16a98e9072968ba255beb04cefcfdf"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.34.1/bin/linux/amd64/kubectl",
      "7721f265e18709862655affba5343e85e1980639395d5754473dafaadcaa69e3"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.34.1/bin/darwin/arm64/kubectl",
      "d80e5fa36f2b14005e5bb35d3a72818acb1aea9a081af05340a000e5fbdb2f76"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.34.1/bin/linux/arm64/kubectl",
      "420e6110e3ba7ee5a3927b5af868d18df17aae36b720529ffa4e9e945aa95450"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.34.1/bin/windows/amd64/kubectl.exe",
      "d118a8ddb0de15ff230189c85f5157e752405eb0ae8fa680d284de094c9a20f0"
    )
  },
  "1.34.2": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.34.2/bin/darwin/amd64/kubectl",
      "d2a71bb7dd7238287f2ba4efefbad4f98584170063f7d9e6c842f772d9255d45"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.34.2/bin/linux/amd64/kubectl",
      "9591f3d75e1581f3f7392e6ad119aab2f28ae7d6c6e083dc5d22469667f27253"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.34.2/bin/darwin/arm64/kubectl",
      "8f38d3a38ae317b00ebf90254dc274dd28d8c6eea4a4b30c5cb12d3d27017b6d"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.34.2/bin/linux/arm64/kubectl",
      "95df604e914941f3172a93fa8feeb1a1a50f4011dfbe0c01e01b660afc8f9b85"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.34.2/bin/windows/amd64/kubectl.exe",
      "7d34dcc49a185d64194ff3e952d5621b7da4f5562fa83df5acf305bd1f7de9cc"
    )
  },
  "1.34.3": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.34.3/bin/darwin/amd64/kubectl",
      "657afbd0e653c4ce3af1b5a645a4eaba282cf8eb2bcda7191ff60866e50e4d7f"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.34.3/bin/linux/amd64/kubectl",
      "ab60ca5f0fd60c1eb81b52909e67060e3ba0bd27e55a8ac147cbc2172ff14212"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.34.3/bin/darwin/arm64/kubectl",
      "e51367d2107d605f4edd7c2fb25897b0c0695a7de1a9f9d04cd6c9356b890b14"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.34.3/bin/linux/arm64/kubectl",
      "46913a7aa0327f6cc2e1cc2775d53c4a2af5e52f7fd8dacbfbfd098e757f19e9"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.34.3/bin/windows/amd64/kubectl.exe",
      "5ef6e0b019cfea5b0eff55b576c0118f64c0758a8bcbf52587c7f454f302f7bc"
    )
  },
  "1.33.1": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.33.1/bin/darwin/amd64/kubectl",
      "8d36a5c66142547ad16e332942fd16a0ca2b3346d9ebaab6c348de2c70d9d875"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.33.1/bin/linux/amd64/kubectl",
      "5de4e9f2266738fd112b721265a0c1cd7f4e5208b670f811861f699474a100a3"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.33.1/bin/darwin/arm64/kubectl",
      "8ae6823839993bb2e394c3cf1919748e530642c625dc9100159595301f53bdeb"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.33.1/bin/linux/arm64/kubectl",
      "d595d1a26b7444e0beb122e25750ee4524e74414bbde070b672b423139295ce6"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.33.1/bin/windows/amd64/kubectl.exe",
      "815c3c39984d1f7347486ad58b8e33e61ee87bc8ad79e0dbc9793e22200614fb"
    )
  },
  "1.33.2": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.33.2/bin/darwin/amd64/kubectl",
      "ff468749bd3b5f4f15ad36f2a437e65fcd3195a2081925140334429eaced1a8a"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.33.2/bin/linux/amd64/kubectl",
      "33d0cdec6967817468f0a4a90f537dfef394dcf815d91966ca651cc118393eea"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.33.2/bin/darwin/arm64/kubectl",
      "8730bf6dab538a1e9710a3668e2cd5f1bdc3c25c68b65a57c5418bdc3472769c"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.33.2/bin/linux/arm64/kubectl",
      "54dc02c8365596eaa2b576fae4e3ac521db9130e26912385e1e431d156f8344d"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.33.2/bin/windows/amd64/kubectl.exe",
      "c45a0fb477262eebd4a4a2936ea6bd10ce6a7db8f1356cff6e703c948538c76b"
    )
  },
  "1.33.3": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.33.3/bin/darwin/amd64/kubectl",
      "9652b55a58e84454196a7b9009f6d990d3961e2bd4bd03f64111d959282b46b1"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.33.3/bin/linux/amd64/kubectl",
      "2fcf65c64f352742dc253a25a7c95617c2aba79843d1b74e585c69fe4884afb0"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.33.3/bin/darwin/arm64/kubectl",
      "3de173356753bacb215e6dc7333f896b7f6ab70479362146c6acca6e608b3f53"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.33.3/bin/linux/arm64/kubectl",
      "3d514dbae5dc8c09f773df0ef0f5d449dfad05b3aca5c96b13565f886df345fd"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.33.3/bin/windows/amd64/kubectl.exe",
      "fbcb21ae1f8e0313ca44c9a3392f62523caf8c1a23b49c80e01cbf541060d592"
    )
  },
  "1.33.4": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.33.4/bin/darwin/amd64/kubectl",
      "4b39b8bb12e78ce801b39c9ec50421e3d6e144d8e3f113cd18e6d61709b8c73b"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.33.4/bin/linux/amd64/kubectl",
      "c2ba72c115d524b72aaee9aab8df8b876e1596889d2f3f27d68405262ce86ca1"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.33.4/bin/darwin/arm64/kubectl",
      "a44662db083fdd1b19ce55ba77eb64d51206310bbae90df90eb5d9e30ea54603"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.33.4/bin/linux/arm64/kubectl",
      "76cd7a2aa59571519b68c3943521404cbce55dafb7d8866f8d0ea2995b396eef"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.33.4/bin/windows/amd64/kubectl.exe",
      "15487c2a017af8ef8a5fbc2390af78b90f98d2909f23eeb684c64a8af3f7c4eb"
    )
  },
  "1.33.5": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.33.5/bin/darwin/amd64/kubectl",
      "ebdefb65c60c920510a605f13622e7eadb85bb83ba393d9eed2389bac30672b1"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.33.5/bin/linux/amd64/kubectl",
      "6a12d6c39e4a611a3687ee24d8c733961bb4bae1ae975f5204400c0a6930c6fc"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.33.5/bin/darwin/arm64/kubectl",
      "22f7256932c1c5205d7323a63d16253b8405ecccfd57c7a2484d3219c6822d3e"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.33.5/bin/linux/arm64/kubectl",
      "6db7c5d846c3b3ddfd39f3137a93fe96af3938860eefdbf2429805ee1656e381"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.33.5/bin/windows/amd64/kubectl.exe",
      "2fa5d21aa99afe994b1e929054d4ca701f7dd5e8124f8f1c83d28186474bc00b"
    )
  },
  "1.33.6": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.33.6/bin/darwin/amd64/kubectl",
      "a0f485c2b8296c84fda606dd585e2458a06a41235f1e96348cf64a2f527f6e77"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.33.6/bin/linux/amd64/kubectl",
      "d25d9b63335c038333bed785e9c6c4b0e41d791a09cac5f3e8df9862c684afbe"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.33.6/bin/darwin/arm64/kubectl",
      "ba6e00a0479d45a4aa59ad550ed0fd68696e73bd2d43d0e00213fba41f61fa54"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.33.6/bin/linux/arm64/kubectl",
      "3ab32d945a67a6000ba332bf16382fc3646271da6b7d751608b320819e5b8f38"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.33.6/bin/windows/amd64/kubectl.exe",
      "bc2e96179cce21fa3ef6e216fe853f41d08850f73f61a150d1485ee18a16acea"
    )
  },
  "1.33.7": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.33.7/bin/darwin/amd64/kubectl",
      "45be3f5293da84d97e86580a541b247fe3cec60196fdd6abd2b811d7dd4d3f1b"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.33.7/bin/linux/amd64/kubectl",
      "471d94e208a89be62eb776700fc8206cbef11116a8de2dc06fc0086b0015375b"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.33.7/bin/darwin/arm64/kubectl",
      "2e333f56d115081af83a48b5f31a91fb32852550f8117a0a31cf8bae2e601704"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.33.7/bin/linux/arm64/kubectl",
      "fa7ee98fdb6fba92ae05b5e0cde0abd5972b2d9a4a084f7052a1fd0dce6bc1de"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.33.7/bin/windows/amd64/kubectl.exe",
      "df8bead144a7a997a79c480083061955ddcd171f803631ac1239c0bd6a8f36ac"
    )
  },
  "1.32.0": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.0/bin/darwin/amd64/kubectl",
      "516585916f499077fac8c2fdd2a382818683f831020277472e6bcf8d1a6f9be4"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.0/bin/linux/amd64/kubectl",
      "646d58f6d98ee670a71d9cdffbf6625aeea2849d567f214bc43a35f8ccb7bf70"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.0/bin/darwin/arm64/kubectl",
      "5bfd5de53a054b4ef614c60748e28bf47441c7ed4db47ec3c19a3e2fa0eb5555"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.0/bin/linux/arm64/kubectl",
      "ba4004f98f3d3a7b7d2954ff0a424caa2c2b06b78c17b1dccf2acc76a311a896"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.0/bin/windows/amd64/kubectl.exe",
      "3601cb47c4d6a42b033a8f8fca68bc6f24baa99f5a1250fdb138d24a6c7cc749"
    )
  },
  "1.32.1": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.1/bin/darwin/amd64/kubectl",
      "8bffe90f5a034d392a0ba6fd7ee16c0d40b1dba1ccc4350821102c5d5c56d846"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.1/bin/linux/amd64/kubectl",
      "e16c80f1a9f94db31063477eb9e61a2e24c1a4eee09ba776b029048f5369db0c"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.1/bin/darwin/arm64/kubectl",
      "5b89f9598e2e7da04cc0b5dd6e8daca01d23855fd00c8ea259fd2aab993114db"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.1/bin/linux/arm64/kubectl",
      "98206fd83a4fd17f013f8c61c33d0ae8ec3a7c53ec59ef3d6a0a9400862dc5b2"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.1/bin/windows/amd64/kubectl.exe",
      "b6378f34dcab2d411fb7a89ba700df0f784c0b063dd02dbb92396a72c4d3104e"
    )
  },
  "1.32.2": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.2/bin/darwin/amd64/kubectl",
      "371b8fbd481e1e9052ace16d9c243e92618a2ea9a18c1aaf235d35fef20c0c32"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.2/bin/linux/amd64/kubectl",
      "4f6a959dcc5b702135f8354cc7109b542a2933c46b808b248a214c1f69f817ea"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.2/bin/darwin/arm64/kubectl",
      "31b6318deaa72014b72121e1c7a2e12496d077cee49bbeda94250aec4c978ffb"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.2/bin/linux/arm64/kubectl",
      "7381bea99c83c264100f324c2ca6e7e13738a73b8928477ac805991440a065cd"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.2/bin/windows/amd64/kubectl.exe",
      "cf51a1c6bf3b6ba6a5b549d1debf8aa6afb00c4c5a3d5d4bb1072f54cbe4390f"
    )
  },
  "1.32.3": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.3/bin/darwin/amd64/kubectl",
      "b814c523071cd09e27c88d8c87c0e9b054ca0cf5c2b93baf3127750a4f194d5b"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.3/bin/linux/amd64/kubectl",
      "ab209d0c5134b61486a0486585604a616a5bb2fc07df46d304b3c95817b2d79f"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.3/bin/darwin/arm64/kubectl",
      "a110af64fc31e2360dd0f18e4110430e6eedda1a64f96e9d89059740a7685bbd"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.3/bin/linux/arm64/kubectl",
      "6c2c91e760efbf3fa111a5f0b99ba8975fb1c58bb3974eca88b6134bcf3717e2"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.3/bin/windows/amd64/kubectl.exe",
      "3fd1576a902ecf713f7d6390ae01799e370883e0341177ee09dbdc362db953e3"
    )
  },
  "1.32.4": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.4/bin/darwin/amd64/kubectl",
      "061f65fe5405538f6fe8edd3c3373f479a1d59944ebf6268905535a617151d16"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.4/bin/linux/amd64/kubectl",
      "10d739e9af8a59c9e7a730a2445916e04bc9cbb44bc79d22ce460cd329fa076c"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.4/bin/darwin/arm64/kubectl",
      "01344900ac3c2c97a3290e9465d36f0dea20ca4533d226dfbe7c9a90e80ff9d4"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.4/bin/linux/arm64/kubectl",
      "c6f96d0468d6976224f5f0d81b65e1a63b47195022646be83e49d38389d572c2"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.4/bin/windows/amd64/kubectl.exe",
      "8e93d01f8efe80db614cf7dc422f9bb3fbad1b16f82d13f0ea70441441e486e4"
    )
  },
  "1.32.5": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.5/bin/darwin/amd64/kubectl",
      "f357d30fc338eb914e6e7a5e0408852d3011fac18d98f4484c4861c4c2cead3c"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.5/bin/linux/amd64/kubectl",
      "aaa7e6ff3bd28c262f2d95c8c967597e097b092e9b79bcb37de699e7488e3e7b"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.5/bin/darwin/arm64/kubectl",
      "b3b08783545e735b030376627133ddf53dc0e2c2ed4c413d87d4bcd7c2b0c632"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.5/bin/linux/arm64/kubectl",
      "9edee84103e63c40a37cd15bd11e04e7835f65cb3ff5a50972058ffc343b4d96"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.5/bin/windows/amd64/kubectl.exe",
      "df01c85015fa2b19fa7f92a7704aae9de5b5dc70fed32a01bd26f57f7ba563a5"
    )
  },
  "1.32.6": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.6/bin/darwin/amd64/kubectl",
      "ad0c1880b1bcd36869d75a54c3401b718c091d75d11d08f57034fb7b4712f6ef"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.6/bin/linux/amd64/kubectl",
      "0e31ebf882578b50e50fe6c43e3a0e3db61f6a41c9cded46485bc74d03d576eb"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.6/bin/darwin/arm64/kubectl",
      "8ac847473a6794dd35d2b980c9249b79dedb6e234d00fd0f223cf6b67be12999"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.6/bin/linux/arm64/kubectl",
      "f7bac84f8c35f55fb2c6ad167beb59eba93de5924b50bbaa482caa14ff480eec"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.6/bin/windows/amd64/kubectl.exe",
      "3b4aabd90c52e01557f08cb2747431a78767cf978646812a69d8d53d73f7049e"
    )
  },
  "1.32.7": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.7/bin/darwin/amd64/kubectl",
      "050a5b4227a07c6d7f5add1863323f9db90b97c12874e2218224c9be74286980"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.7/bin/linux/amd64/kubectl",
      "b8f24d467a8963354b028796a85904824d636132bef00988394cadacffe959c9"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.7/bin/darwin/arm64/kubectl",
      "07a3511f02763076859e37abae33e1513285feec0482798a547441128f84662b"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.7/bin/linux/arm64/kubectl",
      "232f6e517633fbb4696c9eb7a0431ee14b3fccbb47360b4843d451e0d8c9a3a2"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.7/bin/windows/amd64/kubectl.exe",
      "06468f371634191e8a3d7ec63c463dbf81a27518a9f87309153e67d760f94eff"
    )
  },
  "1.32.8": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.8/bin/darwin/amd64/kubectl",
      "a00a8fadd4a7ca520e68e88a640ca60b4601695f68b8dcde33293ed709c8c807"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.8/bin/linux/amd64/kubectl",
      "0fc709a8262be523293a18965771fedfba7466eda7ab4337feaa5c028aa46b1b"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.8/bin/darwin/arm64/kubectl",
      "01e5c58a305f309bd4f268125ba8a9c138a20ca9d602c74cd6b37a0d45fc5818"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.8/bin/linux/arm64/kubectl",
      "8a7371e54187249389a9aa222b150d61a4a745c121ab24dbcbb56d1ac2d0b912"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.8/bin/windows/amd64/kubectl.exe",
      "aa291c1e09267e193bb58cd6533b1824ca11ed0e56ca0869f614c6181d8a4bf2"
    )
  },
  "1.32.9": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.9/bin/darwin/amd64/kubectl",
      "fb7e76a98ee3923615e0e98e42105c7b77ca80c2310b977f56784515190c1941"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.9/bin/linux/amd64/kubectl",
      "509ae171bac7ad3b98cc49f5594d6bc84900cf6860f155968d1059fde3be5286"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.9/bin/darwin/arm64/kubectl",
      "8735038bb808e3c0acd5c553573f4ef2ac6a9ff508e077d46aa5b86b163bf7d2"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.9/bin/linux/arm64/kubectl",
      "d5f6b45ad81b7d199187a28589e65f83406e0610b036491a9abaa49bfd04a708"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.9/bin/windows/amd64/kubectl.exe",
      "730b26050e1395b5ba4dfc4cb6e84b8d79d01e2ca3e95328fb14ef296c30ab58"
    )
  },
  "1.32.10": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.10/bin/darwin/amd64/kubectl",
      "626b52743531779981e7800aaac53a9cf4fc9c0266311c33faaa3854617f6129"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.10/bin/linux/amd64/kubectl",
      "6e14ef4e509e9f3d1dfc2815643f832f853d2d9f6622d4a0f83f77c7e4014b57"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.10/bin/darwin/arm64/kubectl",
      "e6f7871732d5d80eb3987be13b986c9f8210f0f11f9b8b731330ba6c089056e0"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.10/bin/linux/arm64/kubectl",
      "1f4229526e16bf9f5b854fbf3bdb9c7040404a29c1d1e4193258b8a73de06e92"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.10/bin/windows/amd64/kubectl.exe",
      "b7a550dad8945c7c5fa2c86951cb517c90bf9a64f44cc153cfd1b7139dcd1a8e"
    )
  },
  "1.32.11": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.11/bin/darwin/amd64/kubectl",
      "8d0b610df71632d0e9b9c1aa16dde5ec666c05bf24e401ecf20fd27af16879ad"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.11/bin/linux/amd64/kubectl",
      "48581d0e808bd8b7d3c3fc014e86b170e25a987df04c8a879b982b28a5180815"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.11/bin/darwin/arm64/kubectl",
      "a39978a062f0df17d4a5551bd2e3a91eda90039196653935c50140be547141d3"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.11/bin/linux/arm64/kubectl",
      "b1c91c106ec20e61c5dff869e9a39e6af4fb96572bddaac9cce307dfa3ed2348"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.11/bin/windows/amd64/kubectl.exe",
      "8c350738ff800c42e4a11b026f73a656e09213a230a91b9a5646ea3a177edff3"
    )
  },
},
  "1.35.0": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.35.0/bin/darwin/amd64/kubectl",
      "2447cb78911b10a667202b078eeb30541ec78d1280c3682921dc81607e148d96"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.35.0/bin/linux/amd64/kubectl",
      "a2e984a18a0c063279d692533031c1eff93a262afcc0afdc517375432d060989"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.35.0/bin/darwin/arm64/kubectl",
      "cf699c56340dc775230fde4ef84237d27563ea6ef52164c7d078072b586c3918"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.35.0/bin/linux/arm64/kubectl",
      "58f82f9fe796c375c5c4b8439850b0f3f4d401a52434052f2df46035a8789e25"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.35.0/bin/windows/amd64/kubectl.exe",
      "4c5d14b8673bd55f813a8965ad70d5150e3960ee5f274025e2286aea3a0fa8b6"
    )
  },
  "1.34.0": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.34.0/bin/darwin/amd64/kubectl",
      "a5904061dd5c8e57d55e52c78fa23790e76de30924b26ba31be891e75710d7a9"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.34.0/bin/linux/amd64/kubectl",
      "cfda68cba5848bc3b6c6135ae2f20ba2c78de20059f68789c090166d6abc3e2c"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.34.0/bin/darwin/arm64/kubectl",
      "d491f4c47c34856188d38e87a27866bd94a66a57b8db3093a82ae43baf3bb20d"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.34.0/bin/linux/arm64/kubectl",
      "00b182d103a8a73da7a4d11e7526d0543dcf352f06cc63a1fde25ce9243f49a0"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.34.0/bin/windows/amd64/kubectl.exe",
      "856b6a92556452e249db940e7fdb8d8f8f622805d25f67de09a4d4d2da6f6132"
    )
  },
  "1.34.1": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.34.1/bin/darwin/amd64/kubectl",
      "bb211f2b31f2b3bc60562b44cc1e3b712a16a98e9072968ba255beb04cefcfdf"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.34.1/bin/linux/amd64/kubectl",
      "7721f265e18709862655affba5343e85e1980639395d5754473dafaadcaa69e3"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.34.1/bin/darwin/arm64/kubectl",
      "d80e5fa36f2b14005e5bb35d3a72818acb1aea9a081af05340a000e5fbdb2f76"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.34.1/bin/linux/arm64/kubectl",
      "420e6110e3ba7ee5a3927b5af868d18df17aae36b720529ffa4e9e945aa95450"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.34.1/bin/windows/amd64/kubectl.exe",
      "d118a8ddb0de15ff230189c85f5157e752405eb0ae8fa680d284de094c9a20f0"
    )
  },
  "1.34.2": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.34.2/bin/darwin/amd64/kubectl",
      "d2a71bb7dd7238287f2ba4efefbad4f98584170063f7d9e6c842f772d9255d45"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.34.2/bin/linux/amd64/kubectl",
      "9591f3d75e1581f3f7392e6ad119aab2f28ae7d6c6e083dc5d22469667f27253"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.34.2/bin/darwin/arm64/kubectl",
      "8f38d3a38ae317b00ebf90254dc274dd28d8c6eea4a4b30c5cb12d3d27017b6d"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.34.2/bin/linux/arm64/kubectl",
      "95df604e914941f3172a93fa8feeb1a1a50f4011dfbe0c01e01b660afc8f9b85"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.34.2/bin/windows/amd64/kubectl.exe",
      "7d34dcc49a185d64194ff3e952d5621b7da4f5562fa83df5acf305bd1f7de9cc"
    )
  },
  "1.34.3": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.34.3/bin/darwin/amd64/kubectl",
      "657afbd0e653c4ce3af1b5a645a4eaba282cf8eb2bcda7191ff60866e50e4d7f"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.34.3/bin/linux/amd64/kubectl",
      "ab60ca5f0fd60c1eb81b52909e67060e3ba0bd27e55a8ac147cbc2172ff14212"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.34.3/bin/darwin/arm64/kubectl",
      "e51367d2107d605f4edd7c2fb25897b0c0695a7de1a9f9d04cd6c9356b890b14"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.34.3/bin/linux/arm64/kubectl",
      "46913a7aa0327f6cc2e1cc2775d53c4a2af5e52f7fd8dacbfbfd098e757f19e9"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.34.3/bin/windows/amd64/kubectl.exe",
      "5ef6e0b019cfea5b0eff55b576c0118f64c0758a8bcbf52587c7f454f302f7bc"
    )
  },
  "1.33.1": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.33.1/bin/darwin/amd64/kubectl",
      "8d36a5c66142547ad16e332942fd16a0ca2b3346d9ebaab6c348de2c70d9d875"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.33.1/bin/linux/amd64/kubectl",
      "5de4e9f2266738fd112b721265a0c1cd7f4e5208b670f811861f699474a100a3"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.33.1/bin/darwin/arm64/kubectl",
      "8ae6823839993bb2e394c3cf1919748e530642c625dc9100159595301f53bdeb"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.33.1/bin/linux/arm64/kubectl",
      "d595d1a26b7444e0beb122e25750ee4524e74414bbde070b672b423139295ce6"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.33.1/bin/windows/amd64/kubectl.exe",
      "815c3c39984d1f7347486ad58b8e33e61ee87bc8ad79e0dbc9793e22200614fb"
    )
  },
  "1.33.2": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.33.2/bin/darwin/amd64/kubectl",
      "ff468749bd3b5f4f15ad36f2a437e65fcd3195a2081925140334429eaced1a8a"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.33.2/bin/linux/amd64/kubectl",
      "33d0cdec6967817468f0a4a90f537dfef394dcf815d91966ca651cc118393eea"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.33.2/bin/darwin/arm64/kubectl",
      "8730bf6dab538a1e9710a3668e2cd5f1bdc3c25c68b65a57c5418bdc3472769c"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.33.2/bin/linux/arm64/kubectl",
      "54dc02c8365596eaa2b576fae4e3ac521db9130e26912385e1e431d156f8344d"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.33.2/bin/windows/amd64/kubectl.exe",
      "c45a0fb477262eebd4a4a2936ea6bd10ce6a7db8f1356cff6e703c948538c76b"
    )
  },
  "1.33.3": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.33.3/bin/darwin/amd64/kubectl",
      "9652b55a58e84454196a7b9009f6d990d3961e2bd4bd03f64111d959282b46b1"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.33.3/bin/linux/amd64/kubectl",
      "2fcf65c64f352742dc253a25a7c95617c2aba79843d1b74e585c69fe4884afb0"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.33.3/bin/darwin/arm64/kubectl",
      "3de173356753bacb215e6dc7333f896b7f6ab70479362146c6acca6e608b3f53"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.33.3/bin/linux/arm64/kubectl",
      "3d514dbae5dc8c09f773df0ef0f5d449dfad05b3aca5c96b13565f886df345fd"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.33.3/bin/windows/amd64/kubectl.exe",
      "fbcb21ae1f8e0313ca44c9a3392f62523caf8c1a23b49c80e01cbf541060d592"
    )
  },
  "1.33.4": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.33.4/bin/darwin/amd64/kubectl",
      "4b39b8bb12e78ce801b39c9ec50421e3d6e144d8e3f113cd18e6d61709b8c73b"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.33.4/bin/linux/amd64/kubectl",
      "c2ba72c115d524b72aaee9aab8df8b876e1596889d2f3f27d68405262ce86ca1"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.33.4/bin/darwin/arm64/kubectl",
      "a44662db083fdd1b19ce55ba77eb64d51206310bbae90df90eb5d9e30ea54603"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.33.4/bin/linux/arm64/kubectl",
      "76cd7a2aa59571519b68c3943521404cbce55dafb7d8866f8d0ea2995b396eef"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.33.4/bin/windows/amd64/kubectl.exe",
      "15487c2a017af8ef8a5fbc2390af78b90f98d2909f23eeb684c64a8af3f7c4eb"
    )
  },
  "1.33.5": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.33.5/bin/darwin/amd64/kubectl",
      "ebdefb65c60c920510a605f13622e7eadb85bb83ba393d9eed2389bac30672b1"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.33.5/bin/linux/amd64/kubectl",
      "6a12d6c39e4a611a3687ee24d8c733961bb4bae1ae975f5204400c0a6930c6fc"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.33.5/bin/darwin/arm64/kubectl",
      "22f7256932c1c5205d7323a63d16253b8405ecccfd57c7a2484d3219c6822d3e"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.33.5/bin/linux/arm64/kubectl",
      "6db7c5d846c3b3ddfd39f3137a93fe96af3938860eefdbf2429805ee1656e381"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.33.5/bin/windows/amd64/kubectl.exe",
      "2fa5d21aa99afe994b1e929054d4ca701f7dd5e8124f8f1c83d28186474bc00b"
    )
  },
  "1.33.6": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.33.6/bin/darwin/amd64/kubectl",
      "a0f485c2b8296c84fda606dd585e2458a06a41235f1e96348cf64a2f527f6e77"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.33.6/bin/linux/amd64/kubectl",
      "d25d9b63335c038333bed785e9c6c4b0e41d791a09cac5f3e8df9862c684afbe"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.33.6/bin/darwin/arm64/kubectl",
      "ba6e00a0479d45a4aa59ad550ed0fd68696e73bd2d43d0e00213fba41f61fa54"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.33.6/bin/linux/arm64/kubectl",
      "3ab32d945a67a6000ba332bf16382fc3646271da6b7d751608b320819e5b8f38"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.33.6/bin/windows/amd64/kubectl.exe",
      "bc2e96179cce21fa3ef6e216fe853f41d08850f73f61a150d1485ee18a16acea"
    )
  },
  "1.33.7": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.33.7/bin/darwin/amd64/kubectl",
      "45be3f5293da84d97e86580a541b247fe3cec60196fdd6abd2b811d7dd4d3f1b"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.33.7/bin/linux/amd64/kubectl",
      "471d94e208a89be62eb776700fc8206cbef11116a8de2dc06fc0086b0015375b"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.33.7/bin/darwin/arm64/kubectl",
      "2e333f56d115081af83a48b5f31a91fb32852550f8117a0a31cf8bae2e601704"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.33.7/bin/linux/arm64/kubectl",
      "fa7ee98fdb6fba92ae05b5e0cde0abd5972b2d9a4a084f7052a1fd0dce6bc1de"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.33.7/bin/windows/amd64/kubectl.exe",
      "df8bead144a7a997a79c480083061955ddcd171f803631ac1239c0bd6a8f36ac"
    )
  },
  "1.32.0": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.0/bin/darwin/amd64/kubectl",
      "516585916f499077fac8c2fdd2a382818683f831020277472e6bcf8d1a6f9be4"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.0/bin/linux/amd64/kubectl",
      "646d58f6d98ee670a71d9cdffbf6625aeea2849d567f214bc43a35f8ccb7bf70"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.0/bin/darwin/arm64/kubectl",
      "5bfd5de53a054b4ef614c60748e28bf47441c7ed4db47ec3c19a3e2fa0eb5555"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.0/bin/linux/arm64/kubectl",
      "ba4004f98f3d3a7b7d2954ff0a424caa2c2b06b78c17b1dccf2acc76a311a896"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.0/bin/windows/amd64/kubectl.exe",
      "3601cb47c4d6a42b033a8f8fca68bc6f24baa99f5a1250fdb138d24a6c7cc749"
    )
  },
  "1.32.1": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.1/bin/darwin/amd64/kubectl",
      "8bffe90f5a034d392a0ba6fd7ee16c0d40b1dba1ccc4350821102c5d5c56d846"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.1/bin/linux/amd64/kubectl",
      "e16c80f1a9f94db31063477eb9e61a2e24c1a4eee09ba776b029048f5369db0c"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.1/bin/darwin/arm64/kubectl",
      "5b89f9598e2e7da04cc0b5dd6e8daca01d23855fd00c8ea259fd2aab993114db"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.1/bin/linux/arm64/kubectl",
      "98206fd83a4fd17f013f8c61c33d0ae8ec3a7c53ec59ef3d6a0a9400862dc5b2"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.1/bin/windows/amd64/kubectl.exe",
      "b6378f34dcab2d411fb7a89ba700df0f784c0b063dd02dbb92396a72c4d3104e"
    )
  },
  "1.32.2": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.2/bin/darwin/amd64/kubectl",
      "371b8fbd481e1e9052ace16d9c243e92618a2ea9a18c1aaf235d35fef20c0c32"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.2/bin/linux/amd64/kubectl",
      "4f6a959dcc5b702135f8354cc7109b542a2933c46b808b248a214c1f69f817ea"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.2/bin/darwin/arm64/kubectl",
      "31b6318deaa72014b72121e1c7a2e12496d077cee49bbeda94250aec4c978ffb"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.2/bin/linux/arm64/kubectl",
      "7381bea99c83c264100f324c2ca6e7e13738a73b8928477ac805991440a065cd"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.2/bin/windows/amd64/kubectl.exe",
      "cf51a1c6bf3b6ba6a5b549d1debf8aa6afb00c4c5a3d5d4bb1072f54cbe4390f"
    )
  },
  "1.32.3": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.3/bin/darwin/amd64/kubectl",
      "b814c523071cd09e27c88d8c87c0e9b054ca0cf5c2b93baf3127750a4f194d5b"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.3/bin/linux/amd64/kubectl",
      "ab209d0c5134b61486a0486585604a616a5bb2fc07df46d304b3c95817b2d79f"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.3/bin/darwin/arm64/kubectl",
      "a110af64fc31e2360dd0f18e4110430e6eedda1a64f96e9d89059740a7685bbd"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.3/bin/linux/arm64/kubectl",
      "6c2c91e760efbf3fa111a5f0b99ba8975fb1c58bb3974eca88b6134bcf3717e2"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.3/bin/windows/amd64/kubectl.exe",
      "3fd1576a902ecf713f7d6390ae01799e370883e0341177ee09dbdc362db953e3"
    )
  },
  "1.32.4": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.4/bin/darwin/amd64/kubectl",
      "061f65fe5405538f6fe8edd3c3373f479a1d59944ebf6268905535a617151d16"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.4/bin/linux/amd64/kubectl",
      "10d739e9af8a59c9e7a730a2445916e04bc9cbb44bc79d22ce460cd329fa076c"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.4/bin/darwin/arm64/kubectl",
      "01344900ac3c2c97a3290e9465d36f0dea20ca4533d226dfbe7c9a90e80ff9d4"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.4/bin/linux/arm64/kubectl",
      "c6f96d0468d6976224f5f0d81b65e1a63b47195022646be83e49d38389d572c2"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.4/bin/windows/amd64/kubectl.exe",
      "8e93d01f8efe80db614cf7dc422f9bb3fbad1b16f82d13f0ea70441441e486e4"
    )
  },
  "1.32.5": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.5/bin/darwin/amd64/kubectl",
      "f357d30fc338eb914e6e7a5e0408852d3011fac18d98f4484c4861c4c2cead3c"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.5/bin/linux/amd64/kubectl",
      "aaa7e6ff3bd28c262f2d95c8c967597e097b092e9b79bcb37de699e7488e3e7b"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.5/bin/darwin/arm64/kubectl",
      "b3b08783545e735b030376627133ddf53dc0e2c2ed4c413d87d4bcd7c2b0c632"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.5/bin/linux/arm64/kubectl",
      "9edee84103e63c40a37cd15bd11e04e7835f65cb3ff5a50972058ffc343b4d96"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.5/bin/windows/amd64/kubectl.exe",
      "df01c85015fa2b19fa7f92a7704aae9de5b5dc70fed32a01bd26f57f7ba563a5"
    )
  },
  "1.32.6": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.6/bin/darwin/amd64/kubectl",
      "ad0c1880b1bcd36869d75a54c3401b718c091d75d11d08f57034fb7b4712f6ef"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.6/bin/linux/amd64/kubectl",
      "0e31ebf882578b50e50fe6c43e3a0e3db61f6a41c9cded46485bc74d03d576eb"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.6/bin/darwin/arm64/kubectl",
      "8ac847473a6794dd35d2b980c9249b79dedb6e234d00fd0f223cf6b67be12999"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.6/bin/linux/arm64/kubectl",
      "f7bac84f8c35f55fb2c6ad167beb59eba93de5924b50bbaa482caa14ff480eec"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.6/bin/windows/amd64/kubectl.exe",
      "3b4aabd90c52e01557f08cb2747431a78767cf978646812a69d8d53d73f7049e"
    )
  },
  "1.32.7": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.7/bin/darwin/amd64/kubectl",
      "050a5b4227a07c6d7f5add1863323f9db90b97c12874e2218224c9be74286980"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.7/bin/linux/amd64/kubectl",
      "b8f24d467a8963354b028796a85904824d636132bef00988394cadacffe959c9"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.7/bin/darwin/arm64/kubectl",
      "07a3511f02763076859e37abae33e1513285feec0482798a547441128f84662b"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.7/bin/linux/arm64/kubectl",
      "232f6e517633fbb4696c9eb7a0431ee14b3fccbb47360b4843d451e0d8c9a3a2"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.7/bin/windows/amd64/kubectl.exe",
      "06468f371634191e8a3d7ec63c463dbf81a27518a9f87309153e67d760f94eff"
    )
  },
  "1.32.8": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.8/bin/darwin/amd64/kubectl",
      "a00a8fadd4a7ca520e68e88a640ca60b4601695f68b8dcde33293ed709c8c807"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.8/bin/linux/amd64/kubectl",
      "0fc709a8262be523293a18965771fedfba7466eda7ab4337feaa5c028aa46b1b"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.8/bin/darwin/arm64/kubectl",
      "01e5c58a305f309bd4f268125ba8a9c138a20ca9d602c74cd6b37a0d45fc5818"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.8/bin/linux/arm64/kubectl",
      "8a7371e54187249389a9aa222b150d61a4a745c121ab24dbcbb56d1ac2d0b912"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.8/bin/windows/amd64/kubectl.exe",
      "aa291c1e09267e193bb58cd6533b1824ca11ed0e56ca0869f614c6181d8a4bf2"
    )
  },
  "1.32.9": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.9/bin/darwin/amd64/kubectl",
      "fb7e76a98ee3923615e0e98e42105c7b77ca80c2310b977f56784515190c1941"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.9/bin/linux/amd64/kubectl",
      "509ae171bac7ad3b98cc49f5594d6bc84900cf6860f155968d1059fde3be5286"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.9/bin/darwin/arm64/kubectl",
      "8735038bb808e3c0acd5c553573f4ef2ac6a9ff508e077d46aa5b86b163bf7d2"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.9/bin/linux/arm64/kubectl",
      "d5f6b45ad81b7d199187a28589e65f83406e0610b036491a9abaa49bfd04a708"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.9/bin/windows/amd64/kubectl.exe",
      "730b26050e1395b5ba4dfc4cb6e84b8d79d01e2ca3e95328fb14ef296c30ab58"
    )
  },
  "1.32.10": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.10/bin/darwin/amd64/kubectl",
      "626b52743531779981e7800aaac53a9cf4fc9c0266311c33faaa3854617f6129"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.10/bin/linux/amd64/kubectl",
      "6e14ef4e509e9f3d1dfc2815643f832f853d2d9f6622d4a0f83f77c7e4014b57"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.10/bin/darwin/arm64/kubectl",
      "e6f7871732d5d80eb3987be13b986c9f8210f0f11f9b8b731330ba6c089056e0"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.10/bin/linux/arm64/kubectl",
      "1f4229526e16bf9f5b854fbf3bdb9c7040404a29c1d1e4193258b8a73de06e92"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.10/bin/windows/amd64/kubectl.exe",
      "b7a550dad8945c7c5fa2c86951cb517c90bf9a64f44cc153cfd1b7139dcd1a8e"
    )
  },
  "1.32.11": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.32.11/bin/darwin/amd64/kubectl",
      "8d0b610df71632d0e9b9c1aa16dde5ec666c05bf24e401ecf20fd27af16879ad"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.32.11/bin/linux/amd64/kubectl",
      "48581d0e808bd8b7d3c3fc014e86b170e25a987df04c8a879b982b28a5180815"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.32.11/bin/darwin/arm64/kubectl",
      "a39978a062f0df17d4a5551bd2e3a91eda90039196653935c50140be547141d3"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.32.11/bin/linux/arm64/kubectl",
      "b1c91c106ec20e61c5dff869e9a39e6af4fb96572bddaac9cce307dfa3ed2348"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.32.11/bin/windows/amd64/kubectl.exe",
      "8c350738ff800c42e4a11b026f73a656e09213a230a91b9a5646ea3a177edff3"
    )
  },
},
  "1.30.1": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.30.1/bin/darwin/amd64/kubectl",
      "eaefb69cf908b7473d2dce0ba894c956b7e1ad5a4987a96d68a279f5597bb22d"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.30.1/bin/linux/amd64/kubectl",
      "5b86f0b06e1a5ba6f8f00e2b01e8ed39407729c4990aeda961f83a586f975e8a"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.30.1/bin/darwin/arm64/kubectl",
      "55dec3c52702bd68488a5c1ab840b79ea9e73e4b9f597bcf75b201c55d0bd280"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.30.1/bin/linux/arm64/kubectl",
      "d90446719b815e3abfe7b2c46ddf8b3fda17599f03ab370d6e47b1580c0e869e"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.30.1/bin/windows/amd64/kubectl.exe",
      "f7391a2de0491caadedb5178ac2485cbf104189b2e0f3d6c577bd6ea1892898f"
    )
  },
  "1.30.2": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.30.2/bin/darwin/amd64/kubectl",
      "0371b7bcc060f533170ac6fb99bc9aa13fdf3fa005276e3eb14eed162ed8a3a9"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.30.2/bin/linux/amd64/kubectl",
      "c6e9c45ce3f82c90663e3c30db3b27c167e8b19d83ed4048b61c1013f6a7c66e"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.30.2/bin/darwin/arm64/kubectl",
      "ffcba19e77b9521f5779ab32cfcd4bfcc9d20cd42c2f075c7c5aef83f32754ae"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.30.2/bin/linux/arm64/kubectl",
      "56becf07105fbacd2b70f87f3f696cfbed226cb48d6d89ed7f65ba4acae3f2f8"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.30.2/bin/windows/amd64/kubectl.exe",
      "59a5b1028f6e3aea046b103f9a787cf4d70067d554053f713d691b14df3d9bb4"
    )
  },
  "1.30.3": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.30.3/bin/darwin/amd64/kubectl",
      "b3ccb0ba6f7972074b0a1e13340307abfd5a5eef540c521a88b368891ec5cd6b"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.30.3/bin/linux/amd64/kubectl",
      "abd83816bd236b266c3643e6c852b446f068fe260f3296af1a25b550854ec7e5"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.30.3/bin/darwin/arm64/kubectl",
      "71f3febd165423991e0aabef5750cb8de6fc43e93ea130767d12eb183cc63a5b"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.30.3/bin/linux/arm64/kubectl",
      "c6f9568f930b16101089f1036677bb15a3185e9ed9b8dbce2f518fb5a52b6787"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.30.3/bin/windows/amd64/kubectl.exe",
      "4d066fc70ebfaad0da43c93daebdb62208993c04ea9a2b5e9ba459b18d6a1c81"
    )
  },
  "1.30.4": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.30.4/bin/darwin/amd64/kubectl",
      "ce1b79f0720509b7e78e73f4cd8d41d8ea46256a10a16f38ddeee6ff139a2625"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.30.4/bin/linux/amd64/kubectl",
      "2ffd023712bbc1a9390dbd8c0c15201c165a69d394787ef03eda3eccb4b9ac06"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.30.4/bin/darwin/arm64/kubectl",
      "978674da62282da697d889c33e0cc36f4b7ecb3a4d1ff73fc93e6e83877d5945"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.30.4/bin/linux/arm64/kubectl",
      "1d8b4e6443c7df8e92a065d88d146142a202fea5ec694135b83d9668529ea3b1"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.30.4/bin/windows/amd64/kubectl.exe",
      "c9ddaf742a8d4bd8a1b26ff6981976154109b829d544b259464639451e8ddae6"
    )
  },
  "1.29.0": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.29.0/bin/darwin/amd64/kubectl",
      "d69c2b0929070e42518b304758fbe05cf76c4fb60d36e93bb667d7b76e582124"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.29.0/bin/linux/amd64/kubectl",
      "0e03ab096163f61ab610b33f37f55709d3af8e16e4dcc1eb682882ef80f96fd5"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.29.0/bin/darwin/arm64/kubectl",
      "403beb5d64d8a8517f808a320619a28adc89003b1b710f02421933a9ee4eb968"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.29.0/bin/linux/arm64/kubectl",
      "8f7a4bd6bae900a4ddab12bd1399aa652c0d59ea508f39b910e111d248893ff7"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.29.0/bin/windows/amd64/kubectl.exe",
      "0ba8da57990757e3bc02df838d36adaf0586e557f518d0149f00e63ac2057066"
    )
  },
  "1.29.1": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.29.1/bin/darwin/amd64/kubectl",
      "c4da86e5c0fc9415db14a48d9ef1515b0b472346cbc9b7f015175b6109505d2c"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.29.1/bin/linux/amd64/kubectl",
      "69ab3a931e826bf7ac14d38ba7ca637d66a6fcb1ca0e3333a2cafdf15482af9f"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.29.1/bin/darwin/arm64/kubectl",
      "c31b99d7bf0faa486a6554c5f96e36af4821a488e90176a12ba18298bc4c8fb0"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.29.1/bin/linux/arm64/kubectl",
      "96d6dc7b2bdcd344ce58d17631c452225de5bbf59b83fd3c89c33c6298fb5d8b"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.29.1/bin/windows/amd64/kubectl.exe",
      "d48a0b160f361bac2d7cb3e19c755e70f2ae8d7d15c89f52d33b66dd9ba5b89c"
    )
  },
  "1.29.2": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.29.2/bin/darwin/amd64/kubectl",
      "bb04d9450d9c9fa120956c5cc7c8dfaa700297038ff9c941741e730b02bbd1f3"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.29.2/bin/linux/amd64/kubectl",
      "7816d067740f47f949be826ac76943167b7b3a38c4f0c18b902fffa8779a5afa"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.29.2/bin/darwin/arm64/kubectl",
      "ce030f86625df96560402573d86d4e6f4b8b956ca3e3b9df57cb8ccf2b9a540c"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.29.2/bin/linux/arm64/kubectl",
      "3507ecb4224cf05ae2151a98d4932253624e7762159936d5347b19fe037655ca"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.29.2/bin/windows/amd64/kubectl.exe",
      "5107162e20ef6e6f06c2db37e56da5db552858d83fa43b51787bf48c6e6d1caf"
    )
  },
  "1.29.3": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.29.3/bin/darwin/amd64/kubectl",
      "1a1f9040bce74fb28c475dc157a86565fcabf883a697ca576993ab8372935836"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.29.3/bin/linux/amd64/kubectl",
      "89c0435cec75278f84b62b848b8c0d3e15897d6947b6c59a49ddccd93d7312bf"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.29.3/bin/darwin/arm64/kubectl",
      "b54bf7a3f4d52117b79e4d4f0d7273a93cb60bad54a87f3ab35c6800243cbb8e"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.29.3/bin/linux/arm64/kubectl",
      "191a96b27e3c6ae28b330da4c9bfefc9592762670727df4fcf124c9f1d5a466a"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.29.3/bin/windows/amd64/kubectl.exe",
      "0192ce501a39ba1bb6cc7f1971a6004bc4b57d2c27400befaacaa4560dfff46e"
    )
  },
  "1.29.4": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.29.4/bin/darwin/amd64/kubectl",
      "7af9b8a233c49ad5eecb59004719e0bc07972492b674ebbce2919e53326b55b2"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.29.4/bin/linux/amd64/kubectl",
      "10e343861c3cb0010161e703307ba907add2aeeeaffc6444779ad915f9889c88"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.29.4/bin/darwin/arm64/kubectl",
      "b3a881e6208aa41275a97481676a8c8a3c16282f3cd7b441b17f258a054012f1"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.29.4/bin/linux/arm64/kubectl",
      "61537408eedcad064d7334384aed508a8aa1ea786311b87b505456a2e0535d36"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.29.4/bin/windows/amd64/kubectl.exe",
      "23ea3fce3a784b28e9445c1f813bdcdbd1270f92711f44f6e5772fa2ec2ce238"
    )
  },
  "1.29.5": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.29.5/bin/darwin/amd64/kubectl",
      "395082ef84594ea4cb170d599056406ed2cf39555b53e92e0caee013c1ed5cdf"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.29.5/bin/linux/amd64/kubectl",
      "603c8681fc0d8609c851f9cc58bcf55eeb97e2934896e858d0232aa8d1138366"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.29.5/bin/darwin/arm64/kubectl",
      "23b09c126c0a0b71b58cc725a32cf84f1753242b3892dfd762511f2da6cce165"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.29.5/bin/linux/arm64/kubectl",
      "9ee9168def12ac6a6c0c6430e0f73175e756ed262db6040f8aa2121ad2c1f62e"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.29.5/bin/windows/amd64/kubectl.exe",
      "8de419ccecdde90172345e7d12a63de42c217d28768d84c2398d932b44d73489"
    )
  },
  "1.29.6": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.29.6/bin/darwin/amd64/kubectl",
      "d6a844991d3853d9928a7593f583157403ea322ff712d7659b16e621fca00d79"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.29.6/bin/linux/amd64/kubectl",
      "339553c919874ebe3b719e9e1fcd68b55bc8875f9b5a005cf4c028738d54d309"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.29.6/bin/darwin/arm64/kubectl",
      "0b7a3cd78503faf45c6506d594b586f58b9904ad48452466834397641d58d6f5"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.29.6/bin/linux/arm64/kubectl",
      "21816488cf3af4cf2b956ee58f7afc5b4964c29488f63756f5ddcf09b0df5be9"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.29.6/bin/windows/amd64/kubectl.exe",
      "4c9cb51b40598f97bf2506da76d9482fa55e57adadaa49279e21e0cf6168ec18"
    )
  },
  "1.29.7": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.29.7/bin/darwin/amd64/kubectl",
      "e747b90725ebdac7b8a88621fc48ee56fabf5319da3080fa5855712e81fc88f8"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.29.7/bin/linux/amd64/kubectl",
      "e3df008ef60ea50286ea93c3c40a020e178a338cea64a185b4e21792d88c75d6"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.29.7/bin/darwin/arm64/kubectl",
      "f987c6a8cb769ec5062024ef27e2255bf8bc290d47f41b0fb974bb58094e11a7"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.29.7/bin/linux/arm64/kubectl",
      "7b6649aaa298be728c5fb7ccb65f98738a4e8bda0741afbd5a9ed9e488c0e725"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.29.7/bin/windows/amd64/kubectl.exe",
      "c4f76c46b792395e81c72a56ed55893bf5a31e3d1b4d67556c06f37aa7122d10"
    )
  },
  "1.29.8": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.29.8/bin/darwin/amd64/kubectl",
      "b1d780b97c36a2470c603a804476e85ad8fc4e13bb7761fd19180d79f8b06081"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.29.8/bin/linux/amd64/kubectl",
      "038454e0d79748aab41668f44ca6e4ac8affd1895a94f592b9739a0ae2a5f06a"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.29.8/bin/darwin/arm64/kubectl",
      "c48ad5e96b1dab12e504535fb2341476b3fe543d314cf90f98b11326ee0ed71d"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.29.8/bin/linux/arm64/kubectl",
      "adf0007e702e05f59fb8de159463765c4440f872515bd04c24939d9c8fb5e4c7"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.29.8/bin/windows/amd64/kubectl.exe",
      "7608329ae2f92aee0b615f46a09ba550093548bcd09662e17e3d7097421a86c7"
    )
  },
  "1.28.0": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.28.0/bin/darwin/amd64/kubectl",
      "6db117a55a14a47c0dcf9144c31780c6de0c3c84ccb9a297de0d9e6fc481534d"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.28.0/bin/linux/amd64/kubectl",
      "4717660fd1466ec72d59000bb1d9f5cdc91fac31d491043ca62b34398e0799ce"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.28.0/bin/darwin/arm64/kubectl",
      "5d74042f5972b342a02636cf5969d4d73234f2d3afe84fe5ddaaa4baff79cdd8"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.28.0/bin/linux/arm64/kubectl",
      "f5484bd9cac66b183c653abed30226b561f537d15346c605cc81d98095f1717c"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.28.0/bin/windows/amd64/kubectl.exe",
      "ee15a9ea4796e1acad1188b200155afdfde01083850570a8218e2f03fbbff019"
    )
  },
  "1.28.1": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.28.1/bin/darwin/amd64/kubectl",
      "d6b8f2bac5f828478eade0acf15fb7dde02d7613fc9e644dc019a7520d822a1a"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.28.1/bin/linux/amd64/kubectl",
      "e7a7d6f9d06fab38b4128785aa80f65c54f6675a0d2abef655259ddd852274e1"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.28.1/bin/darwin/arm64/kubectl",
      "8fe9f753383574863959335d8b830908e67a40c3f51960af63892d969bfc1b10"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.28.1/bin/linux/arm64/kubectl",
      "46954a604b784a8b0dc16754cfc3fa26aabca9fd4ffd109cd028bfba99d492f6"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.28.1/bin/windows/amd64/kubectl.exe",
      "810a6f576367bfb7420ebf318fd3fe95fd445c785e39042b9edeaae55d834d44"
    )
  },
  "1.28.2": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.28.2/bin/darwin/amd64/kubectl",
      "fb90ffc2b1751537ec1131276dd3a2f165464191025c3392a0ee2ed1575a19f0"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.28.2/bin/linux/amd64/kubectl",
      "c922440b043e5de1afa3c1382f8c663a25f055978cbc6e8423493ec157579ec5"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.28.2/bin/darwin/arm64/kubectl",
      "a00300f8463f659f4eeb04ff2ad92fec5f552e3de041bf4eae23587cc7408fbc"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.28.2/bin/linux/arm64/kubectl",
      "ea6d89b677a8d9df331a82139bb90d9968131530b94eab26cee561531eff4c53"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.28.2/bin/windows/amd64/kubectl.exe",
      "b52b17a1fcfa6cc1cb3e480a0d6066c7d7175159ecd4a62ac8e44d8ce2c7a931"
    )
  },
  "1.28.3": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.28.3/bin/darwin/amd64/kubectl",
      "3130398698b131ceae24879745aa536f9ec38790b397d806e4f6db03d65e4abb"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.28.3/bin/linux/amd64/kubectl",
      "0c680c90892c43e5ce708e918821f92445d1d244f9b3d7513023bcae9a6246d1"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.28.3/bin/darwin/arm64/kubectl",
      "b1b83c298177b849f9a8564b0dfcde8ecabc646b7f409d18001b8d4de407e0bf"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.28.3/bin/linux/arm64/kubectl",
      "06511f03e34d8ee350bd55717845e27ebec3116526db7c60092eeb33a475a337"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.28.3/bin/windows/amd64/kubectl.exe",
      "0f25d33ba68f526afeceb11e263a97473e8c42f856b05f0b9c4863bead6dac4c"
    )
  },
  "1.28.4": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.28.4/bin/darwin/amd64/kubectl",
      "70ac52dab10e4e276ce49a5cde05d495149ecaa0dc3126ba50611542ef0e6d56"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.28.4/bin/linux/amd64/kubectl",
      "893c92053adea6edbbd4e959c871f5c21edce416988f968bec565d115383f7b8"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.28.4/bin/darwin/arm64/kubectl",
      "7e49c25887fe7cbf2a4b01a41604bdf1edc18a9faede45c810713af2b3a28361"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.28.4/bin/linux/arm64/kubectl",
      "edf1e17b41891ec15d59dd3cc62bcd2cdce4b0fd9c2ee058b0967b17534457d7"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.28.4/bin/windows/amd64/kubectl.exe",
      "6fd6e38de7ec6362fc6582ee439014ed1be661e04eb851c6cdc6759856929ef2"
    )
  },
  "1.28.5": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.28.5/bin/darwin/amd64/kubectl",
      "b6ca01e3f21bc5e7fe711c917d3f59b06c0849d688ccc4590a82f1098e078849"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.28.5/bin/linux/amd64/kubectl",
      "2a44c0841b794d85b7819b505da2ff3acd5950bd1bcd956863714acc80653574"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.28.5/bin/darwin/arm64/kubectl",
      "9c9ec9e69c7fe989fde601b5a71035782271fb788bb92b24eed20bc4fbc8b310"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.28.5/bin/linux/arm64/kubectl",
      "f87fe017ae3ccfd93df03bf17edd4089672528107f230563b8c9966909661ef2"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.28.5/bin/windows/amd64/kubectl.exe",
      "3e632093a6e02bde8962c308148edab5b1fb4d8a71b41810866d4cedbc04317c"
    )
  },
  "1.28.6": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.28.6/bin/darwin/amd64/kubectl",
      "2853d5a40a618a8d25f3cb30e72d03a8394a92e32842d60428271755e46bf2fe"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.28.6/bin/linux/amd64/kubectl",
      "c8351fe0611119fd36634dd3f53eb94ec1a2d43ef9e78b92b4846df5cc7aa7e3"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.28.6/bin/darwin/arm64/kubectl",
      "35a3ae87eee5af0a561f90d5139bda21da2f41884ec37bfe31547d271b0b2339"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.28.6/bin/linux/arm64/kubectl",
      "0de705659a80c3fef01df43cc0926610fe31482f728b0f992818abd9bdcd2cb9"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.28.6/bin/windows/amd64/kubectl.exe",
      "a9de70249210a7638e35644275ba9f9a2737df2f4d21bf32ceb56fc89c55e888"
    )
  },
  "1.28.7": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.28.7/bin/darwin/amd64/kubectl",
      "69bdb3f618e40de912400c2e56d085325f872abc604e87a4f9f2da6bb25c8aa4"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.28.7/bin/linux/amd64/kubectl",
      "aff42d3167685e4d8e86fda0ad9c6ce6ec6c047bc24d608041d54717a18192ba"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.28.7/bin/darwin/arm64/kubectl",
      "250104cd000aa31a45075c82b1267938833e1ca8f9322a9512f96caa489b68ec"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.28.7/bin/linux/arm64/kubectl",
      "13d547495bdea49b223fe06bffb6d2bef96436634847f759107655aa80fc990e"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.28.7/bin/windows/amd64/kubectl.exe",
      "b6620c4ff153bbdf4ff26890b657e5b76a4d8fdfe4e46531dc495173acea26cf"
    )
  },
  "1.28.8": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.28.8/bin/darwin/amd64/kubectl",
      "959acd160b2c858c08426c64f533e768581182428bf9afd6965e1d0f37909b16"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.28.8/bin/linux/amd64/kubectl",
      "e02aad5c0bac52c970700b814645b62c4f18b634144398ac344875dbaf1072f8"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.28.8/bin/darwin/arm64/kubectl",
      "280b9ad125bb648ef81839349e2b921db6d674cc153b3c6116d65383260aeae5"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.28.8/bin/linux/arm64/kubectl",
      "93d60dd36093b4c719f1f1bafcf59437c17cb2209341c7c94771e7dd9acdab33"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.28.8/bin/windows/amd64/kubectl.exe",
      "fd310b2faa2f4fd0a06d607cc79c115ea3251eeeb83c5442ca225211ecee61f9"
    )
  },
  "1.28.9": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.28.9/bin/darwin/amd64/kubectl",
      "99df1db1c735e7f6aceb1f53a0c8c313f51be34cda9d964b0764e96dd7275d09"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.28.9/bin/linux/amd64/kubectl",
      "b4693d0b22f509250694b10c7727c42b427d570af04f2065fe23a55d6c0051f1"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.28.9/bin/darwin/arm64/kubectl",
      "48cb2db4cc76a9a3a0f5d7f4dd9bd839196b39d9726247384b91e32e6a83be94"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.28.9/bin/linux/arm64/kubectl",
      "e0341d3973213f8099e7fcbbf6d1d506967bc2b7a4faac3fb3b4340f226e9b2f"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.28.9/bin/windows/amd64/kubectl.exe",
      "e82439aca2a1d7a1138912eb5d25d191ff5f1fc6d3753723f8b36168e6b95db8"
    )
  },
  "1.28.10": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.28.10/bin/darwin/amd64/kubectl",
      "426e1cdfe990b6f0e26d3b5243e079650cc65d6b4b5374824197c5d471f99cff"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.28.10/bin/linux/amd64/kubectl",
      "389c17a9700a4b01ebb055e39b8bc0886330497440dde004b5ed90f2a3a028db"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.28.10/bin/darwin/arm64/kubectl",
      "da88c27eeab82512f9a23c6d80a9c6cc933d3514d3cd4fb215c8b57868a78195"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.28.10/bin/linux/arm64/kubectl",
      "e659d23d442c2706debe5b96742326c0a1e1d7b5c695a9fe7dfe8ea7402caee8"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.28.10/bin/windows/amd64/kubectl.exe",
      "eddfbb875a7458a474b3b9ed089369baa8a782b9921be01ecb8abd4e9f1097d9"
    )
  },
  "1.28.11": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.28.11/bin/darwin/amd64/kubectl",
      "0eb7314ee18185d9e4782f70e79b1554e5d787d8e4a590532ab90b64b94384ac"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.28.11/bin/linux/amd64/kubectl",
      "1dba63e1a5c9520fc516c6e817924d927b9b83b8e08254c8fe2a2edb65da7a9c"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.28.11/bin/darwin/arm64/kubectl",
      "85f752c0ac9e7a560da57e904ef0dee310fe53fc6ad39c2e301b1dda4a21cf96"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.28.11/bin/linux/arm64/kubectl",
      "7984a98d52365d190b6f56caa962339a7228b6f432e58ba5f1b1e60dbedac275"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.28.11/bin/windows/amd64/kubectl.exe",
      "83c7867c133e895cf5c97314145cf4daa1dbe5ffa4b344bbbc8687100fbf500f"
    )
  },
  "1.28.12": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.28.12/bin/darwin/amd64/kubectl",
      "c148882b1fe3de2fb5534f79dbd4e5e6ae08ffafd406c97e25b0042c957dc876"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.28.12/bin/linux/amd64/kubectl",
      "e8aee7c9206c00062ced394418a17994b58f279a93a1be1143b08afe1758a3a2"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.28.12/bin/darwin/arm64/kubectl",
      "353c28cb6cc687ce74bdfc15c462c4e2400275fd4ff4040c79a28e982190bf28"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.28.12/bin/linux/arm64/kubectl",
      "f7e01dfffebb1d5811c37d558f28eefd80cbfadc0b9783b0b0ebf37c40c5c891"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.28.12/bin/windows/amd64/kubectl.exe",
      "2cb52dd2fc67d76fd99ab154364f67c09c3f7159c6d52e941a9349801cf48ba0"
    )
  },
  "1.28.13": {
    "darwin_amd64": (
      "https://dl.k8s.io/release/v1.28.13/bin/darwin/amd64/kubectl",
      "da83c1db4a2270584e4562f0ff658bd2cb8d56997a5441b62920030a4d48cb27"
    ),
    "linux_amd64": (
      "https://dl.k8s.io/release/v1.28.13/bin/linux/amd64/kubectl",
      "d7d363dd5a4c95444329bc5239b8718ebe84a043052958b2f15ee2feef9a28c6"
    ),
    "darwin_arm64": (
      "https://dl.k8s.io/release/v1.28.13/bin/darwin/arm64/kubectl",
      "b908e29c56c87e3d09dc0258b3ead8a32c23eea5e8619a6a7134419c00d1141e"
    ),
    "linux_arm64": (
      "https://dl.k8s.io/release/v1.28.13/bin/linux/arm64/kubectl",
      "a22d234724b82101e1f17e95ab60e0e13c91a0fe17ad0890b3d92681cd551bfa"
    ),
    "windows_amd64": (
      "https://dl.k8s.io/release/v1.28.13/bin/windows/amd64/kubectl.exe",
      "99afa16f9d4ccd47daa473dbe255d3b4f1f142e3e877db1b4ee33023717bdd8c"
    )
  }
}

DEFAULT_KUBECTL_VERSION = "1.35.0"
DEFAULT_KUBECTL_REPOSITORY = "kubectl"

KUBECTL_PLATFORMS = {
    "darwin_amd64": struct(
        release_platform = "macos-amd64",
        compatible_with = [
            "@platforms//os:macos",
            "@platforms//cpu:x86_64",
        ],
    ),
    "darwin_arm64": struct(
        release_platform = "macos-arm64",
        compatible_with = [
            "@platforms//os:macos",
            "@platforms//cpu:aarch64",
        ],
    ),
    "linux_amd64": struct(
        release_platform = "linux-amd64",
        compatible_with = [
            "@platforms//os:linux",
            "@platforms//cpu:x86_64",
        ],
    ),
    "linux_arm64": struct(
        release_platform = "linux-arm64",
        compatible_with = [
            "@platforms//os:linux",
            "@platforms//cpu:aarch64",
        ],
    ),
}

KubectlInfo = provider(
    doc = "Provide info for executing kubectl",
    fields = {
        "bin": "Executable kubectl binary",
    },
)

def _kubectl_toolchain_impl(ctx):
    binary = ctx.file.bin

    # Make the $(KUBECTL_BIN) variable available in places like genrules.
    # See https://docs.bazel.build/versions/main/be/make-variables.html#custom_variables
    template_variables = platform_common.TemplateVariableInfo({
        "KUBECTL_BIN": binary.path,
    })
    default_info = DefaultInfo(
        files = depset([binary]),
        runfiles = ctx.runfiles(files = [binary]),
    )
    kubectl_info = KubectlInfo(
        bin = binary,
    )

    # Export all the providers inside our ToolchainInfo
    # so the resolved_toolchain rule can grab and re-export them.
    toolchain_info = platform_common.ToolchainInfo(
        kubectlinfo = kubectl_info,
        template_variables = template_variables,
        default = default_info,
    )

    return [default_info, toolchain_info, template_variables]

kubectl_toolchain = rule(
    implementation = _kubectl_toolchain_impl,
    attrs = {
        "bin": attr.label(
            mandatory = True,
            allow_single_file = True,
            executable = True,
            cfg = "exec",
        ),
    },
)

def _kubectl_toolchains_repo_impl(rctx):
    # Expose a concrete toolchain which is the result of Bazel resolving the toolchain
    # for the execution or target platform.
    # Workaround for https://github.com/bazelbuild/bazel/issues/14009
    starlark_content = """# @generated by @rules_k8s_cd//kubectl_toolchain.bzl

# Forward all the providers
def _resolved_toolchain_impl(ctx):
    toolchain_info = ctx.toolchains["@rules_k8s_cd//lib:kubectl_toolchain_type"]
    return [
        toolchain_info,
        toolchain_info.default,
        toolchain_info.kubectlinfo,
        toolchain_info.template_variables,
    ]

# Copied from java_toolchain_alias
# https://cs.opensource.google/bazel/bazel/+/master:tools/jdk/java_toolchain_alias.bzl
resolved_toolchain = rule(
    implementation = _resolved_toolchain_impl,
    toolchains = ["@rules_k8s_cd//lib:kubectl_toolchain_type"],
)
"""
    rctx.file("defs.bzl", starlark_content)

    build_content = """# @generated by @rules_k8s_cd//lib/private:kubectl_toolchain.bzl
#
# These can be registered in the workspace file or passed to --extra_toolchains flag.
# By default all these toolchains are registered by the kubectl_register_toolchains macro
# so you don't normally need to interact with these targets.

load(":defs.bzl", "resolved_toolchain")

resolved_toolchain(name = "resolved_toolchain", visibility = ["//visibility:public"])

"""

    for [platform, meta] in KUBECTL_PLATFORMS.items():
        build_content += """
toolchain(
    name = "{platform}_toolchain",
    exec_compatible_with = {compatible_with},
    toolchain = "@{user_repository_name}_{platform}//:kubectl_toolchain",
    toolchain_type = "@rules_k8s_cd//lib:kubectl_toolchain_type",
)
""".format(
            platform = platform,
            user_repository_name = rctx.attr.user_repository_name,
            compatible_with = meta.compatible_with,
        )

    # Base BUILD file for this repository
    rctx.file("BUILD.bazel", build_content)

kubectl_toolchains_repo = repository_rule(
    _kubectl_toolchains_repo_impl,
    doc = """Creates a repository with toolchain definitions for all known platforms
     which can be registered or selected.""",
    attrs = {
        "user_repository_name": attr.string(doc = "Base name for toolchains repository"),
    },
)

def _kubectl_platform_repo_impl(rctx):
    is_windows = rctx.attr.platform.startswith("windows_")
    meta = KUBECTL_PLATFORMS[rctx.attr.platform]
    release_platform = meta.release_platform if hasattr(meta, "release_platform") else rctx.attr.platform
    download_toolchain_binary(
        rctx = rctx,
        toolchain_name = "kubectl",
        platform = rctx.attr.platform,
        binary = _binaries[rctx.attr.version][rctx.attr.platform],
    )

kubectl_platform_repo = repository_rule(
    implementation = _kubectl_platform_repo_impl,
    doc = "Fetch external tools needed for kubectl toolchain",
    attrs = {
        "platform": attr.string(mandatory = True, values = KUBECTL_PLATFORMS.keys()),
        "version": attr.string(mandatory = False, default = DEFAULT_KUBECTL_VERSION, values = _binaries.keys()),
    },
)

def _kubectl_host_alias_repo(rctx):
    ext = ".exe" if repo_utils.is_windows(rctx) else ""

    # Base BUILD file for this repository
    rctx.file("BUILD.bazel", """# @generated by @rules_k8s_cd//lib/private:kubectl_toolchain.bzl
package(default_visibility = ["//visibility:public"])

exports_files(["kubectl{ext}"])
""".format(
        ext = ext,
    ))

    rctx.symlink("../{name}_{platform}/kubectl{ext}".format(
        name = rctx.attr.name,
        platform = repo_utils.platform(rctx),
        ext = ext,
    ), "kubectl{ext}".format(ext = ext))

kubectl_host_alias_repo = repository_rule(
    _kubectl_host_alias_repo,
    doc = """Creates a repository with a shorter name meant for the host platform, which contains
    a BUILD.bazel file that exports symlinks to the host platform's binaries
    """,
)
