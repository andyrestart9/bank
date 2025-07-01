# 31. Store & retrieve production secrets with AWS secrets manager

store environment variable on AWS sevret manager

## Install AWS cli

<https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html>

setup credentuals to access AWS account

run `aws configure`

輸入 IAM user 的 access key id 和 secret access key

這個使用這要有 AmazonEC2ContainerRegistryFullAccess 和 AmazonEC2ContainerRegistryFullAccess permissions

取回 Secrets Manager store 的 secrets ， 指令 `aws secretsmanager get-secret-value --secret-id [Secret name or Secret ARN] --query SecretString --output text`

run `aws secretsmanager get-secret-value --secret-id bank --query SecretString --output text`

output:

```text
{"DB_SOURCE":"postgresql://root:hIYy5e@bank.cns0e0k8sdy2.ap-northeast-1.rds.amazonaws.com:5432/bank","DB_Driver":"postgres","SERVER_ADDRESS":"0.0.0.0:8080","ACCESS_TOKEN_DURATION":"15m","TOKEN_SYMMETRIC_KEY":"9117465bc5"}
```

但是他的 output 是 json string 所以我們可以下載 jq 工具幫我們拿到想要的值

下載網址： <https://jqlang.org/download/> ，使用手冊： <https://jqlang.org/manual/>

run `aws secretsmanager get-secret-value --secret-id bank --query SecretString --output text | jq -r 'to_entries|map("\(.key)=\(.value)")|.[]' > app.env` 測試從 secrets manager 取回 secrets ，用 jq 改成我們要的格式，再寫進 app.env ，成功後就可以放到 deploy.yml

deploy.yml 不需要再寫下載 jq 和 AWS CLI ，因為 github actions 的 runner 上 Ubuntu 映像內建 AWS CLI 和 jq ，
