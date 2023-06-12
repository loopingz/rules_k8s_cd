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

When you deploy the main resources have labels:

Like this service:
```
apiVersion: v1
kind: Service
metadata:
  annotations:
    gitops.loopingz.com/target: deployments/website:preview
  labels:
    app: website
    gitops.loopingz.com/commit: 09f7b66
    gitops.loopingz.com/environment: preview
```

It is then easy to clean-up with a command like this:

```
kubectl -n bazel-preview delete all -l 'gitops.loopingz.com/commit,gitops.loopingz.com/commit!=09f7b66,gitops.loopingz.com/environment=preview'
```

