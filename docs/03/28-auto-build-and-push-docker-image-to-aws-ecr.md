# Auto build & push docker image to AWS ECR with Github Actions

AWS ECR -> Private repositories -> 全部都選預設 -> create

創建 .github/workflows/deploy.yml -> 到 github marketplace <https://github.com/marketplace> 找 AWS 官方出的 ECR "Login" Action <https://github.com/marketplace/actions/amazon-ecr-login-action-for-github-actions> 配置憑證登入 ECR ， build and push docker image to ECR
