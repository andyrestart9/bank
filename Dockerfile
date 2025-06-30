# Build stage
FROM golang:1.24-alpine3.22 AS builder
WORKDIR /app
COPY . .
RUN go build -o main main.go

# Run stage
# 要能夠跑 app 我們只需要 go build 出來的 binary 檔案，所以我們可以用一個更小的 image 來執行 binary，我們先用 builder 的 image 來 build 出 binary 檔案，然後再從 builder 的 image 中複製出 binary 檔案，最後再在 run stage 的 image 中執行 binary 檔案。 
# 實際測試 image size 從 1.38GB 縮小到 47.5MB
FROM alpine:3.22
WORKDIR /app
COPY --from=builder /app/main .
COPY app.env .

EXPOSE 8080
CMD ["/app/main"]