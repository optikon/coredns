### coredns with helm

This is a single node minikube example. Running off my laptop.


1. `minikube start`
2. configure `kubectl` on mac to talk to minikube cluster

```
➜  ~ kubectl get nodes
NAME       STATUS    ROLES     AGE       VERSION
minikube   Ready     <none>    1m        v1.9.0
```


3. install helm tiller on minikube with special privileges

- `kubectl create serviceaccount -n kube-system tiller`
- `kubectl create clusterrolebinding tiller-binding --clusterrole=cluster-admin --serviceaccount kube-system:tiller`
- `helm init --service-account tiller`

4. make sure helm is running

```
➜  ~ kubectl logs po/tiller-deploy-865dd6c794-4st4m --namespace=kube-system
[main] 2018/03/23 15:56:12 Starting Tiller v2.8.2 (tls=false)
[main] 2018/03/23 15:56:12 GRPC listening on :44134
[main] 2018/03/23 15:56:12 Probes listening on :44135
```


5. deploy prometheus onto cluster [using Helm](https://github.com/kubernetes/charts/tree/master/stable/prometheus):

```
helm install stable/prometheus
```

6. [Install helm chart](https://github.com/kubernetes/charts/tree/master/stable/coredns) for CoreDNS onto minikube cluster:
```
 helm install --name coredns -f coredns-helm-values.yaml --namespace=kube-system stable/coredns
```

6. CoreDNS will replace KubeDNS.
