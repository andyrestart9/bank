# Automatic deploy to Kubernetes with Github Action

在 github runner 上安裝 Kubectl <https://github.com/marketplace/actions/kubectl-tool-installer>

部署 resources 到 cluster

## error validating data: failed to download openapi: the server has asked for the client to provide credentials

github action 的 assume role 必須要有訪問 EKS 集群的權限，否則會出現 error validating data: failed to download openapi: the server has asked for the client to provide credentials

問題分析

1. EKS 集群只允許創建者訪問：只有創建 EKS 集群的 IAM 身份（通常是 root user）才有初始訪問權限
2. GitHub Actions 角色沒有訪問權限：github-actions-ecr-push 角色沒有被授權訪問 EKS 集群
3. 無法應用 aws-auth.yaml：因為當前身份沒有權限，所以無法部署 aws-auth ConfigMap

> 解決方案

您需要使用有權限的身份（比如 root user）先授權 GitHub Actions 角色訪問 EKS 集群。有兩種方法：

方法 1：使用 EKS Access Entries（推薦）

根據您的文檔，使用 root user 或有權限的身份執行以下命令：

```sh
# 1. 為 GitHub Actions 角色創建 Access Entry
aws eks create-access-entry \
  --cluster-name bank \
  --region ap-northeast-1 \
  --principal-arn arn:aws:iam::149536480497:role/github-actions-ecr-push \
  --type STANDARD \
  --kubernetes-groups eks-admins

# 2. 創建 ClusterRoleBinding（使用 kubectl，確保你用的是有權限的身份）
kubectl create clusterrolebinding eks-admins-cluster-admin-binding \
  --clusterrole cluster-admin \
  --group eks-admins
```

方法 2：在 AWS 控制台操作

1. 切換到 root user：
    - 在 AWS 控制台用 root 帳戶登錄
    - 或者在本地 AWS CLI 切換到 root user profile
2. 通過 AWS 控制台配置 EKS 集群訪問：
    - 進入 EKS 控制台
    - 選擇 bank 集群
    - 進入 "Access" 標籤
    - 添加 IAM 角色 arn:aws:iam::149536480497:role/github-actions-ecr-push
