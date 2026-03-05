load("@aspect_bazel_lib//lib/private:repo_utils.bzl", "repo_utils")
load("//lib:repo_utils.bzl", "download_toolchain_binary")

# version=https://dl.k8s.io/release/stable.txt
# https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl https://dl.k8s.io/release/${version}/bin/darwin/arm64/kubectl.sha256
# https://dl.k8s.io/release/${version}/bin/darwin/amd64/kubectl https://dl.k8s.io/release/${version}/bin/darwin/amd64/kubectl.sha256
# https://dl.k8s.io/release/${version}/bin/linux/arm64/kubectl https://dl.k8s.io/release/${version}/bin/linux/arm64/kubectl.sha256
# https://dl.k8s.io/release/${version}/bin/linux/amd64/kubectl https://dl.k8s.io/release/${version}/bin/linux/amd64/kubectl.sha256

_binaries = {
    "1.35.2": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.35.2/bin/darwin/amd64/kubectl", "163955964d4ed9e66656eab45c0114f5c1110d1b430ace432b20ddc430023df5"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.35.2/bin/darwin/arm64/kubectl", "b0b59cdd7ba20ca20b85214943100e578dd50ddd85242fcddf277a87c2249706"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.35.2/bin/linux/amd64/kubectl", "924eb50779153f20cb668117d141440b95df2f325a64452d78dff9469145e277"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.35.2/bin/linux/arm64/kubectl", "cd859449f54ad2cb05b491c490c13bb836cdd0886ae013c0aed3dd67ff747467"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.35.2/bin/windows/amd64/kubectl.exe", "a6d02010aae330a3031afe7843747d5a18db59c9f0bca415e45fb24bbdccdd2c"),
    },
    "1.35.1": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.35.1/bin/darwin/amd64/kubectl", "07a04d82bc2de2f5d53dfd81f2109ca864f634a82b225257daa2f9c2db15ccef"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.35.1/bin/darwin/arm64/kubectl", "2b000dded317319b1ebca19c2bc70f772c7aaa0e8962fae2d987ba04dd1a1b50"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.35.1/bin/linux/amd64/kubectl", "36e2f4ac66259232341dd7866952d64a958846470f6a9a6a813b9117bd965207"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.35.1/bin/linux/arm64/kubectl", "706256e21a4e9192ee62d1a007ac0bfcff2b0b26e92cc7baad487a6a5d08ff82"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.35.1/bin/windows/amd64/kubectl.exe", "d2d28ca3440ed94262b9b4bffb30119cfca69ac30b9cad531a08e8ebd9720dd2"),
    },
    "1.34.5": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.34.5/bin/darwin/amd64/kubectl", "61ccfd05992fcc1135818d25691e55997155e914c98b704df7ddd542339a93cb"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.34.5/bin/darwin/arm64/kubectl", "0c0b575db594e0f8842aa44e5d36b224bb9f8bf4b92cd34c0547d29b61a0277f"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.34.5/bin/linux/amd64/kubectl", "6a17dd8387783b3144a65535e38d02c351027e9718ea34a6c360476cb26d28bb"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.34.5/bin/linux/arm64/kubectl", "2d433b53b99ea532f877df6fa5044286e3950d4933967ac3d99262760bc649fd"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.34.5/bin/windows/amd64/kubectl.exe", "ecaabd85663e2beece62c53bcf4ad999e1959f60abd69188837e1a22d98604eb"),
    },
    "1.34.4": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.34.4/bin/darwin/amd64/kubectl", "9861e775578651d903a779f840ff8e4c0013ae29a1ffc67f05d967262fcf734c"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.34.4/bin/darwin/arm64/kubectl", "d9b239414a5fdf624835bfb62eef28a0885f1645d579f98056a9ab3293f51f87"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.34.4/bin/linux/amd64/kubectl", "d50c359d95e0841eaad08ddc27c7be37cba8fdccfba5c8e2ded65e121ff112db"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.34.4/bin/linux/arm64/kubectl", "5b982c0644ab1e27780246b9085a5886651b4a7ed86243acbb2bacc1bea01dda"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.34.4/bin/windows/amd64/kubectl.exe", "cbb29e9943d6afcc23a39ec62970495230637ebe6c26df063dcd62b19bea7f85"),
    },
    "1.34.3": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.34.3/bin/darwin/amd64/kubectl", "657afbd0e653c4ce3af1b5a645a4eaba282cf8eb2bcda7191ff60866e50e4d7f"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.34.3/bin/darwin/arm64/kubectl", "e51367d2107d605f4edd7c2fb25897b0c0695a7de1a9f9d04cd6c9356b890b14"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.34.3/bin/linux/amd64/kubectl", "ab60ca5f0fd60c1eb81b52909e67060e3ba0bd27e55a8ac147cbc2172ff14212"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.34.3/bin/linux/arm64/kubectl", "46913a7aa0327f6cc2e1cc2775d53c4a2af5e52f7fd8dacbfbfd098e757f19e9"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.34.3/bin/windows/amd64/kubectl.exe", "5ef6e0b019cfea5b0eff55b576c0118f64c0758a8bcbf52587c7f454f302f7bc"),
    },
    "1.34.2": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.34.2/bin/darwin/amd64/kubectl", "d2a71bb7dd7238287f2ba4efefbad4f98584170063f7d9e6c842f772d9255d45"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.34.2/bin/darwin/arm64/kubectl", "8f38d3a38ae317b00ebf90254dc274dd28d8c6eea4a4b30c5cb12d3d27017b6d"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.34.2/bin/linux/amd64/kubectl", "9591f3d75e1581f3f7392e6ad119aab2f28ae7d6c6e083dc5d22469667f27253"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.34.2/bin/linux/arm64/kubectl", "95df604e914941f3172a93fa8feeb1a1a50f4011dfbe0c01e01b660afc8f9b85"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.34.2/bin/windows/amd64/kubectl.exe", "7d34dcc49a185d64194ff3e952d5621b7da4f5562fa83df5acf305bd1f7de9cc"),
    },
    "1.34.1": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.34.1/bin/darwin/amd64/kubectl", "bb211f2b31f2b3bc60562b44cc1e3b712a16a98e9072968ba255beb04cefcfdf"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.34.1/bin/darwin/arm64/kubectl", "d80e5fa36f2b14005e5bb35d3a72818acb1aea9a081af05340a000e5fbdb2f76"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.34.1/bin/linux/amd64/kubectl", "7721f265e18709862655affba5343e85e1980639395d5754473dafaadcaa69e3"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.34.1/bin/linux/arm64/kubectl", "420e6110e3ba7ee5a3927b5af868d18df17aae36b720529ffa4e9e945aa95450"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.34.1/bin/windows/amd64/kubectl.exe", "d118a8ddb0de15ff230189c85f5157e752405eb0ae8fa680d284de094c9a20f0"),
    },
    "1.34.0": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.34.0/bin/darwin/amd64/kubectl", "a5904061dd5c8e57d55e52c78fa23790e76de30924b26ba31be891e75710d7a9"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.34.0/bin/darwin/arm64/kubectl", "d491f4c47c34856188d38e87a27866bd94a66a57b8db3093a82ae43baf3bb20d"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.34.0/bin/linux/amd64/kubectl", "cfda68cba5848bc3b6c6135ae2f20ba2c78de20059f68789c090166d6abc3e2c"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.34.0/bin/linux/arm64/kubectl", "00b182d103a8a73da7a4d11e7526d0543dcf352f06cc63a1fde25ce9243f49a0"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.34.0/bin/windows/amd64/kubectl.exe", "856b6a92556452e249db940e7fdb8d8f8f622805d25f67de09a4d4d2da6f6132"),
    },
    "1.33.9": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.33.9/bin/darwin/amd64/kubectl", "26b2eafcfafaf9fae8d3a5e280571ebdbf46d5f90dffe43e71fd6c721a1fd372"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.33.9/bin/darwin/arm64/kubectl", "9699c9369a2ed65e73e9721b14d88ac14be27858c558dfb6e813503b93cd7056"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.33.9/bin/linux/amd64/kubectl", "9e33e3234c0842cd44a12c13e334b4ce930145ea84b855ce7cc0a7b6bc670c22"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.33.9/bin/linux/arm64/kubectl", "af4dc943a6f447ecb070340efe63c7f8ee2808e6c0bc42126efe7cde0cc1e69b"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.33.9/bin/windows/amd64/kubectl.exe", "e0af0b84fb9323c1e9f705bcd7071476fcd117d1f252270059e194920c57bf71"),
    },
    "1.33.8": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.33.8/bin/darwin/amd64/kubectl", "713ceda90ca7d0de8ce70f74f6613064a74a47a7a8b2cffe3785f9b3f4a61ecc"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.33.8/bin/darwin/arm64/kubectl", "a43be01c74267e2748ddfdeaa33eeda6dbe073b6f369df6b3ffcfdc6843f7ad9"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.33.8/bin/linux/amd64/kubectl", "7f9c3faab7c9f9cc3f318d49eb88efc60eb3b3a7ce9eee5feb39b1280e108a29"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.33.8/bin/linux/arm64/kubectl", "76e284669f1f6343bd9fe2a011757809c8c01cf51da9f85ee6ef4eb93c8393a8"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.33.8/bin/windows/amd64/kubectl.exe", "3d1d28a143623941972142e43ab827b6161c489bac9d3d38f209eda43bd5da4d"),
    },
    "1.33.7": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.33.7/bin/darwin/amd64/kubectl", "45be3f5293da84d97e86580a541b247fe3cec60196fdd6abd2b811d7dd4d3f1b"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.33.7/bin/darwin/arm64/kubectl", "2e333f56d115081af83a48b5f31a91fb32852550f8117a0a31cf8bae2e601704"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.33.7/bin/linux/amd64/kubectl", "471d94e208a89be62eb776700fc8206cbef11116a8de2dc06fc0086b0015375b"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.33.7/bin/linux/arm64/kubectl", "fa7ee98fdb6fba92ae05b5e0cde0abd5972b2d9a4a084f7052a1fd0dce6bc1de"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.33.7/bin/windows/amd64/kubectl.exe", "df8bead144a7a997a79c480083061955ddcd171f803631ac1239c0bd6a8f36ac"),
    },
    "1.33.6": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.33.6/bin/darwin/amd64/kubectl", "a0f485c2b8296c84fda606dd585e2458a06a41235f1e96348cf64a2f527f6e77"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.33.6/bin/darwin/arm64/kubectl", "ba6e00a0479d45a4aa59ad550ed0fd68696e73bd2d43d0e00213fba41f61fa54"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.33.6/bin/linux/amd64/kubectl", "d25d9b63335c038333bed785e9c6c4b0e41d791a09cac5f3e8df9862c684afbe"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.33.6/bin/linux/arm64/kubectl", "3ab32d945a67a6000ba332bf16382fc3646271da6b7d751608b320819e5b8f38"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.33.6/bin/windows/amd64/kubectl.exe", "bc2e96179cce21fa3ef6e216fe853f41d08850f73f61a150d1485ee18a16acea"),
    },
    "1.33.5": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.33.5/bin/darwin/amd64/kubectl", "ebdefb65c60c920510a605f13622e7eadb85bb83ba393d9eed2389bac30672b1"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.33.5/bin/darwin/arm64/kubectl", "22f7256932c1c5205d7323a63d16253b8405ecccfd57c7a2484d3219c6822d3e"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.33.5/bin/linux/amd64/kubectl", "6a12d6c39e4a611a3687ee24d8c733961bb4bae1ae975f5204400c0a6930c6fc"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.33.5/bin/linux/arm64/kubectl", "6db7c5d846c3b3ddfd39f3137a93fe96af3938860eefdbf2429805ee1656e381"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.33.5/bin/windows/amd64/kubectl.exe", "2fa5d21aa99afe994b1e929054d4ca701f7dd5e8124f8f1c83d28186474bc00b"),
    },
    "1.33.4": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.33.4/bin/darwin/amd64/kubectl", "4b39b8bb12e78ce801b39c9ec50421e3d6e144d8e3f113cd18e6d61709b8c73b"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.33.4/bin/darwin/arm64/kubectl", "a44662db083fdd1b19ce55ba77eb64d51206310bbae90df90eb5d9e30ea54603"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.33.4/bin/linux/amd64/kubectl", "c2ba72c115d524b72aaee9aab8df8b876e1596889d2f3f27d68405262ce86ca1"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.33.4/bin/linux/arm64/kubectl", "76cd7a2aa59571519b68c3943521404cbce55dafb7d8866f8d0ea2995b396eef"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.33.4/bin/windows/amd64/kubectl.exe", "15487c2a017af8ef8a5fbc2390af78b90f98d2909f23eeb684c64a8af3f7c4eb"),
    },
    "1.33.3": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.33.3/bin/darwin/amd64/kubectl", "9652b55a58e84454196a7b9009f6d990d3961e2bd4bd03f64111d959282b46b1"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.33.3/bin/darwin/arm64/kubectl", "3de173356753bacb215e6dc7333f896b7f6ab70479362146c6acca6e608b3f53"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.33.3/bin/linux/amd64/kubectl", "2fcf65c64f352742dc253a25a7c95617c2aba79843d1b74e585c69fe4884afb0"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.33.3/bin/linux/arm64/kubectl", "3d514dbae5dc8c09f773df0ef0f5d449dfad05b3aca5c96b13565f886df345fd"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.33.3/bin/windows/amd64/kubectl.exe", "fbcb21ae1f8e0313ca44c9a3392f62523caf8c1a23b49c80e01cbf541060d592"),
    },
    "1.33.2": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.33.2/bin/darwin/amd64/kubectl", "ff468749bd3b5f4f15ad36f2a437e65fcd3195a2081925140334429eaced1a8a"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.33.2/bin/darwin/arm64/kubectl", "8730bf6dab538a1e9710a3668e2cd5f1bdc3c25c68b65a57c5418bdc3472769c"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.33.2/bin/linux/amd64/kubectl", "33d0cdec6967817468f0a4a90f537dfef394dcf815d91966ca651cc118393eea"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.33.2/bin/linux/arm64/kubectl", "54dc02c8365596eaa2b576fae4e3ac521db9130e26912385e1e431d156f8344d"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.33.2/bin/windows/amd64/kubectl.exe", "c45a0fb477262eebd4a4a2936ea6bd10ce6a7db8f1356cff6e703c948538c76b"),
    },
    "1.33.1": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.33.1/bin/darwin/amd64/kubectl", "8d36a5c66142547ad16e332942fd16a0ca2b3346d9ebaab6c348de2c70d9d875"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.33.1/bin/darwin/arm64/kubectl", "8ae6823839993bb2e394c3cf1919748e530642c625dc9100159595301f53bdeb"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.33.1/bin/linux/amd64/kubectl", "5de4e9f2266738fd112b721265a0c1cd7f4e5208b670f811861f699474a100a3"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.33.1/bin/linux/arm64/kubectl", "d595d1a26b7444e0beb122e25750ee4524e74414bbde070b672b423139295ce6"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.33.1/bin/windows/amd64/kubectl.exe", "815c3c39984d1f7347486ad58b8e33e61ee87bc8ad79e0dbc9793e22200614fb"),
    },
    "1.32.13": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.32.13/bin/darwin/amd64/kubectl", "925e89bdf7e30d9a356625b3b368b93261300cdee6c053935eb50ed9e657e769"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.32.13/bin/darwin/arm64/kubectl", "97c78912825a83e91c67fdae588a5a26dec53d58370210c78612dec04a71fc21"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.32.13/bin/linux/amd64/kubectl", "db2ae479a63f3665d7f704ab18c0d4d4050144237980763221835b7305703c4c"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.32.13/bin/linux/arm64/kubectl", "b1f87f196633a89208546d79bfa4e2470bda70e7bf42c4d3adb008ec208da9d1"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.32.13/bin/windows/amd64/kubectl.exe", "3dbef017631b5052a2f2f17246295afd137925687f5bf9c4cb7ca90ff45e4b37"),
    },
    "1.32.12": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.32.12/bin/darwin/amd64/kubectl", "784afed9df776945578dc3be1fc1fa4154badbee5699c2954bd0643315ce00aa"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.32.12/bin/darwin/arm64/kubectl", "2e59daf8dc865e68ce61a0e00a97ac0fb900fc250c6c25355f7592c3d6cf8ab7"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.32.12/bin/linux/amd64/kubectl", "adac5674d19f47ed3f0620bfeb1932d3d5f4557d49f311f6f65b6b75e0721ee1"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.32.12/bin/linux/arm64/kubectl", "86fa465134c54de6202d89d41eb89504810236cc968e1402a31f3674f66ffdbd"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.32.12/bin/windows/amd64/kubectl.exe", "f69dfdcbbb3a51a3cc76715314cebec782bcc8f52b516f8e024041a299ec3b93"),
    },
    "1.32.11": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.32.11/bin/darwin/amd64/kubectl", "8d0b610df71632d0e9b9c1aa16dde5ec666c05bf24e401ecf20fd27af16879ad"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.32.11/bin/darwin/arm64/kubectl", "a39978a062f0df17d4a5551bd2e3a91eda90039196653935c50140be547141d3"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.32.11/bin/linux/amd64/kubectl", "48581d0e808bd8b7d3c3fc014e86b170e25a987df04c8a879b982b28a5180815"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.32.11/bin/linux/arm64/kubectl", "b1c91c106ec20e61c5dff869e9a39e6af4fb96572bddaac9cce307dfa3ed2348"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.32.11/bin/windows/amd64/kubectl.exe", "8c350738ff800c42e4a11b026f73a656e09213a230a91b9a5646ea3a177edff3"),
    },
    "1.32.10": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.32.10/bin/darwin/amd64/kubectl", "626b52743531779981e7800aaac53a9cf4fc9c0266311c33faaa3854617f6129"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.32.10/bin/darwin/arm64/kubectl", "e6f7871732d5d80eb3987be13b986c9f8210f0f11f9b8b731330ba6c089056e0"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.32.10/bin/linux/amd64/kubectl", "6e14ef4e509e9f3d1dfc2815643f832f853d2d9f6622d4a0f83f77c7e4014b57"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.32.10/bin/linux/arm64/kubectl", "1f4229526e16bf9f5b854fbf3bdb9c7040404a29c1d1e4193258b8a73de06e92"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.32.10/bin/windows/amd64/kubectl.exe", "b7a550dad8945c7c5fa2c86951cb517c90bf9a64f44cc153cfd1b7139dcd1a8e"),
    },
    "1.32.9": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.32.9/bin/darwin/amd64/kubectl", "fb7e76a98ee3923615e0e98e42105c7b77ca80c2310b977f56784515190c1941"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.32.9/bin/darwin/arm64/kubectl", "8735038bb808e3c0acd5c553573f4ef2ac6a9ff508e077d46aa5b86b163bf7d2"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.32.9/bin/linux/amd64/kubectl", "509ae171bac7ad3b98cc49f5594d6bc84900cf6860f155968d1059fde3be5286"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.32.9/bin/linux/arm64/kubectl", "d5f6b45ad81b7d199187a28589e65f83406e0610b036491a9abaa49bfd04a708"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.32.9/bin/windows/amd64/kubectl.exe", "730b26050e1395b5ba4dfc4cb6e84b8d79d01e2ca3e95328fb14ef296c30ab58"),
    },
    "1.32.8": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.32.8/bin/darwin/amd64/kubectl", "a00a8fadd4a7ca520e68e88a640ca60b4601695f68b8dcde33293ed709c8c807"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.32.8/bin/darwin/arm64/kubectl", "01e5c58a305f309bd4f268125ba8a9c138a20ca9d602c74cd6b37a0d45fc5818"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.32.8/bin/linux/amd64/kubectl", "0fc709a8262be523293a18965771fedfba7466eda7ab4337feaa5c028aa46b1b"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.32.8/bin/linux/arm64/kubectl", "8a7371e54187249389a9aa222b150d61a4a745c121ab24dbcbb56d1ac2d0b912"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.32.8/bin/windows/amd64/kubectl.exe", "aa291c1e09267e193bb58cd6533b1824ca11ed0e56ca0869f614c6181d8a4bf2"),
    },
    "1.32.7": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.32.7/bin/darwin/amd64/kubectl", "050a5b4227a07c6d7f5add1863323f9db90b97c12874e2218224c9be74286980"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.32.7/bin/darwin/arm64/kubectl", "07a3511f02763076859e37abae33e1513285feec0482798a547441128f84662b"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.32.7/bin/linux/amd64/kubectl", "b8f24d467a8963354b028796a85904824d636132bef00988394cadacffe959c9"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.32.7/bin/linux/arm64/kubectl", "232f6e517633fbb4696c9eb7a0431ee14b3fccbb47360b4843d451e0d8c9a3a2"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.32.7/bin/windows/amd64/kubectl.exe", "06468f371634191e8a3d7ec63c463dbf81a27518a9f87309153e67d760f94eff"),
    },
    "1.32.6": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.32.6/bin/darwin/amd64/kubectl", "ad0c1880b1bcd36869d75a54c3401b718c091d75d11d08f57034fb7b4712f6ef"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.32.6/bin/darwin/arm64/kubectl", "8ac847473a6794dd35d2b980c9249b79dedb6e234d00fd0f223cf6b67be12999"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.32.6/bin/linux/amd64/kubectl", "0e31ebf882578b50e50fe6c43e3a0e3db61f6a41c9cded46485bc74d03d576eb"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.32.6/bin/linux/arm64/kubectl", "f7bac84f8c35f55fb2c6ad167beb59eba93de5924b50bbaa482caa14ff480eec"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.32.6/bin/windows/amd64/kubectl.exe", "3b4aabd90c52e01557f08cb2747431a78767cf978646812a69d8d53d73f7049e"),
    },
    "1.32.5": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.32.5/bin/darwin/amd64/kubectl", "f357d30fc338eb914e6e7a5e0408852d3011fac18d98f4484c4861c4c2cead3c"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.32.5/bin/darwin/arm64/kubectl", "b3b08783545e735b030376627133ddf53dc0e2c2ed4c413d87d4bcd7c2b0c632"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.32.5/bin/linux/amd64/kubectl", "aaa7e6ff3bd28c262f2d95c8c967597e097b092e9b79bcb37de699e7488e3e7b"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.32.5/bin/linux/arm64/kubectl", "9edee84103e63c40a37cd15bd11e04e7835f65cb3ff5a50972058ffc343b4d96"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.32.5/bin/windows/amd64/kubectl.exe", "df01c85015fa2b19fa7f92a7704aae9de5b5dc70fed32a01bd26f57f7ba563a5"),
    },
    "1.32.4": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.32.4/bin/darwin/amd64/kubectl", "061f65fe5405538f6fe8edd3c3373f479a1d59944ebf6268905535a617151d16"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.32.4/bin/darwin/arm64/kubectl", "01344900ac3c2c97a3290e9465d36f0dea20ca4533d226dfbe7c9a90e80ff9d4"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.32.4/bin/linux/amd64/kubectl", "10d739e9af8a59c9e7a730a2445916e04bc9cbb44bc79d22ce460cd329fa076c"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.32.4/bin/linux/arm64/kubectl", "c6f96d0468d6976224f5f0d81b65e1a63b47195022646be83e49d38389d572c2"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.32.4/bin/windows/amd64/kubectl.exe", "8e93d01f8efe80db614cf7dc422f9bb3fbad1b16f82d13f0ea70441441e486e4"),
    },
    "1.32.3": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.32.3/bin/darwin/amd64/kubectl", "b814c523071cd09e27c88d8c87c0e9b054ca0cf5c2b93baf3127750a4f194d5b"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.32.3/bin/darwin/arm64/kubectl", "a110af64fc31e2360dd0f18e4110430e6eedda1a64f96e9d89059740a7685bbd"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.32.3/bin/linux/amd64/kubectl", "ab209d0c5134b61486a0486585604a616a5bb2fc07df46d304b3c95817b2d79f"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.32.3/bin/linux/arm64/kubectl", "6c2c91e760efbf3fa111a5f0b99ba8975fb1c58bb3974eca88b6134bcf3717e2"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.32.3/bin/windows/amd64/kubectl.exe", "3fd1576a902ecf713f7d6390ae01799e370883e0341177ee09dbdc362db953e3"),
    },
    "1.32.2": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.32.2/bin/darwin/amd64/kubectl", "371b8fbd481e1e9052ace16d9c243e92618a2ea9a18c1aaf235d35fef20c0c32"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.32.2/bin/darwin/arm64/kubectl", "31b6318deaa72014b72121e1c7a2e12496d077cee49bbeda94250aec4c978ffb"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.32.2/bin/linux/amd64/kubectl", "4f6a959dcc5b702135f8354cc7109b542a2933c46b808b248a214c1f69f817ea"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.32.2/bin/linux/arm64/kubectl", "7381bea99c83c264100f324c2ca6e7e13738a73b8928477ac805991440a065cd"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.32.2/bin/windows/amd64/kubectl.exe", "cf51a1c6bf3b6ba6a5b549d1debf8aa6afb00c4c5a3d5d4bb1072f54cbe4390f"),
    },
    "1.32.1": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.32.1/bin/darwin/amd64/kubectl", "8bffe90f5a034d392a0ba6fd7ee16c0d40b1dba1ccc4350821102c5d5c56d846"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.32.1/bin/darwin/arm64/kubectl", "5b89f9598e2e7da04cc0b5dd6e8daca01d23855fd00c8ea259fd2aab993114db"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.32.1/bin/linux/amd64/kubectl", "e16c80f1a9f94db31063477eb9e61a2e24c1a4eee09ba776b029048f5369db0c"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.32.1/bin/linux/arm64/kubectl", "98206fd83a4fd17f013f8c61c33d0ae8ec3a7c53ec59ef3d6a0a9400862dc5b2"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.32.1/bin/windows/amd64/kubectl.exe", "b6378f34dcab2d411fb7a89ba700df0f784c0b063dd02dbb92396a72c4d3104e"),
    },
    "1.32.0": {
        "darwin_amd64": ("https://dl.k8s.io/release/v1.32.0/bin/darwin/amd64/kubectl", "516585916f499077fac8c2fdd2a382818683f831020277472e6bcf8d1a6f9be4"),
        "darwin_arm64": ("https://dl.k8s.io/release/v1.32.0/bin/darwin/arm64/kubectl", "5bfd5de53a054b4ef614c60748e28bf47441c7ed4db47ec3c19a3e2fa0eb5555"),
        "linux_amd64": ("https://dl.k8s.io/release/v1.32.0/bin/linux/amd64/kubectl", "646d58f6d98ee670a71d9cdffbf6625aeea2849d567f214bc43a35f8ccb7bf70"),
        "linux_arm64": ("https://dl.k8s.io/release/v1.32.0/bin/linux/arm64/kubectl", "ba4004f98f3d3a7b7d2954ff0a424caa2c2b06b78c17b1dccf2acc76a311a896"),
        "windows_amd64": ("https://dl.k8s.io/release/v1.32.0/bin/windows/amd64/kubectl.exe", "3601cb47c4d6a42b033a8f8fca68bc6f24baa99f5a1250fdb138d24a6c7cc749"),
    },
}

DEFAULT_KUBECTL_VERSION = "1.35.2"
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
