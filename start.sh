#!/bin/sh

set -e

echo "run db migration"

source /app/app.env # 在 Dockerfile 我們有 COPY app.env 到 WORKDIR /app ，所以先 source 載入環境變數，跑 db migration 時才會有環境變數可用

# /app/migrate：容器映像內的 migrate CLI 可執行檔路徑。
# -path /app/migration：指定遷移檔 (SQL 檔) 所在資料夾。
# -database "$DB_SOURCE"：資料庫連線字串由環境變數 DB_SOURCE 提供，維持機密與彈性。
# -verbose：顯示詳細執行過程，方便除錯。
# up：把遷移「往上」套用到最新版本。
/app/migrate -path /app/migration -database "$DB_SOURCE" -verbose up

echo "start the app"

# exec 會用傳入腳本的所有參數 ($@) 取代當前 shell 行程。
# 這樣可以讓應用程式承接 PID 1 的位置，正確收到 SIGTERM/SIGINT 等訊號（Docker、K8s 在停止容器時會送出），避免殭屍進程或無法優雅關閉。
# PID 1 是 /bin/sh（由 ENTRYPOINT ["sh", "-c", …] 產生），而你的應用程式是 shell 的子行程，訊號及殭屍回收都仰賴 shell 實作。
# 使用 exec /app/server 的做法能讓 你的 Go 服務本身成為 PID 1，直接收訊號並自行 wait() 子行程，減少潛在問題。
# 例如在 Dockerfile 的 CMD ["./start.sh", "/app/server"] 中，"$@" 便是 /app/server，exec 後 shell 消失，容器直接執行可執行檔。
exec "$@"