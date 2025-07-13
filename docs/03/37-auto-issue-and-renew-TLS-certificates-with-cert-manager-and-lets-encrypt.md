# Auto issue & renew TLS certificates with cert-manager and Let's Encrypt

<https://kubernetes.io/docs/concepts/services-networking/ingress/#tls>

## install cert-manager

<https://cert-manager.io/docs/installation/kubectl/#1-install-from-the-cert-manager-release-manifest>

`kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.18.2/cert-manager.yaml`

安裝完後去 namespaces cert-manager 裡面看有沒有 cert-manager, cert-manager-cainjector, and cert-manager-webhook pods 有就表示成功了

## config and deploy

<https://cert-manager.io/docs/configuration/acme/#creating-a-basic-acme-issuer>

看設定： eks/issuer.yaml

設定完後提交 `kubectl apply -f eks/issuer.yaml`

到 k9s 搜尋 `:clusterissuers`

Describe clusterissuers 看看 Registered Email 是不是你寫的 email

到 `:secrets` 會發現有私鑰了： letsencrypt-account-private-key

這時 issuer 應該準備好簽發 TLS certificates

`:certificates` 卻發現沒有任何 certificates

`:certificaterequests` 也是沒有發出任何證書請求

那是因為我們還沒把 issuer 接到 ingress

<https://cert-manager.io/docs/usage/ingress/#how-it-works>

```yaml
metadata:
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt
```

```yaml
  tls:
    - hosts:
        - api.bank9.click
      secretName: bank-api-cert
```

提交變更 `kubectl apply -f eks/ingress.yaml`

進入 k9s 切換到 all namespace 看有沒有 certificates 和 certificaterequests

`:namespace` 選 all ，`:ingress` ， Describe ingress 會看到 TLS 已啟用

```txt
TLS:                                                                                                             bank-api-cert terminates api.bank9.click
Rules:
  Host             Path  Backends
  ----             ----  -------- 
  api.bank9.click 
                    /   bank-api-service:80 (10.0.13.154:8080,10.0.15.4:8080)   
```

解釋：

TLS (傳輸層安全性)

- bank-api-cert terminates api.bank9.click
  - 意思: 這表示有一個名為 bank-api-cert 的 TLS/SSL 憑證。當有使用者透過 HTTPS 協定存取 api.bank9.click 這個網址時，Ingress 控制器會使用此憑證來進行 TLS 握手，建立一個安全的加密連線。
  - terminates (終止) 這個詞的意思是，加密的 HTTPS 流量到 Ingress 控制器這裡就會被解密，然後 Ingress 會將未加密的 HTTP 流量轉發到後端的服務。
Rules (路由規則)
- Host: api.bank9.click
意思: 這個規則只適用於目標主機名稱 (Host) 為 api.bank9.click 的請求。
- Path: /
意思: 這個規則會匹配該主機下的所有路徑。例如，<https://api.bank9.click/users> 和 <https://api.bank9.click/login> 都會符合此規則。
- Backends: bank-api-service:80 (10.0.13.154:8080, 10.0.15.4:8080)
  - 意思: 所有符合上述 Host 和 Path 規則的流量，都會被轉發到一個名為 bank-api-service 的 Kubernetes Service 的 80 埠。
  - 這個 Service 接著會將流量進行負載平衡，分送到後端兩個實際運行的應用程式 Pod。這兩個 Pod 的 IP 位址和埠號分別是 10.0.13.154:8080 和 10.0.15.4:8080。

也會看到 Message: Successfully created Certificate "bank-api-cert"

搜尋 certificate `:certificate` 可能在 Status 下會看到 Issuing 表示 cert-manager 正在叫 Let's Encrypt 簽發 certificate

如果過了很久還是 Issuing 可能是卡住了可以清掉 Pod 和卡住的 CertificateRequest/Order/Challenge 讓 cert-manager 重新啟動

> 重新啟動 cert-manager 的流程

進入 k9s

1. 釋放節點容量再排新版 Pod

   查看 Deployments `:deployment` 找到

    - cert-manager
    - cert-manager-cainjector
    - cert-manager-webhook

    按 s（scale）→ 在右下角輸入 0 → Enter

    再用 s 把每個 Deployment 改回 1

2. 清掉卡住的 CertificateRequest/Order/Challenge

   `:certificaterequests`  Shift+D 刪除對應的 certificaterequests

    `:orders` Shift+D 刪除對應的 orders

    `:challenges`  Shift+D 刪除對應的 challenges

---

> challenge: Error presenting challenge: admission webhook "validate.nginx.ingress.kubernetes.io" denied the request: ingress contains invalid paths: path /.well-known/acme-challenge/lvbckuLnFTRp-_XLU9h7CO3-4Rydr07HN3kPW9Rqs50 cannot be used with pathType Exact

進 k9s `:challenge` 發現 Error presenting challenge: admission webhook "validate.nginx.ingress.kubernetes.io" denied the request: ingress contains invalid paths: path /.well-known/acme-challenge/lvbckuLnFTRp-_XLU9h7CO3-4Rydr07HN3kPW9Rqs50 cannot be used with pathType Exact

在 `:ConfigMap` 的 ingress-nginx-controller

選中 ingress-nginx-controller，按 e 進入編輯

```yaml
data:
  strict-validate-path-type: "false"
```

重新提交 `kubectl apply -f eks/ingress.yaml` `kubectl apply -f eks/issuer.yaml`

---

成功的話 Describe certificate ， Status 底下會看到 Reason: Ready 還有三個時間 Not After, Not Before Renewal, 且 Events 下會有 Message: Time The certificate has been successfully issued

`:certificaterequest` 會看到 APPROVED True

`:ingress` 會看到 PORTS 80, 443

用 postman 把 http 改成 https 打通就成功了
