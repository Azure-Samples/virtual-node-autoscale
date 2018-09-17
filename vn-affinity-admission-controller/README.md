# Kubernetes Virtual node Admission Webhook

This Kubernetes Admission adds pod affinity and toleration key/values to all pods in a correctly labeled namespace

## Project State

Experimental

## Pod patches

All pods on a correctly labelled namespace will be patched as follows:

### Anti-affinity

```yaml
spec:
  affinity:
    nodeAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - preference:
          matchExpressions:
          - key: type
            operator: NotIn
            values:
            - virtual-kubelet
```

### Toleration

```yaml
spec:
  tolerations:
  - key: virtual-kubelet.io/provider
    operator: Exists
  - effect: NoSchedule
    key: azure.com/aci
```

## Attribution

This projects uses the upstream examples found in the following repos:
* https://github.com/caesarxuchao/example-webhook-admission-controller
* https://github.com/kubernetes/kubernetes/tree/release-1.9/test/images/webhook

Massive thanks for all the work that went into crafting reusable examples.

## Supported Kubernetes versions

* 1.10
* 1.11


## Prerequisites
Please enable the admission webhook feature
[doc](https://kubernetes.io/docs/admin/extensible-admission-controllers/#enable-external-admission-webhooks).

## Build

```bash
make docker_build
```

## Deploy

Enable the relevant Kubernetes Admission controller by adding to following `--admission-control` and restarting kube-apiserver. See the relevant [docs](https://kubernetes.io/docs/admin/extensible-admission-controllers/#external-admission-webhooks).
```
MutatingAdmissionWebhook
```

```
helm install --name admission-webhook charts/vn-affinity-admission-controller --namespace vn-affinity
```

```
helm inspect charts/vn-affinity-admission-controller
```

Label the namespace you wish enable the webhook to function on
```
kubectl label namespace default vk-affinity-injection=enabled
```
