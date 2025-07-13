# How to deploy a web app to Kubernetes cluster on AWS EKS

Doc: <https://kubernetes.io/docs/concepts/workloads/controllers/deployment/>

部署指令： `kubectl apply -f eks/deployment.yaml`

在 k9s 中 describe pods 最下面的 Events 會出現錯誤訊息 nodes are available: 1 Too many pods. preemption: 0/1 nodes are available: 1 No preemption victims found for incoming pod.

往下挖，describe nodes 會發現 Capacity pods 和 Allocatable pods 都是 4 ，然後下面 Non-terminated Pods 已經有 Namespace kube-system 的 4 個 pods ，所以沒辦法再 deploy pod ，可以查一下我們開的 EC2 type t3.micro 可以開多少 pods

<https://github.com/awslabs/amazon-eks-ami/blob/main/nodeadm/internal/kubelet/eni-max-pods.txt>

EC2 執行個體上可執行的最大 pod 數量取決於 Elastic Network Interfaces（或 ENI）的數量以及該執行個體上允許的每個 ENI 的 IP 數量。

pod 的最大數量公式： `# of ENI * (# of IPv4 per ENI - 1) + 2`

<https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-eni.html>

<http://docs.aws.amazon.com/ec2/latest/instancetypes/gp.html#gp_network>

t3.micro: Max. network interfaces: 2, IP addresses per interface: 2

所以 pod 的最大數量是 2*(2-1)+2=4

所以我們必須升級機器增加 pods 的容量，Edit node group 沒辦法改 Instance types ，所以要刪掉 Node group 重建一個

可以升級到 t3.small ， t3.small: Max. network interfaces: 3, IP addresses per interface: 4

所以 pod 的最大數量是 3*(4-1)+2=11

node group ready 之後再進 k9s Describe pod ，看是不是正常，對 pod 回車可以進去 container ，選 container 按 l 可以 container

## 讓外部流量進來 EKS

<https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service>

提交 service 指令： `kubectl apply -f eks/service.yaml`

到 k9s services 看有沒有 EXTERNAL-IP

用 nslookup 指令看能不能連上 service `nslookup <EXTERNAL-IP>` ，舉例： `nslookup ad93b571a0c954ada8337450202343bb-937725869.ap-northeast-1.elb.amazonaws.com`

log container 再用 postman 打打看應該會看到 log

把 deploy 的 replicas 改成 2 ， `kubectl apply -f eks/deployment.yaml` ， Describe bank-api-service ，會變成有兩個 Endpoints ， pods 也會變成兩個，因為 LoadBalancer type 的 service 會幫我們分配流量到這兩個 pods
