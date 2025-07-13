# How to use kubectl & k9s to connect to a kubernetes cluster on AWS EKS

install kubectl: <https://kubernetes.io/docs/tasks/tools/install-kubectl-macos/#install-with-homebrew-on-macos>

kubectl configuration, run `kubectl cluster-info` 預設會嘗試連線到 localhost:8080 所以會 connect: connection refused ，我們要改連 EKS cluster ，所以要先改 kube config

fetch AWS EKS cluster's information and store it in the config file 的指令 `aws eks update-kubeconfig --name <cluster name> --region <region name>`

example: `aws eks update-kubeconfig --name bank --region ap-northeast-1`

在 fetch AWS EKS cluster's information 要確認你現在的 AWS CLI user 或是所屬的 user group 有沒有訪問 eks 的權限

檢查有沒有寫進 kube config `cat ~/.kube/config`

如果你的機器上有多個針對不同叢集的上下文，你可以用 `kubectl config use-context <cluster>` 選擇你的上下文

example: `kubectl config use-context arn:aws:eks:ap-northeast-1:149536:cluster/bank`

測試連線到 EKS bank cluster , run `kubectl cluster-info`

如果出現

err="couldn't get current server API group list: the server has asked for the client to provide credentials"

error: You must be logged in to the server

參考 <https://repost.aws/knowledge-center/amazon-eks-cluster-access> ，原因可能是 aws cli 的使用者或角色不是當初建立 Amazon EKS 叢集的使用者或角色。需要設定 Amazon EKS 叢集的角色為基礎的存取控制 (RBAC) 以授權 IAM 實體。

就是 「這個 IAM 身分尚未被加入 EKS 的 aws-auth ConfigMap」。

EKS 會先檢查 token 內的 ARN 是否列在 aws-auth → mapUsers / mapRoles，不在名單就直接回 401，kubectl 便顯示上述訊息。

create Root user access keys ，改 ~/.aws/credentials ，把之前的 default 改名成 github ， default 改成 Root user 的 access key

測試 root user 能不能訪問 EKS cluster `kubectl cluster-info`

切回 github user 看看，指令 `export AWS_PROFILE=<username>` ， 範例： `export AWS_PROFILE=github`

會發現 github user 不能訪問

向 AWS STS 查自己目前憑證的身分（User / Role ARN、Account ID）: `aws sts get-caller-identity`

## 如何讓其他 user 訪問 EKS

步驟 1：為 github-ci 建立 Access Entry

挑一個自訂群組名稱，例如 eks-admins：

```sh
aws eks create-access-entry \
  --cluster-name bank \
  --region ap-northeast-1 \
  --principal-arn arn:aws:iam::149536480497:user/github-ci \
  --type STANDARD \
  --kubernetes-groups eks-admins
```

參數說明：  

- `--principal-arn`　要授權的 IAM User / Role。  
- `--type STANDARD`　標準模式（另一種是　`ASYNC`）。  
- `--kubernetes-groups eks-admins`　映射到 K8s 群組 `eks-admins`。  

→ 這條指令把「github-ci 這個 IAM User」登錄進 EKS 的 Access Control 系統。  

成功會回傳 JSON，裡面有 accessEntryId 等資訊。

步驟 2：在叢集裡建立 ClusterRoleBinding

用 root 身分執行下列指令，把 eks-admins 群組綁到現成的 cluster-admin 角色：

```sh
kubectl create clusterrolebinding eks-admins-cluster-admin-binding \
  --clusterrole cluster-admin \
  --group eks-admins
```

- clusterrolebinding　把「群組 / 使用者 / ServiceAccount」連結到「ClusterRole」。
- cluster-admin　K8s 內建的最高權限角色。
- --group eks-admins　剛才 Access Entry 指定的群組。

→ 完成「K8s 群組 → RBAC」這一段對映，群組成員繼承 cluster-admin 權限。

（如果先前試過相同名稱、Kubernetes 會說 already exists，可改用別的 binding 名稱再跑一次。）

步驟 3：切回 github-ci 測試

```bash
export AWS_PROFILE=github # 切換回 github-ci 的設定
kubectl cluster-info # 應該可以看到 API server URL
kubectl get nodes # 應列出節點；若目前還沒建立工作節點，會回 empty
```

## 跟 kubernetes 互動更方便的工具 - k9s

install: <http://k9scli.io/topics/install/>
