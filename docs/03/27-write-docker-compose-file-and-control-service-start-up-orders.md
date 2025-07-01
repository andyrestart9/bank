# How to write docker-compose file and control service start-up orders

docs: <https://docs.docker.com/reference/compose-file/> , <https://docs.docker.com/compose/>

docker compose 啟動指令 `docker compose up`

啟動過程可以在終端機看到

```text
 ✔ api                        Built                                                                                                                                                                0.0s 
 ✔ Network bank_default       Created                                                                                                                                                              0.0s 
 ✔ Container bank-api-1       Created                                                                                                                                                              0.1s 
 ✔ Container bank-postgres-1  Created                                                                                                                                                              0.1s 
Attaching to api-1, postgres-1
```

docker compose 會用`根目錄名-服務名`構建 image ，自動創建 network ， network 名稱會是 `根目錄名_default`，容器名稱名稱會是`根目錄名-服務名-序列號`，並把容器連上自動創建的 network

用 `docker network inspect bank_default` 可以看到

```text
"Containers": {
    "8aadcd278bc3006abf643cf68f953043140dcb6d782fa4e6da3b38f012965e25": {
        "Name": "bank-api-1",
        "EndpointID": "69920fc9c04ad443345b8f8430534f6605fa287d2cdf35d805e617f3612964ab",
        "MacAddress": "4a:ea:aa:b7:d5:b1",
        "IPv4Address": "172.20.0.2/16",
        "IPv6Address": ""
    },
    "d7281e30a043ee2b9e55012dd5db1b1c3c33b8b750a98578ed4f9d95ef4a51d1": {
        "Name": "bank-postgres-1",
        "EndpointID": "ccccc73f2c23ee1a45cff88606c13fa211760c29996174a8326b62e213285af5",
        "MacAddress": "22:2b:9d:69:b7:ea",
        "IPv4Address": "172.20.0.3/16",
        "IPv6Address": ""
    }
},
```

在 Dockerfile 的 Build stage 要下載 curl ，在下載 migrate ，在 Run stage 要把 Build stage 下載的 migrate 複製到 Run stage ，把 db/migration 的檔案都複製到 Run stage

寫一個 sh 用於在啟動 app 前先跑 db migration ，創建 start.sh ，執行 `chmod +x start.sh` ，讓這個檔案可以被執行

在執行 start.sh 前要確保 postgres 服務已經啟動準備好了，因為啟動到準備好需要一段時間，如果服務還沒準備好就連線會出現錯誤，連線被拒絕， error: failed to open database: dial tcp 172.20.0.2:5432: connect: connection refused ，可以參考 <https://docs.docker.com/compose/how-tos/startup-order/>

所以我們先清除舊的容器、 network 、 image

清除舊容器和 network `docker compose down`

清除舊鏡像 `docker rmi 鏡像名:tag`

在 docker-compose.yaml 的 postgres service 加上 healthcheck

```yaml
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U root -d bank"]
      interval: 2s
      timeout: 1s
      retries: 10
```

api service 加上 depends_on

```yaml
    depends_on:
         postgres:
           condition: service_healthy
```

再 `docker compose up` 一次就正常運行了
