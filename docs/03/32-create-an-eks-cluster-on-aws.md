# How to create an EKS cluster on AWS

## NodeCreationFailure 的原因

發現創建 Node Group 失敗，錯誤訊息 NodeCreationFailure ，把 Node Group 子網設定成功有子網後就成功創建了

因為當你把 Node Group 的子網都換成「公有子網」後，節點就能自動拿到公網 IP 並透過 Internet Gateway 直接跟 EKS 控制平面通訊，所以就能順利加入叢集。

公有子網：啟用了 Auto-assign public IPv4，配合路由表裡 0.0.0.0/0 → igw-…，EC2 節點一開機就有公網 IP，能立刻從公網呼叫 EKS API。

私有子網（原本設定）：沒有公網 IP，又沒走 NAT Gateway 或 PrivateLink，因此節點根本無法連到控制平面，導致 NodeCreationFailure。

因為 Node Group 掛在那兩個「純私有子網」裡，而私有子網又沒有任何出口（既沒 NAT Gateway、也沒 PrivateLink Interface Endpoint），節點啟動後就無法連到 EKS 控制平面（API Server），才會一直卡在 NodeCreationFailure。

## 如果想在 Node Group 想設定在私有子網怎麼做？

私有子網 + NAT Gateway

1. 確認子網分佈

    公有子網：確保有路由 0.0.0.0/0 → igw-…，且 Auto-assign public IPv4 = Enabled

    私有子網：目前只有路由到本地 VPC，不對外

2. 準備 Elastic IP

    - VPC → Elastic IPs → Allocate Elastic IP → 記下 EIP-xxx

3. 建立 NAT Gateway

    - VPC → NAT Gateways → Create NAT Gateway

        - Subnet：選一個公有子網

        - Elastic IP：選剛才的 EIP-xxx

    - Create → 記下 NAT-yyy

4. 更新私有子網的路由表

    - VPC → Route Tables → 找跟私有子網關聯的那張

    - Edit routes → Add route

        ```text
        Destination: 0.0.0.0/0  
        Target:      nat-yyy
        ```

    - Save

5. 驗證私網出站

    - 啟一台臨時 EC2 放在私有子網，不要公網 IP

    - SSH/SSM 連入後：

        ```bash
        curl -s <http://checkip.amazonaws.com/>     # 應回傳你綁在 NAT Gateway 的公網 IP
        ```

6. 刪除舊的混合／公網 Node Group

    - EKS → Clusters → bank → Compute → Node groups → 選舊的 → Delete

7. 建立新的私有 Node Group

    - EKS → Clusters → bank → Compute → Add Node group

        - Subnets：只選那些私有子網

        - Remote access：關閉（因為你要用 SSM 或 Bastion）

        - 其他維持原設

    - Create

8. 驗證節點加入

    ```bash
    kubectl get nodes
    ```

    應看到新節點 Ready。
