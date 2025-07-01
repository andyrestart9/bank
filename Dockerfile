# Build stage
FROM golang:1.24-alpine3.22 AS builder
WORKDIR /app
COPY . .
RUN go build -o main main.go
RUN apk add curl
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.18.3/migrate.linux-amd64.tar.gz | tar xvz

# Run stage
# 要能夠跑 app 我們只需要 go build 出來的 binary 檔案，所以我們可以用一個更小的 image 來執行 binary，我們先用 builder 的 image 來 build 出 binary 檔案，然後再從 builder 的 image 中複製出 binary 檔案，最後再在 run stage 的 image 中執行 binary 檔案。 
# 實際測試 image size 從 1.38GB 縮小到 47.5MB
FROM alpine:3.22
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/migrate ./migrate
COPY app.env .
COPY start.sh .
COPY db/migration ./migration

EXPOSE 8080

# CMD 是提供「預設的執行指令或參數」。
# ENTRYPOINT 是指定「永遠要執行的主程式」，就像容器的 PID 1。
# 如果同時有 ENTRYPOINT，CMD 不再是一條完整指令，而是給 ENTRYPOINT 的「預設參數」。
# 要執行複雜前置腳本，再交棒給主程式 → ENTRYPOINT 指向腳本，腳本裡 exec "$@"；CMD 指向主程式。
CMD ["/app/main"]
ENTRYPOINT ["/app/start.sh"]
