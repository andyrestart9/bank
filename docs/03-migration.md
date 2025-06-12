# Migration

## Install

<https://github.com/golang-migrate/migrate/tree/master/cmd/migrate#macos>

install golang-migrate

`brew install golang-migrate`

## Migrate 操作手冊

<https://github.com/golang-migrate/migrate/tree/master/cmd/migrate#usage>

`migrate -help`

## Create migration file

`migrate create -ext sql -dir db/migration -seq init_schema`

create [-ext E] [-dir D] [-seq] NAME

-ext: extension of the file

-dir: dest directory

-seq generate sequential version number

指令執行完會在指定 dir 產生 `[seq]_[NAME].up.[sql]` 向上遷移的檔案和 `[seq]_[NAME].up.[sql]` 向下遷移的檔案

我們要把 migrate sql 寫入這兩個產生的檔案

000001_init_schema.up.sql 可以用我們在 dbdiagram 產生的 sql <https://dbdiagram.io/d/bank-6834442d0240c65c443a818e>

000001_init_schema.down.sql 要把通過 up sql 的 changes revert ，有 foreign key constraint 的 table 要先 drop

## Get into postgresql container

`docker exec -it postgres12 /bin/sh`

通過 sh 和 postgres12 comtainer 交互，postgres container 有給我們一些 CLI 命令讓我們從 shell 直接和 postgres server 交互，像是 createdb dropdb 等等

`createdb --username=root --owner=root bank`

--username=root: 用 root user 連接

--owner=root: db owner 是 root

bank: db name

`psql bank`

用 postgresql 的 client `psql` access bank db

`\q`

退出 psql

`dropdb bank`

刪除 db bank

`exit`

退出 shell

## 創建 db

創建 db

`docker exec -it postgres12 createdb --username=root --owner=root bank`

不通過 shell access database console

`docker exec -it postgres12 psql -U root bank`

## Makefile

``` Makefile
# 在 Makefile 裡，**目标（target）**就是一段命令块的名字，加上它的依赖（prerequisites）一起定义了「当我执行 make <目标> 时要做哪些事」

postgres:
 docker run --name postgres12 -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:12-alpine

createdb:
 docker exec -it postgres12 createdb --username=root --owner=root bank

dropdb:
 docker exec -it postgres12 dropdb bank

# 在 Makefile 裡，每個「目標」（target）預設都對應到檔案名稱──Make 會檢查這個檔案是否存在，以及它的修改時間，來決定需不需要執行它下面的指令（recipe）。
# 所以我們要用 .PHONY 聲明「這些目標不是要對應檔案」，而是「純粹的命令集合」。
.PHONY: postgres createdb dropdb
```

## 用 migrate migrate up 和 migrate down

### migrate up

`migrate -path db/migration -database "postgresql://root:secret@localhost:5432/bank?sslmode=disable" -verbose up`

-path            Shorthand for -source=file://path

-database        Run migrations against this database (driver://url)

-verbose         Print verbose logging

up [N]       Apply all or N up migrations

除了執行指定的 migrate sql 還會創建 schema_migrations table ，裡面兩個欄位， version 和 dirty ， version 用來記錄 migration 的版本， dirty 用來記錄 migration 成功或是失敗， false 表示成功 ， true 表示失敗，需要手動修復才能繼續進行其他的 migration

### migrate down

`migrate -path db/migration -database "postgresql://root:secret@localhost:5432/bank?sslmode=disable" -verbose down`
