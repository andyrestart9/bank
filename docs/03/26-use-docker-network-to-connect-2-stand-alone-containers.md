# How to use docker network to connect 2 stand-alone containers

用 image 創建 container

docker run 用 imger 創建容器

--name 指定容器名稱

-p 發布端口

`docker run --name 容器名稱 -p 容器端口:宿主機端口 鏡像名稱:鏡像標籤`

`docker run --name bank -p 8080:8080 bank:latest`

出現錯誤 cannot load config:Config File "app" Not Found in "[/app]"

因為我們之前在 Dockerfile 只有複製二進位執行文件到 run stage image ，但是沒有複製 config 文件，所以在複製二進位執行文件之後還要複製 config 文件，也就是 app.env `COPY app.env .`

清除啟動失敗的容器 `docker rm bank`

清除舊鏡像 `docker rmi bank:latest`

再構建一個新的 docker image ，再用新鏡像啟動一次容器就會成功了

我們可以用環境變數把 gin 改成 release 模式試試看

清除舊容器，用 -e option 設定環境變數啟動容器 `docker run --name bank -p 8080:8080 -e GIN_MODE=release bank:latest`

用 postman 對容器發出請求，測試是不是都正常，出現錯誤 500 Internal Server Error ， "error": "dial tcp [::1]:5432: connect: connection refused" ， port 5432 是我們的 postgres database

原因是我們在 app.env 寫的是通過 localhost 地址連線到資料庫，但是 bank 和 postgres 是兩個獨立的容器，所以他們不會有相同的網路 IP 地址

用 `docker container inspect` 命令看容器的網路設定

run `docker container inspect postgres12`

```text
"Networks": {
    "bridge": {
        "IPAMConfig": null,
        "Links": null,
        "Aliases": null,
        "MacAddress": "ea:17:f3:46:6e:59",
        "DriverOpts": null,
        "GwPriority": 0,
        "NetworkID": "3b9a44b1484d40c41fea82771b5bfefb169ceeb09504df9361803299440a298d",
        "EndpointID": "69904344970e54e05f60e29f1dd925061577e0468c71769fdb7ab895c76b7bc2",
        "Gateway": "172.17.0.1",
        "IPAddress": "172.17.0.2",
        "IPPrefixLen": 16,
        "IPv6Gateway": "",
        "GlobalIPv6Address": "",
        "GlobalIPv6PrefixLen": 0,
        "DNSNames": null
    }
}
```

run `docker container inspect bank`

```text
"Networks": {
    "bridge": {
        "IPAMConfig": null,
        "Links": null,
        "Aliases": null,
        "MacAddress": "96:4e:53:51:ff:40",
        "DriverOpts": null,
        "GwPriority": 0,
        "NetworkID": "3b9a44b1484d40c41fea82771b5bfefb169ceeb09504df9361803299440a298d",
        "EndpointID": "3bad46108361f7285203f7b04875db707741ed7d03bff8a2636556069597fa78",
        "Gateway": "172.17.0.1",
        "IPAddress": "172.17.0.3",
        "IPPrefixLen": 16,
        "IPv6Gateway": "",
        "GlobalIPv6Address": "",
        "GlobalIPv6PrefixLen": 0,
        "DNSNames": null
    }
}
```

可以看到雖然他們都是在 bridge 網路，但是兩個容器在不同的 IP 地址，postgres12 在 172.17.0.2 ， bank 在 172.17.0.3

我們可以用 -e option 覆蓋 bank 的 DB_SOURCE db 連線環境變數讓他連到 172.17.0.2

先清除舊容器

`docker stop bank`

`docker rm bank`

覆蓋 bank 的 DB_SOURCE db 連線環境並啟動容器，記得連線要用雙引號包起來，因為裡面有一些特殊字元

`docker run --name bank -p 8080:8080 -e GIN_MODE=release -e DB_SOURCE="postgresql://root:secret@172.17.0.2:5432/bank?sslmode=disable" bank:latest`

再打一次 api 就會成功了，因為我們正確連線到 db 的 ip 位址了

但是我們重新啟動容器 IP 位址會變，所以這樣的方式並不理想，我們可以用更好的方式，不通過 IP 地址去連接 postgres ，改用 user-defined network

列出現在有的 network `docker network ls`

查看 network 細節 `docker network inspect 網路名稱`

run `docker network inspect bridge` 查看 bridge network 的細節

可以看到 bridge network 裡的容器

```text
"Containers": {
    "9e4abe0483caebba53dc874d491dce024a97ff317e527ad9c974cd4255423568": {
        "Name": "postgres12",
        "EndpointID": "69904344970e54e05f60e29f1dd925061577e0468c71769fdb7ab895c76b7bc2",
        "MacAddress": "ea:17:f3:46:6e:59",
        "IPv4Address": "172.17.0.2/16",
        "IPv6Address": ""
    }
},
```

