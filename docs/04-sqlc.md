# sqlc

<https://sqlc.dev/>

## Install

<https://docs.sqlc.dev/en/latest/overview/install.html>

`brew install sqlc`

see how to use sqlc

`sqlc help` or <https://docs.sqlc.dev/en/latest/reference/cli.html>

## Configuration

<https://docs.sqlc.dev/en/latest/reference/config.html>

### golang options

<https://docs.sqlc.dev/en/latest/reference/config.html#go>

### Example

```yaml
version: "2" # 版本号 2 是当前的版本
sql: # 数据库配置 
  - schema: "db/migration" # schema DDL 文件
    queries: "db/query" # 查询 DML 文件
    engine: "postgresql" # 数据库引擎
    gen: # 生成配置
      go: # 生成 Go 代码
        package: "db" # 包名
        out: "db/sqlc" # 输出目录
        sql_package: "database/sql" # 使用 database/sql 包
        emit_json_tags: true # 生成 JSON 标签
        emit_interface: true # 生成 Querier 接口
        emit_empty_slices: true # 如果为 true，则 :many 查询返回的切片将为空而不是 nil
```

記得要創建 db/query, db/sqlc 這兩個目錄

## Write queries

<https://docs.sqlc.dev/en/latest/tutorials/getting-started-postgresql.html#schema-and-queries>

example:

db/query/account.sql

```sql
-- name: CreateAccount :one
INSERT INTO accounts (
  owner, balance, currency
) VALUES (
  $1, $2, $3
)
RETURNING *;

```

## Initialize module and install dependencies

Initialize module

`go mod init github.com/andyrestart9/bank`

install dependencies

`go mod tidy`

## 配置生成命令

```Makefile
sqlc:
    sqlc generate

.PHONY: sqlc
```

執行命令 `make sqlc`
