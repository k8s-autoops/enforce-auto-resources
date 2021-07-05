# template-autoops-admission

## 使用方式

* 初始化 `admission-bootstrapper`
  参照此文档 https://github.com/k8s-autoops/admission-bootstrapper ，完成 `admission-bootstrapper` 的初始化步骤
* 部署以下 YAML

```yaml
# create serviceaccount
apiVersion: v1
kind: ServiceAccount
metadata:
  name: template-autoops-admission
  namespace: autoops
---
# create clusterrole
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: template-autoops-admission
rules:
  - apiGroups: [ "" ]
    resources: [ "namespaces" ]
    verbs: [ "get" ]
---
# create clusterrolebinding
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: template-autoops-admission
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: template-autoops-admission
subjects:
  - kind: ServiceAccount
    name: template-autoops-admission
    namespace: autoops
---
# create job
apiVersion: batch/v1
kind: Job
metadata:
  name: install-template-autoops-admission
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
              value: template-autoops-admission
            - name: ADMISSION_IMAGE
              value: autoops/template-autoops-admission
            - name: ADMISSION_ENVS
              value: ""
            - name: ADMISSION_SERVICE_ACCOUNT
              value: "template-autoops-admission"
            - name: ADMISSION_MUTATING
              value: "true"
            - name: ADMISSION_IGNORE_FAILURE
              value: "false"
            - name: ADMISSION_SIDE_EFFECT
              value: "None"
            - name: ADMISSION_RULES
              value: '[{"operations":["CREATE"],"apiGroups":[""], "apiVersions":["*"], "resources":["services"]}]'
      restartPolicy: OnFailure
```

## Credits

Guo Y.K., MIT License