通常在同一 network 的容器可以通過 name 發現到對方，但是在 bridge network 不行

所以我們要創建一個自己的網路，再把 postgres 和 bank 容器放到這個 network 裡面

查看 docker network 命令手冊 run `docker network --help`

創建 network 的命令 `docker network create 網路名稱`

創建一個 network `docker network create bank-network`

然後我們可以用 docker network connect 命令把容器連接到網路 `docker network connect 網路名稱 容器名稱`

把我們的 postgres 容器連接到我們創建的 bank-network ， run `docker network connect bank-network postgres12`

run `docker network inspect bank-network` 檢查網路裡面有哪些容器

```text
"Containers": {
    "9e4abe0483caebba53dc874d491dce024a97ff317e527ad9c974cd4255423568": {
        "Name": "postgres12",
        "EndpointID": "730387015ad28aa7945ca1409939850b1f9bd442e4354e67ed516f2bddc21dfb",
        "MacAddress": "12:b0:dc:07:1a:a8",
        "IPv4Address": "172.18.0.2/16",
        "IPv6Address": ""
    }
},
```

我們再執行 `docker container inspect postgres12` 檢查容器連上了哪些網路

```text
"Networks": {
    "bank-network": {
        "IPAMConfig": {},
        "Links": null,
        "Aliases": [],
        "MacAddress": "12:b0:dc:07:1a:a8",
        "DriverOpts": {},
        "GwPriority": 0,
        "NetworkID": "888c291889596e4386b5c0a1970998f73c4f38a82de0175feebaeab07b6a5f48",
        "EndpointID": "730387015ad28aa7945ca1409939850b1f9bd442e4354e67ed516f2bddc21dfb",
        "Gateway": "172.18.0.1",
        "IPAddress": "172.18.0.2",
        "IPPrefixLen": 16,
        "IPv6Gateway": "",
        "GlobalIPv6Address": "",
        "GlobalIPv6PrefixLen": 0,
        "DNSNames": [
            "postgres12",
            "9e4abe0483ca"
        ]
    },
    "bridge": {
        "IPAMConfig": null,
        "Links": null,
        "Aliases": null,
        "MacAddress": "ea:17:f3:46:6e:59",
        "DriverOpts": null,
        "GwPriority": 0,
        "NetworkID": "3b9a44b1484d40c41fea82771b5bfefb169ceeb09504df9361803299440a298d",
        "EndpointID": "69904344970e54e05f60e29f1dd925061577e0468c71769fdb7ab895c76b7bc2",
        "Gateway": "172.17.0.1",
        "IPAddress": "172.17.0.2",
        "IPPrefixLen": 16,
        "IPv6Gateway": "",
        "GlobalIPv6Address": "",
        "GlobalIPv6PrefixLen": 0,
        "DNSNames": null
    }
}
```

可以看到 postgres12 連上了兩個網路， bank-network 和 bridge network

接下來我們可以清除舊容器，創建新容器連上 bank-network network

清除舊容器

`docker stop bank`

`docker rm bank`

創建新容器並連接上 bank-network network ，因為 bank 和 postgres12 在同一個網路，所以我們通過容器名稱來尋址，把 IP 地址 172.17.0.2 改成容器名稱 postgres12

`docker run --network bank-network -p 8080:8080 -e GIN_MODE=release -e DB_SOURCE="postgresql://root:secret@postgres12:5432/bank?sslmode=disable" bank:latest`

再打一次 api 就會成功了

我們再檢查看看 bank-network network ， run `docker network inspect bank-network` ，會發現裡面有兩個容器， bank 和 postgres12

```text
"Containers": {
    "4564340099f7796351ee0eca2ff39b17332f0f0ac7fcc083538564c1bade615f": {
        "Name": "mystifying_khorana",
        "EndpointID": "79a5eaf5f5e6f2338ba0a38ce1c985fdf0fc21e6d3719c56289e97a6996ee925",
        "MacAddress": "a6:f9:5f:58:6f:b7",
        "IPv4Address": "172.18.0.3/16",
        "IPv6Address": ""
    },
    "9e4abe0483caebba53dc874d491dce024a97ff317e527ad9c974cd4255423568": {
        "Name": "postgres12",
        "EndpointID": "730387015ad28aa7945ca1409939850b1f9bd442e4354e67ed516f2bddc21dfb",
        "MacAddress": "12:b0:dc:07:1a:a8",
        "IPv4Address": "172.18.0.2/16",
        "IPv6Address": ""
    }
},
```

這就是如何利用 user-defined network 讓兩個 stand-alone 的容器通過名稱和對方通信

最後，更新 Makefile 的 postgres 指令讓 postgres12 容器連接到 bank-network network

```Makefile
postgres:
    docker run --name postgres12 --network bank-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:12-alpine
```
