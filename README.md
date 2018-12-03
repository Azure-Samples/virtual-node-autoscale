# Virtual node Autoscale Demo

This repository demonstrates how to use custom metrics in combination with the Kubernetes Horizontal Pod Autoscaler to autoscale an application. Virtual nodes let you scale quickly and easily run Kubernetes pods on Azure Container Instances where you'll pay only for the container instance runtime. 

This repository will guide you through first installing the virtual node admission controller, followed by the Prometheus Operator. Then create a Prometheus instance and install the Prometheus Metric Adapter. With these in place, the provided Helm chart will install our demo application, along with supporting monitoring components, like a **ServiceMonitor** for Prometheus, a Horizontal Pod Autoscaler, and a custom container that will count the instances of the application and expose them to Prometheus. Finally, an optional Grafana dashboard can be installed to view the metrics in real-time.

This demo was used at Microsoft Ignite 2018 Kenote. Check out the [video](https://mediastream.microsoft.com/events/2018/1809/Ignite/player/tracks/track2.html?start=17300).

## Prerequisites
* [Virtual node enabled AKS cluster](https://docs.microsoft.com/azure/aks/virtual-kubelet) running Kubernetes version 1.10 or later
* Setup AKS with Advanced Networking
* [MutatingAdmissionWebhook admission controller](https://kubernetes.io/docs/admin/extensible-admission-controllers/#external-admission-webhooks) activated
* Running Ingress controller (nginx or similar)

## Initialize Helm

Helm will be used to install the both the demo application and the supporting components. First ensure you have Helm [installed](https://docs.helm.sh/using_helm/#installing-helm). Once installed, you will need to initialize your cluster to use Helm. To do this, you will install the `Tiller` component with the `helm init` command.

```
kubectl -n kube-system create sa tiller
kubectl create clusterrolebinding tiller --clusterrole cluster-admin --serviceaccount=kube-system:tiller
helm init --service-account tiller
```

## Install Virtual Node admission-controller (OPTIONAL)

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
helm install --name vn-affinity ./charts/vn-affinity-admission-controller
```

Label the namespace you wish enable the webhook to function on
```
kubectl label namespace default vn-affinity-injection=enabled
```

## Install Prometheus Operator

We will use the [Prometheus Operator](https://coreos.com/operators/prometheus/docs/latest/user-guides/getting-started.html) to create the Prometheus cluster and to create the relevant configuration to monitor our app. So first, install the operator:

```bash
kubectl apply -f https://raw.githubusercontent.com/coreos/prometheus-operator/master/bundle.yaml
```

This will create a number of CRDs, Service Accounts and RBAC things. Once it's finished, you'll have the Prometheus Operator, but not a Prometheus instance. We will create one of those next. 

## Create a Prometheus instance

```bash
kubectl apply -f online-store/prometheus-config/prometheus
```

This will create a single replica Prometheus instance.

## Expose a Service for Prometheus instance

```bash
kubectl expose pod prometheus-prometheus-0 --port 9090 --target-port 9090
```

## Deploy online-store app

You will need your Virtual Kubelet node name, external IP address for Ingress, and decide if you wish to use App Insights to install the online-store app. 

### Export Virtual Kubelet node name
The app will also install a counter that will get the pod count for the application and provide a metric for pods on Virtual Kubelet and pods on all other nodes.

```bash
$ kubectl get nodes
NAME                       STATUS    ROLES     AGE       VERSION
aks-nodepool1-30440750-0   Ready     agent     27d       v1.10.6
aks-nodepool1-30440750-1   Ready     agent     27d       v1.10.6
aks-nodepool1-30440750-2   Ready     agent     27d       v1.10.6
virtual-kubelet            Ready     agent     16h       v1.8.3
```

In this case, it's Virtual Kubelet. If you've installed with the ACI Connector, you may have a node name like **virtual-kubelet-aci-connector-linux-westcentralus**.

Export the node name to an environment variable

```bash
export VK_NODE_NAME=<your_node_name>
```

### Export the ingress external IP address

Stated in the pre-requisites, an ingress solution must exist to accept requests from the sample application. The easiest way to set this up is by installing the [HTTP application routing add-on for AKS](https://docs.microsoft.com/azure/aks/http-application-routing).

This can be installed with the following add-on command.
```bash
az aks enable-addons --resource-group myResourceGroup --name myAKSCluster --addons http_application_routing
```

Once installed, export the external IP address of your ingress point by retrieving details of your kube-system.

```bash
kubectl get svc --all-namespaces
```

In this example output using NGINX for ingress, it is 104.40.223.193.

```bash
NAMESPACE     NAME                                                  TYPE           CLUSTER-IP     EXTERNAL-IP      PORT(S)                      AGE
default       kubernetes                                            ClusterIP      10.0.0.1       <none>      443/TCP                      4h
default       prometheus-operated                                   ClusterIP      None           <none>      9090/TCP                     21m
default       prometheus-prometheus-0                               ClusterIP      10.0.188.67    <none>      9090/TCP                     21m
kube-system   addon-http-application-routing-default-http-backend   ClusterIP      10.0.253.241   <none>      80/TCP                       4h
kube-system   addon-http-application-routing-nginx-ingress          LoadBalancer   10.0.91.10     104.40.223.193   80:31237/TCP,443:30963/TCP   4h
```

Export this external IP to an environment variable.

```bash
export INGRESS_EXTERNAL_IP=<ingress_external_ip>
```
### Set Application Insights on or off

By default, the online store will also send data to [Application Insights](https://docs.microsoft.com/en-us/azure/application-insights/app-insights-nodejs-quick-start#enable-application-insights). Once you have created a workspace, you'll need the [`Instrumentation Key`](https://docs.microsoft.com/en-us/azure/application-insights/app-insights-nodejs-quick-start#configure-app-insights-sdk). If you'd prefer to run without Application Insights, you can skip this step.

```bash
export APP_INSIGHT_KEY=<INSTRUMENTATION_KEY>
helm install ./charts/online-store --name online-store --set counter.specialNodeName=$VK_NODE_NAME,app.ingress.host=store.$INGRESS_EXTERNAL_IP.nip.io,appInsight.key=$APP_INSIGHT_KEY
```
To run this demo **without** Application Insights, run the command:

```bash
helm install ./charts/online-store --name online-store --set counter.specialNodeName=$VK_NODE_NAME,app.ingress.host=store.$INGRESS_EXTERNAL_IP.nip.io,appInsight.enabled=false
```

## Deploy the Prometheus Metric Adapter

NOTE: if you have the Azure application insights adapter installed, you'll need to remove that first.

```bash
helm install stable/prometheus-adapter --name prometheus-adaptor -f ./online-store/prometheus-config/prometheus-adapter/values.yaml
```

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
        "name": "online-store-8684976576-7hvc9",
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
        "name": "online-store-8684976576-p6wm7",
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

Retreive admin user password for UI access
```bash
kubectl get secret --namespace default grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo
```

Use kubectl to create a port-forward to the grafana pod
```bash
export POD_NAME=$(kubectl get pods --namespace default -l "app=grafana" -o jsonpath="{.items[0].metadata.name}")
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
$ kubectl get hpa online-store -w
NAME                REFERENCE                      TARGETS   MINPODS   MAXPODS   REPLICAS   AGE
online-store   Deployment/online-store   0 / 10    2         10        2          4m
online-store   Deployment/online-store   128500m / 10   2         10        2         4m
online-store   Deployment/online-store   170500m / 10   2         10        4         5m
online-store   Deployment/online-store   111500m / 10   2         10        4         5m
online-store   Deployment/online-store   95250m / 10   2         10        4         6m
online-store   Deployment/online-store   141 / 10   2         10        4         6m
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
