# How to create a production database on AWS RDS

在 RDS 的 DB Subnet Group 中混用公有與私有子網雖然不會被 AWS 立刻阻止，但往往不是最佳做法，原因在於：

AWS 亂選子網，無法預測 ENI 落點

RDS 只會保證「每個 AZ 至少一個子網、至少兩個 AZ」的需求，並不分公／私，自動在您指定的子網裡挑選來建立 ENI。

如果您同時放入公有與私有子網，RDS 可能在某個 AZ 用了公有子網，另一 AZ 卻用私有子網，導致行為不一致，像是有時您能 Internet 直連、有時卻只能在 VPC 內部存取。

最佳實踐：只放入一種類型的子網

私有子網：最常見的做法，將 DB 放在私有子網裡，並配合堡壘機、VPN 或 Direct Connect 來存取，安全性最高。

公有子網：若您確實需要「Publicly Accessible = Yes」並且要從 Internet 直連，就應該在這個 Subnet Group 只放那兩個公有子網，確保所有 AZ 的 ENI 都在能走 IGW 的子網中。

建議操作
若要把 RDS 完全私有化

建一組只包含「兩個私有子網」的 DB Subnet Group。

Modify RDS → 換用此 Subnet Group → Apply。

若要讓 RDS 完全公開可存取

建一組只包含「兩個公有子網」的 DB Subnet Group。

Modify RDS → 換用此 Subnet Group → Apply。

這麼做後，您就可以清楚地控制 RDS ENI 的落地子網類型，避免出現有時在私有、有時在公有的混亂狀況。

## 步驟

在 Aurora and RDS 服務下創建 Subnet groups ， Subnets 要是全私網或是全公網 -> 選擇或創建 VPC security groups 決定 inbound outbound ，創建的話先輸入名稱再到 ec2/security groups 下面改進出站規則 -> 點擊 create

