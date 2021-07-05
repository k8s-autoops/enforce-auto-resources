# enforce-auto-resources

## 使用方式

* 初始化 `admission-bootstrapper`
  参照此文档 https://github.com/k8s-autoops/admission-bootstrapper ，完成 `admission-bootstrapper` 的初始化步骤
* 部署以下 YAML

```yaml
# create serviceaccount
apiVersion: v1
kind: ServiceAccount
metadata:
  name: enforce-auto-resources
  namespace: autoops
---
# create clusterrole
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: enforce-auto-resources
rules:
  - apiGroups: [ "apps" ]
    resources: [ "deployments", "replicasets" ]
    verbs: [ "get" ]
  - apiGroups: [ "" ]
    resources: [ "pods" ]
    verbs: [ "list" ]
  - apiGroups: [ "metrics.k8s.io" ]
    resources: [ "podmetrics" ]
    verbs: [ "get" ]
---
# create clusterrolebinding
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: enforce-auto-resources
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: enforce-auto-resources
subjects:
  - kind: ServiceAccount
    name: enforce-auto-resources
    namespace: autoops
---
# create job
apiVersion: batch/v1
kind: Job
metadata:
  name: install-enforce-auto-resources
  namespace: autoops
spec:
  template:
    spec:
      serviceAccount: admission-bootstrapper
      containers:
        - name: admission-bootstrapper
          image: autoops/admission-bootstrapper
          env:
            - name: ADMISSION_NAME
              value: enforce-auto-resources
            - name: ADMISSION_IMAGE
              value: autoops/enforce-auto-resources
            - name: ADMISSION_SERVICE_ACCOUNT
              value: "enforce-auto-resources"
            - name: ADMISSION_MUTATING
              value: "true"
            - name: ADMISSION_IGNORE_FAILURE
              value: "true"
            - name: ADMISSION_SIDE_EFFECT
              value: "None"
            - name: ADMISSION_RULES
              value: '[{"operations":["CREATE"],"apiGroups":[""], "apiVersions":["*"], "resources":["pods"]}]'
      restartPolicy: OnFailure
```

## Credits

Guo Y.K., MIT License
