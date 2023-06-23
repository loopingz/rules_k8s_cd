namespace: {{namespace}}
patches:
    - patch: |-
        - op: add
          path: "/metadata/labels/gitops.loopingz.com~1commit"
          value: "{{commit}}"
        - op: add
          path: "/metadata/labels/gitops.loopingz.com~1environment"
          value: "{{environment}}"
        - op: add
          path: "/metadata/annotations/gitops.loopingz.com~1target"
          value: "{{source}}"
      target:
        name: ".*"