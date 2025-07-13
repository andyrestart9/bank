# Register a domain & set up A-record using Route53

到 route53 註冊 domain

註冊成功後到你註冊時填的 email 收信，點擊驗證連結

到 Hosted zones ，Create record type A ，勾選 Alias ， Alias to Network Load Balancer ， Asia Pacific (Tokyo) ， 輸入 k9s 看到的 EXTERNAL-IP 也就是 ELB 的 DNS name ad93b571a0c954ada8337450202343bb-937725869.ap-northeast-1.elb.amazonaws.com

用 postman 往剛剛設定的 domain 打看看是否正常
