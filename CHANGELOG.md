# Changelog

## [1.5.2](https://github.com/loopingz/rules_k8s_cd/compare/v1.5.1...v1.5.2) (2024-01-22)


### Bug Fixes

* **deps:** update dependency bazel to v7.0.1 ([#84](https://github.com/loopingz/rules_k8s_cd/issues/84)) ([d724586](https://github.com/loopingz/rules_k8s_cd/commit/d72458692623b59cab3350544e7dcb0771cc2c8e))
* **deps:** update dependency io_bazel_rules_go to v0.45.1 ([#86](https://github.com/loopingz/rules_k8s_cd/issues/86)) ([d241127](https://github.com/loopingz/rules_k8s_cd/commit/d24112712fefca6cb77f81a8b98eeb922fd91117))

## [1.5.1](https://github.com/loopingz/rules_k8s_cd/compare/v1.5.0...v1.5.1) (2024-01-15)


### Bug Fixes

* **deps:** update nginx docker digest to 161ef4b ([#78](https://github.com/loopingz/rules_k8s_cd/issues/78)) ([87af021](https://github.com/loopingz/rules_k8s_cd/commit/87af021fc2126eca280f359fd25b8b0f264c04a2))

## [1.5.0](https://github.com/loopingz/rules_k8s_cd/compare/v1.4.3...v1.5.0) (2024-01-02)


### Features

* add secret/config generator ([f1a5078](https://github.com/loopingz/rules_k8s_cd/commit/f1a507890d048012807e97e88c165b848aa479a5))

## [1.4.3](https://github.com/loopingz/rules_k8s_cd/compare/v1.4.2...v1.4.3) (2023-12-23)


### Bug Fixes

* import from bazellib ([87877e7](https://github.com/loopingz/rules_k8s_cd/commit/87877e71892dc0fcc261300a06421fe9c9dd62b0))

## [1.4.2](https://github.com/loopingz/rules_k8s_cd/compare/v1.4.1...v1.4.2) (2023-12-23)


### Bug Fixes

* Update grype.bzl ([6fe3086](https://github.com/loopingz/rules_k8s_cd/commit/6fe3086dcb8c9418b9eddef243be3c6c69000b6a))

## [1.4.1](https://github.com/loopingz/rules_k8s_cd/compare/v1.4.0...v1.4.1) (2023-11-27)


### Bug Fixes

* **deps:** update dependencies ([378f825](https://github.com/loopingz/rules_k8s_cd/commit/378f82541ba42b07df457238b9f47b65cb5d91a1))

## [1.4.0](https://github.com/loopingz/rules_k8s_cd/compare/v1.3.0...v1.4.0) (2023-11-15)


### Features

* allow image direct references ([ade0eb0](https://github.com/loopingz/rules_k8s_cd/commit/ade0eb0ee4f08f98aa22829e4819a02e40f75c07))

## [1.3.0](https://github.com/loopingz/rules_k8s_cd/compare/v1.2.3...v1.3.0) (2023-11-12)


### Features

* allow to not provide a context ([dc18281](https://github.com/loopingz/rules_k8s_cd/commit/dc182812e1a21fad98735dceae44be59babddafd))


### Bug Fixes

* allow files for kubectl ([e882a4e](https://github.com/loopingz/rules_k8s_cd/commit/e882a4e03ec860899fce84ce0152d05925aede83))

## [1.2.3](https://github.com/loopingz/rules_k8s_cd/compare/v1.2.2...v1.2.3) (2023-10-13)


### Miscellaneous Chores

* **deps:** update actions/setup-node action to v3 ([#28](https://github.com/loopingz/rules_k8s_cd/issues/28)) ([96b708b](https://github.com/loopingz/rules_k8s_cd/commit/96b708b28854b741c39800ac460045529d465469))

## [1.2.2](https://github.com/loopingz/rules_k8s_cd/compare/v1.2.1...v1.2.2) (2023-10-12)


### Bug Fixes

* data files usage ([2a0c831](https://github.com/loopingz/rules_k8s_cd/commit/2a0c831960582cf0d4ffe5ec0fe3c9c198d9b535))

## [1.2.1](https://github.com/loopingz/rules_k8s_cd/compare/v1.2.0...v1.2.1) (2023-10-12)


### Bug Fixes

* kubectl data/context mismatch ([a12db26](https://github.com/loopingz/rules_k8s_cd/commit/a12db26a7e3fb11a2c93695edc0a63fe763b7ee9))

## [1.2.0](https://github.com/loopingz/rules_k8s_cd/compare/v1.1.0...v1.2.0) (2023-10-12)


### Features

* allow replacements in strategicMerge patch ([eaa070c](https://github.com/loopingz/rules_k8s_cd/commit/eaa070ccf3dcf109b6ab7a1b2aa6e578b933622d))


### Bug Fixes

* arm64 kubectl hash ([7f37d9c](https://github.com/loopingz/rules_k8s_cd/commit/7f37d9c15914b68c0e4a42cbd4adfe3bd7f93581))

## [1.1.0](https://github.com/loopingz/rules_k8s_cd/compare/v1.0.2...v1.1.0) (2023-10-03)


### Features

* add stamp replacement ([2487881](https://github.com/loopingz/rules_k8s_cd/commit/24878819e6b01601aaea3a28035c09d003901031))
* add substitution to kustomization_injector ([b6d3e33](https://github.com/loopingz/rules_k8s_cd/commit/b6d3e334fe10b205a2786eaf1b1be8cc0fcc22d2))
* add vars replacement in kustomization_injector ([1f202ff](https://github.com/loopingz/rules_k8s_cd/commit/1f202ff2ce75aa9465e784fd8d40b690aa1501ef))

## [1.0.2](https://github.com/loopingz/rules_k8s_cd/compare/v1.0.1...v1.0.2) (2023-09-30)


### Bug Fixes

* update grype to latest version ([4135209](https://github.com/loopingz/rules_k8s_cd/commit/4135209f203697c38d9186f9b40e6035fdb98f23))

## [1.0.1](https://github.com/loopingz/rules_k8s_cd/compare/v1.0.0...v1.0.1) (2023-06-26)


### Bug Fixes

* auto link context ([204a11d](https://github.com/loopingz/rules_k8s_cd/commit/204a11d89271bf3e53d915fe947f9b8bab4813d8))

## 1.0.0 (2023-06-24)


### Features

* add a kustomization_injector ([e94f98a](https://github.com/loopingz/rules_k8s_cd/commit/e94f98a110f1e7c71161cc8738b6913c2145a79a))
* add images injector and push for oci_image ([00bb41a](https://github.com/loopingz/rules_k8s_cd/commit/00bb41a9f61d21bd6832f76879401a98ad6d4476))
* add patches system ([eb47ff4](https://github.com/loopingz/rules_k8s_cd/commit/eb47ff440fa4f4f60befdf0e9b40785cc02a67d4))
* add show rule to help troubleshoot ([3464f0a](https://github.com/loopingz/rules_k8s_cd/commit/3464f0a1eae53e222e0c0a92e25d3e52a86a03f2))
* add write_source_files rule and kustomize macros ([4e0331a](https://github.com/loopingz/rules_k8s_cd/commit/4e0331a37d2ce8f6f887c39de57aae60d2317376))
* run_all rule and preparation for additional rules ([250bf45](https://github.com/loopingz/rules_k8s_cd/commit/250bf45fde359bf36b5469e7764303ae385c603a))


### Bug Fixes

* handle relative path and resource injection ([d5a7afa](https://github.com/loopingz/rules_k8s_cd/commit/d5a7afa33c650a46535ae9b5459c9afd6ed7360d))
* use Label to allow public rules ([125c7cf](https://github.com/loopingz/rules_k8s_cd/commit/125c7cf20d5d1827810bf1b0e2f6253ee8a0b221))
