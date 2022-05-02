# PVC-cleaner
## A configurable Persistant Volume Claim cleaner for Kubernetes 

This is a CronJob that looks for PVCs that match the configured filters, and deletes them.

To use this, download the [pvc-cleaner.yaml](pvc-cleaner.yaml) file, and configure the filters in the confgMap.
All fields are optional, and can be left empty (""), but the result can be undesirable.

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: pvc-cleaner-config
  namespace: kube-system
data:
  config.yaml: |
    namespace: "my-namespace"       <- set this to chose a namespace (all by default)
    prefixFilter: "my-prefix"       <- set this to use a prefix filter
    sufixFilter: ""                 <- set this to use a sufix filter
...
```

How the filters work: 
- prefixFilter :  This app will try to delete PVCs with name that start with "my-prefix" 
- sufixFilter :  This app will try to delete PVCs with name that end with "my-sufix"
- namespace : This makes sure only the chosen namespace will be used
