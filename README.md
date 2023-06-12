# Bazel empty directory

This repository includes a Bazel empty directory for monorepo with gitops.

## Usage

```bash
# To see all the targets
bazel query //deployments/...
```

Result
```bash
//deployments/website:beta
//deployments/website:beta.apply
//deployments/website:beta.gitops
//deployments/website:beta.show
//deployments/website:dev
//deployments/website:dev.apply
//deployments/website:dev.delete
//deployments/website:dev.show
//deployments/website:docker.loopingz.com_bazel_website.push
//deployments/website:preview
//deployments/website:preview.apply
//deployments/website:preview.gitops
//deployments/website:preview.show
//deployments/website:prod
//deployments/website:prod.apply
//deployments/website:prod.gitops
//deployments/website:prod.show
//deployments/website:user
//deployments/website:user.apply
//deployments/website:user.delete
//deployments/website:user.show
```

