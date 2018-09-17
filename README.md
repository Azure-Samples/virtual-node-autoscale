# Virutal node Autoscale Demo

This repository demonstrates how to use custom metrics in combination with the Kubernetes Horizontal Pod Autoscaler to autoscale an application. Virtual nodes let you scale quickly and easily run Kubernetes pods on Azure Container Instances where you'll pay only for the container instance runtime. This repository will guide you through first installing the virtual node admission controller, followed by the Prometheus Operator. Then create a Prometheus instance and install the Prometheus Metric Adapter. With these in place, the provided Helm chart will install our demo application, along with supporting monitoring components, like a **ServiceMonitor** for Prometheus, a Horizontal Pod Autoscaler, and a custom container that will count the instances of the application and expose them to Prometheus. Finally, an optional Grafana dashboard can be installed to view the metrics in real-time.

## Prerequisites
* Virtual node enabled AKS cluster running Kubernetes version 1.10 or later
* Advanced Networking
* MutatingAdmissionWebhook admission controller activated - (https://kubernetes.io/docs/admin/extensible-admission-controllers/#external-admission-webhooks).
* Running Ingress controller (nginx or similar)

## Install Virtual node admission-controller (OPTIONAL)

In order to control the Kubernetes scheduler to "favor" VM backed Kubernetes nodes BEFORE scaling out to the virtual node we can use a Kubernetes Webhook Admission Controller to add pod affinity and toleration key/values to all pods in a correctly labeled namespace. 

### Pod patches

All pods on a correctly labelled namespace will be patched as follows:

#### Anti-affinity

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

#### Toleration

```yaml
spec:
  tolerations:
  - key: virtual-kubelet.io/provider
    operator: Exists
  - effect: NoSchedule
    key: azure.com/aci
```

### Install

```
helm install --name admission-webhook charts/vn-affinity-admission-controller --namespace vn-affinity
```

Label the namespace you wish enable the webhook to function on
```
kubectl label namespace default vn-affinity-injection=enabled
```

## Install Prometheus Operator

We will use the Prometheus Operator to create the Prometheus cluster and to create the relevant configuration to monitor our app. So first, install the operator:

```bash
kubectl apply -f https://raw.githubusercontent.com/coreos/prometheus-operator/master/bundle.yaml
```

This will create a number of CRDs, Service Accounts and RBAC things. Once it's finished, you'll have the Prometheus Operator, but not a Prometheus instance. You'll need to create one of those next. 

## Create a Prometheus instance

```bash
kubectl apply -f prometheus-config/prometheus
```

This will create a single replica Prometheus instance.

## Expose a Service for Prometheus instance

```bash
kubectl expose pod prometheus-prometheus-0 --port 9090 --target-port 9090
```

## Deploy The Adopt Dog App

You will need your Virtual Kubelet node name to install the app. The app will install a counter that will get the pod count for the application and provide a metric for pods on Virtual Kubelet and pods on all other nodes.

```bash
$ kubectl get nodes
AME                       STATUS    ROLES     AGE       VERSION
aks-nodepool1-30440750-0   Ready     agent     27d       v1.10.6
aks-nodepool1-30440750-1   Ready     agent     27d       v1.10.6
aks-nodepool1-30440750-2   Ready     agent     27d       v1.10.6
virtual-kubelet            Ready     agent     16h       v1.8.3
```
In this case, it's Virtual Kubelet. If you've installed with the ACI Connector, you may have a node name like **virtual-kubelet-aci-connector-linux-westcentralus**.

Export the node name to an environment variable

```base
export VK_NODE_NAME=<your_node_name>
```

```bash
helm install ./charts/adoptdog --name rps-prom --set counter.specialNodeName=$VK_NODE_NAME
```

This will deploy with an ingress and should create the HPA, Prometheus ServiceMonitor and everything else needed, except the adapter. Do that next.

Change the values.yaml as needed (especially for ingress)

## Deploy the Prometheus Metric Adapter

```bash
helm install stable/prometheus-adapter --name prometheus-adaptor -f prometheus-config/prometheus-adapter/values.yaml
```

NOTE: if you have the Azure application insights adapter installed, you'll need to remove that first.

There might be some lag time between when you create the adapter and when the metrics are available.

```bash
kubectl get --raw /apis/custom.metrics.k8s.io/v1beta1/namespaces/default/pod/*/requests_per_second | jq .
```

This should show metrics if everything is setup correctly. Example:

```console
$ kubectl get --raw /apis/custom.metrics.k8s.io/v1beta1/namespaces/default/pod/*/requests_per_second | jq .
{
  "kind": "MetricValueList",
  "apiVersion": "custom.metrics.k8s.io/v1beta1",
  "metadata": {
    "selfLink": "/apis/custom.metrics.k8s.io/v1beta1/namespaces/default/pod/%2A/requests_per_second"
  },
  "items": [
    {
      "describedObject": {
        "kind": "Pod",
        "namespace": "default",
        "name": "rps-prom-adoptdog-8684976576-7hvc9",
        "apiVersion": "/__internal"
      },
      "metricName": "requests_per_second",
      "timestamp": "2018-09-05T03:57:44Z",
      "value": "0"
    },
    {
      "describedObject": {
        "kind": "Pod",
        "namespace": "default",
        "name": "rps-prom-adoptdog-8684976576-p6wm7",
        "apiVersion": "/__internal"
      },
      "metricName": "requests_per_second",
      "timestamp": "2018-09-05T03:57:44Z",
      "value": "0"
    }
  ]
}
```

## Deploy the Grafana Dashboard (OPTIONAL)

This optional step installs a Grafana dashboard to view measured metrics in real-time.

```bash
helm install stable/grafana --name grafana -f grafana/values.yaml
```

```bash
export POD_NAME=$(kubectl get pods --namespace default -l "app=grafana,component=" -o jsonpath="{.items[0].metadata.name}")
kubectl --namespace default port-forward $POD_NAME 3000
```

Browse to localhost:3000 and login

## Hit it with some Load

I've been using [Hey](https://github.com/rakyll/hey)

```
export GOPATH=~/go
export PATH=$GOPATH/bin:$PATH
go get -u github.com/rakyll/hey
hey -z 20m http://<whatever-the-ingress-url-is>
```

## Watch it scale

```
$ kubectl get hpa rps-prom-adoptdog -w
NAME                REFERENCE                      TARGETS   MINPODS   MAXPODS   REPLICAS   AGE
rps-prom-adoptdog   Deployment/rps-prom-adoptdog   0 / 10    2         10        2          4m
rps-prom-adoptdog   Deployment/rps-prom-adoptdog   128500m / 10   2         10        2         4m
rps-prom-adoptdog   Deployment/rps-prom-adoptdog   170500m / 10   2         10        4         5m
rps-prom-adoptdog   Deployment/rps-prom-adoptdog   111500m / 10   2         10        4         5m
rps-prom-adoptdog   Deployment/rps-prom-adoptdog   95250m / 10   2         10        4         6m
rps-prom-adoptdog   Deployment/rps-prom-adoptdog   141 / 10   2         10        4         6m
```

Overtime, this should go up.

## Some Prometheus Queries

Rounded average requests per second per container

```
round(avg(irate(request_durations_histogram_secs_count[1m])))
```

Rounded total requests per second

```
round(sum(irate(request_durations_histogram_secs_count[1m])))
```

Individual number of data points, should correspond to number of containers

```
count(irate(request_durations_histogram_secs_count[1m]))
```

The number of containers running, per the counter

```
running_containers
```

Average response time in seconds:

```
avg(request_durations_histogram_secs_sum / request_durations_histogram_secs_count)
```
