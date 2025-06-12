# note

```yml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/query"
    schema: "db/migration"
    gen:
      go:
        package: "db"
        out: "db/sqlc"
        sql_package: "database/sql"
        emit_json_tags: true
        emit_interface: true
        emit_empty_slices: true
```

---
SELECT * FROM transfers
WHERE
    from_account_id = $1 OR
    to_account_id = $2
ORDER BY id
LIMIT $3
OFFSET $4;
