# How to use Ingress to route traffics to different services in Kubernetes

<https://kubernetes.io/docs/concepts/services-networking/ingress/#hostname-wildcards>

把 eks/service.yaml   type: LoadBalancer 改成   type: ClusterIP ，因為我們不想讓外部進來，而是通過 Ingress 進來

eks/ingress.yaml

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: bank-ingress
spec:
  rules:
  - host: <route 53 的 A Record name>
    http:
      paths:
      - pathType: Prefix
        path: "/"
        backend:
          service:
            name: <service name>
            port:
              number: 80

```

提交變更： `kubectl apply -f eks/service.yaml` `kubectl apply -f eks/ingress.yaml`

到 k9s 看輸入 `:ingress` 是不是有我們剛剛設定的 ingress

Describe ingresses 會發現沒有外部 Address ，因為只有 Ingress 還不夠還需要設定 Ingress Controllers

<https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/>

<https://github.com/kubernetes/ingress-nginx/blob/main/README.md#readme>

<https://kubernetes.github.io/ingress-nginx/deploy/#aws>

會找到類似 `kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.13.0/deploy/static/provider/aws/deploy.yaml` 的指令，將 NGINX ingress 部署到在 AWS 上執行的 Kubernetes 叢集， NGINX ingress controller 將會暴露在 NLB 之後

成功的話再 k9s pods 會看到 ingress-nginx-controller ，如果 ingresses Address 還是空的可能是在 eks/ingress.yaml 沒有設定到 ingressClassName: nginx

<https://kubernetes.io/docs/concepts/services-networking/ingress/#ingress-class>

eks/ingress.yaml

```yaml
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: nginx
spec:
  controller: k8s.io/ingress-nginx
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: bank-ingress
spec:
  ingressClassName: nginx
  rules:
    - host: "api.bank9.click"
      http:
        paths:
          - pathType: Prefix
            path: "/"
            backend:
              service:
                name: bank-api-service
                port:
                  number: 80

```

為什麼 spec.controller 是 k8s.io/ingress-nginx ，因為在部署 NGINX ingress 的時候用的是 <https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.13.0/deploy/static/provider/aws/deploy.yaml> 其中有一行是 - --controller-class=k8s.io/ingress-nginx ， Describe ingress-nginx pod 也可以看到 --controller-class=k8s.io/ingress-nginx

重新提交 ingress `kubectl apply -f eks/ingress.yaml` 在 k9s Describe ingresses 就會看到 Address

把 route53 之前的 A 記錄設定成新的 Address ，Alias to Application and Classic Load Balancer ， Asia Pacific (Tokyo) ， ab20a09a7e727477eb36dafd1319dcba-cae5be6928192c95.elb.ap-northeast-1.amazonaws.com

nslook up 看看 `nslookup api.bank9.click`

再用 postman 打看看

流量追蹤： DNS -> NLB (ingress nginx) -> services -> pods -> containers