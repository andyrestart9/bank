# 在 Makefile 裡，**目标（target）**就是一段命令块的名字，加上它的依赖（prerequisites）一起定义了「当我执行 make <目标> 时要做哪些事」

postgres:
	docker run --name postgres12 -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:12-alpine

createdb:
	docker exec -it postgres12 createdb --username=root --owner=root bank

dropdb:
	docker exec -it postgres12 dropdb bank

migrateup:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/bank?sslmode=disable" -verbose up

migratedown:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/bank?sslmode=disable" -verbose down

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

# 在 Makefile 裡，每個「目標」（target）預設都對應到檔案名稱──Make 會檢查這個檔案是否存在，以及它的修改時間，來決定需不需要執行它下面的指令（recipe）。
# 所以我們要用 .PHONY 聲明「這些目標不是要對應檔案」，而是「純粹的命令集合」。
.PHONY: postgres createdb dropdb migrateup migratedown sqlc test
