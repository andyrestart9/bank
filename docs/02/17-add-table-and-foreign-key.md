# Add table & foreign key

## Create migrate

`migrate create -ext sql -dir db/migration -seq add_users`

## 往上升一版或往下降一版的指令

往上升一版

`migrate -path db/migration -database "postgresql://root:secret@localhost:5432/bank?sslmode=disable" -verbose up 1`

往下降一版
`migrate -path db/migration -database "postgresql://root:secret@localhost:5432/bank?sslmode=disable" -verbose down 1`