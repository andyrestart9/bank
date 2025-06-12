# How to handle deadlock in golang

## “丢失更新”（lost update）

在事务里先单独 GetAccount 然后再 UpdateAccount，在高并发下两条事务可能会同时读到同一个旧余额，然后各自算出相同的新余额再写回去，导致“丢失更新”（lost update）：

两个事务都读到 100，减 10 后都写 90，最终只减了一次。

### 證明

開啟兩個終端， `docker exec -it postgres12 psql -U root -d bank` 都進入交互界面，用 psql 客戶端訪問 bank 資料庫

兩個終端都輸入 `BEGIN;` 開始 transaction

分別在兩個終端都輸入 `SELECT * FROM accounts WHERE id = 1;` 會發現兩邊都會返回查詢結果，也就表示兩個 transaction 會查到同樣的餘額並更新該餘額，出現做了兩次 100 - 10 的狀況，但我們預期的是先查詢到 100 再減 10 再讀到 90 再減 10

證明完後在兩個客戶端都輸入 `ROLLBACK;` 退回 transation

## 如何用行排他鎖（TupleExclusiveLock）防止“丢失更新”（lost update）

開啟兩個終端， `docker exec -it postgres12 psql -U root -d bank` 都進入交互界面，用 psql 客戶端訪問 bank 資料庫

兩個終端都輸入 `BEGIN;` 開始 transaction

先在第一個客戶端輸入 `SELECT * FROM accounts WHERE id = 1 FOR UPDATE;`

再在第二個客戶端輸入 `SELECT * FROM accounts WHERE id = 1 FOR UPDATE;`

會發現只有第一個客戶端返回查詢結果，第二個客戶端沒有返回查詢結果，這是因為第二個 transaction 在等第一個 transaction COMMIT 或 ROLLBACK

我們在第一個客戶端執行 `COMMIT;` 後，行鎖解除，第二個客戶端就返回查詢結果了

## 加上行排他鎖（TupleExclusiveLock）之後遇到死鎖 pq: deadlock detected 怎麼辦？

加入 context 和 log 抓出執行哪些 sql 時會出現 deadlock

```sh
Running tool: /Users/andylin/.goenv/versions/1.24.3/bin/go test -timeout 30s -run ^TestTransferTx$ github.com/andyrestart9/bank/db/sqlc -v -count=1

=== RUN   TestTransferTx
>> before: 981 208
tx 3 create transfer
tx 3 create entry 1
tx 3 create entry 2
tx 1 create transfer
tx 2 create transfer
tx 3 get account 1
tx 3 update account 1
tx 3 get account 2
tx 3 update account 2
tx 1 create entry 1
tx 2 create entry 1
tx 2 create entry 2
tx 1 create entry 2
tx 2 get account 1
tx 1 get account 1
>> tx: 971 218
tx 1 update account 1
    /Users/andylin/Documents/github/andyrestart9/bank/db/sqlc/store_test.go:62:
         Error Trace: /Users/andylin/Documents/github/andyrestart9/bank/db/sqlc/store_test.go:62
         Error:       Received unexpected error:
                      pq: deadlock detected
         Test:        TestTransferTx
--- FAIL: TestTransferTx (1.03s)
FAIL
FAIL github.com/andyrestart9/bank/db/sqlc 1.214s
```

從 log 可以發現 tx3 已經完成， deadlock 是由 tx1 tx2 造成的，所以我們可以開兩個 psql client 重現錯誤

### 重現錯誤

terminal2

```sql
BEGIN;
INSERT INTO transfers(from_account_id,to_account_id,amount)VALUES(1,2,10)RETURNING*;
INSERT INTO entries(account_id,amount)VALUES(1,-10)RETURNING*;
```

terminal1

```sql
BEGIN;
INSERT INTO transfers(from_account_id,to_account_id,amount)VALUES(1,2,10)RETURNING*;
```

terminal2

```sql
INSERT INTO entries(account_id,amount)VALUES(2,10)RETURNING*;
SELECT*FROM accounts WHERE id=1 FOR UPDATE;
```

這時候會發現查詢語句沒有返回結果，被 terminal1 的 tx1 block 了

這時候就會發生鎖衝突

### 證明鎖衝突

<https://wiki.postgresql.org/wiki/Lock_Monitoring>

以下查詢可能有助於查看哪些程序正在阻塞 SQL 語句（這些查詢只會尋找行級鎖，而不是物件級鎖）。

執行

```sql
SELECT blocked_locks.pid     AS blocked_pid,
        blocked_activity.usename  AS blocked_user,
        blocking_locks.pid     AS blocking_pid,
        blocking_activity.usename AS blocking_user,
        blocked_activity.query    AS blocked_statement,
        blocking_activity.query   AS current_statement_in_blocking_process
FROM  pg_catalog.pg_locks         blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity  ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks         blocking_locks 
    ON blocking_locks.locktype = blocked_locks.locktype
    AND blocking_locks.database IS NOT DISTINCT FROM blocked_locks.database
    AND blocking_locks.relation IS NOT DISTINCT FROM blocked_locks.relation
    AND blocking_locks.page IS NOT DISTINCT FROM blocked_locks.page
    AND blocking_locks.tuple IS NOT DISTINCT FROM blocked_locks.tuple
    AND blocking_locks.virtualxid IS NOT DISTINCT FROM blocked_locks.virtualxid
    AND blocking_locks.transactionid IS NOT DISTINCT FROM blocked_locks.transactionid
    AND blocking_locks.classid IS NOT DISTINCT FROM blocked_locks.classid
    AND blocking_locks.objid IS NOT DISTINCT FROM blocked_locks.objid
    AND blocking_locks.objsubid IS NOT DISTINCT FROM blocked_locks.objsubid
    AND blocking_locks.pid != blocked_locks.pid

JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.granted;
```

返回查詢結果

```csv
blocked_pid,blocked_user,blocking_pid,blocking_user,blocked_statement,current_statement_in_blocking_process
28254,root,28312,root,SELECT*FROM accounts WHERE id=1 FOR UPDATE;,"INSERT INTO transfers(from_account_id,to_account_id,amount)VALUES(1,2,10)RETURNING*;"
```

这段结果告诉我们，进程 28254 (blocked_pid) 正在执行

```sql
SELECT * FROM accounts WHERE id=1 FOR UPDATE;
```

但因为它在等待一把“针对事务 ID的共享锁（ShareLock on transactionid）”，而这把锁正被进程 28312 (blocking_pid) 持有。28312 此时正在执行：

```sql
INSERT INTO transfers(from_account_id, to_account_id, amount)
  VALUES (1,2,10) RETURNING *;
```

具体原理：

`INSERT INTO transfers`

在 transfers 表上拿了一个 RowExclusiveLock（表层写锁），以及它自己的 事务ID 排他锁（ExclusiveLock on transactionid），因为这笔插入还没提交。

这个排他锁表明：在它提交/回滚之前，任何要做外键相关检查的事务都不能获得“针对这个事务ID”的共享锁。

`SELECT … FOR UPDATE`

除了在 accounts.id=1 这行上拿一个行级排他锁（tuple‐level ExclusiveLock）之外，为了外键一致性，PostgreSQL 还会对所有“已插入或更新了 transfers 并引用了 accounts.id=1”的活跃事务申请一把共享锁（ShareLock on transactionid）。

只有等到那些子表事务（也就是进程 28312）结束，释放它的事务ID排他锁后，才允许 SELECT … FOR UPDATE 拿到它的共享锁，进而完成对父表行的锁定。

因此，这里看到的阻塞，就完全是因为：

28312 持有它自己事务的排他锁（正在插入还没提交），

28254 又必须拿该事务的一把共享锁，

排他锁 vs 共享锁 冲突，只能等 28312 完成后才能继续。

---

接著往下挖

以下查詢是對同一數據的另一種看法，它包含了一個狀態的歷史

```sql
SELECT
 a.datname,
 a.application_name,
 l.relation::regclass,
 l.transactionid,
 l.mode,
 l.locktype,
 l.GRANTED,
 a.usename,
 a.query,
 a.pid
FROM
 pg_stat_activity a
 JOIN pg_locks l ON l.pid = a.pid
WHERE
 a.application_name = 'psql'
ORDER BY
 a.pid;
```

返回查詢結果

```csv
datname,application_name,relation,transactionid,mode,locktype,granted,usename,query,pid
bank,psql,,2420,ExclusiveLock,transactionid,true,root,SELECT*FROM accounts WHERE id=1 FOR UPDATE;,28254
bank,psql,accounts,,RowShareLock,relation,true,root,SELECT*FROM accounts WHERE id=1 FOR UPDATE;,28254
bank,psql,transfers_id_seq,,RowExclusiveLock,relation,true,root,SELECT*FROM accounts WHERE id=1 FOR UPDATE;,28254
bank,psql,entries_id_seq,,RowExclusiveLock,relation,true,root,SELECT*FROM accounts WHERE id=1 FOR UPDATE;,28254
bank,psql,entries,,RowExclusiveLock,relation,true,root,SELECT*FROM accounts WHERE id=1 FOR UPDATE;,28254
bank,psql,transfers,,RowExclusiveLock,relation,true,root,SELECT*FROM accounts WHERE id=1 FOR UPDATE;,28254
bank,psql,,,ExclusiveLock,virtualxid,true,root,SELECT*FROM accounts WHERE id=1 FOR UPDATE;,28254
bank,psql,,2421,ShareLock,transactionid,false,root,SELECT*FROM accounts WHERE id=1 FOR UPDATE;,28254
bank,psql,accounts_owner_idx,,RowShareLock,relation,true,root,SELECT*FROM accounts WHERE id=1 FOR UPDATE;,28254
bank,psql,accounts_pkey,,RowShareLock,relation,true,root,SELECT*FROM accounts WHERE id=1 FOR UPDATE;,28254
bank,psql,accounts_pkey,,RowShareLock,relation,true,root,"INSERT INTO transfers(from_account_id,to_account_id,amount)VALUES(1,2,10)RETURNING*;",28312
bank,psql,accounts,,RowShareLock,relation,true,root,"INSERT INTO transfers(from_account_id,to_account_id,amount)VALUES(1,2,10)RETURNING*;",28312
bank,psql,transfers_id_seq,,RowExclusiveLock,relation,true,root,"INSERT INTO transfers(from_account_id,to_account_id,amount)VALUES(1,2,10)RETURNING*;",28312
bank,psql,transfers,,RowExclusiveLock,relation,true,root,"INSERT INTO transfers(from_account_id,to_account_id,amount)VALUES(1,2,10)RETURNING*;",28312
bank,psql,,,ExclusiveLock,virtualxid,true,root,"INSERT INTO transfers(from_account_id,to_account_id,amount)VALUES(1,2,10)RETURNING*;",28312
bank,psql,accounts_owner_idx,,RowShareLock,relation,true,root,"INSERT INTO transfers(from_account_id,to_account_id,amount)VALUES(1,2,10)RETURNING*;",28312
bank,psql,,2421,ExclusiveLock,transactionid,true,root,"INSERT INTO transfers(from_account_id,to_account_id,amount)VALUES(1,2,10)RETURNING*;",28312
```

这个查询把「谁在等锁」和「谁占着锁」的信息都给你列出来了，重点看这几列：

|列名|含义|
|-|-|
|blocked_pid|被阻塞的进程 ID（这里是 28254）|
|blocking_pid|正在持有冲突锁的进程 ID（这里是 28312）|
|blocked_statement|阻塞进程正在执行、需要等锁才能继续的 SQL（SELECT … FOR UPDATE）|
|current_statement_in_blocking_process|占锁进程当前执行的 SQL（INSERT INTO transfers）|

从结果看，冲突在哪？

1. 阻塞原因

    进程 28254 在做：

    ```sql
    SELECT * 
    FROM accounts 
    WHERE id = 1 
    FOR UPDATE;
    ```

    为了保证外键一致性，PostgreSQL 不仅要在 accounts.id=1 那一行上加行锁（tuple‐level ExclusiveLock），还要对所有「引用了这个 account、但还没提交」的事务拿一把 ShareLock on transactionid。

    这把 ShareLock 具体是针对事务 ID = 2421 的锁（也就是第一条 INSERT 的那笔事务）。

2. 冲突点

    占锁进程 28312（就是第一次做 INSERT INTO transfers 的那个会话，它还没提交）已经对自己的事务 ID = 2421 拿了 排他锁（ExclusiveLock on transactionid）。

    而 28254 又想对同一个事务 ID（2421）拿共享锁，排他锁 vs 共享锁冲突，就只能等。

---

***鎖衝突的原因與總結***

1. 原因與脈絡：

    进程 28312 的 INSERT INTO transfers…

    在 transfers 表上取得了 RowExclusiveLock（表级写锁），

    并且在它自己的事务号 XID=2421 上自动打了一把 ExclusiveLock on transactionid。

    进程 28254 的 SELECT … FOR UPDATE

    在 accounts.id=1 那一行上取得了一个 tuple‐level ExclusiveLock（行级排他锁），

    同时因为外键一致性要对所有引用了 accounts.id=1 且还未提交的事务做检查，于是在 XID=2421 上申请了一把 ShareLock on transactionid。

    由于 ExclusiveLock on transactionid（28312 持有）与 ShareLock on transactionid（28254 请求）互斥，28254 的锁请求被阻塞（granted = false），只能等 28312 提交或回滚后才能继续。

2. 為什麼进程 28254 的 SELECT … FOR UPDATE 知道要跟 XID=2421 上申请一把 ShareLock on transactionid ？

    PostgreSQL 在初始化父表的锁时，会查看该表上所有与外键相关的子表约束（从系统目录 pg_constraint 里读出来）。

    然后对于每个引用此行的、仍处于“in-progress”状态的事务 ID，它就向锁管理器去 LockAcquire(..., ShareLock)。

    “引用此行” 中的“此行”其实指的是你在父表中那条被 SELECT … FOR UPDATE 锁定的记录（例如 accounts.id = 1 那一行）。当你对它加锁时，PostgreSQL 会去看所有子表（如 transfers）里有外键指向这条父表记录的行，看哪些事务正对这些子表行做插入或更新但还没结束。

    那些“正在对 transfers.from_account_id = 1 做插入／更新，却还没提交 (COMMIT) 或回滚 (ROLLBACK)”的事务，它们的事务编号（XID）就被视为“引用此行的、仍处于 in-progress 的事务 ID”。

    PostgreSQL 会对这些 XID 申请一个 ShareLock on transactionid，以等这些事务先完成，保证外键的一致性。

    那个事务 ID，恰好就是正在插入 transfers、并引用了 accounts.id=1 的事务的 XID（2421）。

    这个设计保证了外键参照完整性：任何对子表的未决修改，必须在你真正拿到父表行锁之前，先行完成，才能避免“半提交”的数据相互冲突。

3. 半提交是什麼？

    “半提交”（或称“in-flight”）指的就是事务已经对数据库做了修改，但还没走到提交（COMMIT）那一步。

    在这段时间里，那些修改对其他事务的可见性受 MVCC 快照规则控制，可能看得到也可能看不到，最重要的是它随时可能被回滚。

    因此，如果不在 transactionid 层面做锁，就可能出现“子表的外键行已经插入，但父表刚好在这时被改了，最终导致外键指向不存在或错误的键值”这种不一致情况。

    通过“等待所有引用此行的 in-flight 事务先完成”这样的 ShareLock on transactionid 机制，PostgreSQL 就能在锁定父表行前，避免任何尚未提交的子表修改破坏外键完整性。

4. RowExclusiveLock、ExclusiveLock on transactionid、tuple‐level ExclusiveLock、ShareLock on transactionid 的作用和用途是什麼？

    1. RowExclusiveLock（表级写锁）
        粒度：整张表（relation lock）。

        何时申请：任何对表中数据做写操作时，都会自动获得，比如 INSERT、UPDATE、DELETE；也包括 SELECT … FOR UPDATE/SHARE 前的写准备。

        主要作用：

        保证写操作期间表结构（schema）不被 DDL（ALTER TABLE、DROP TABLE 等）干扰。

        允许高并发的写入（多会话可以同时对不同的行执行写操作），但阻塞需要更高等级锁（如 AccessExclusiveLock）的操作。

    2. tuple-level ExclusiveLock（行级排他锁）
        粒度：单行（tuple lock）。

        何时申请：

        显式：SELECT … FOR UPDATE

        隐式：UPDATE、DELETE 时也会对受影响行自动加此锁。

        主要作用：

        确保同一行在一个事务修改（或预锁）期间，其他事务无法再次修改或取得同等锁。

        为编程者提供悲观锁机制，保证读－改－写操作的原子性和可见性。

    3. ExclusiveLock on transactionid（事务级排他锁）
        粒度：某个活动事务的内部标识（transaction ID）。

        何时申请：任何执行写操作的事务在开始后都会自动对自身 XID 申请此锁，并一直持有到提交或回滚。

        主要作用：

        标记“事务 X 正在进行中”，阻止其他事务即时对它做外键一致性检查或死锁检测时抢锁。

        确保引用此事务所写入数据的其他事务，必须等它结束后才能继续，避免出现半提交（in-flight）数据。

    4. ShareLock on transactionid（事务级共享锁）
        粒度：某个活动事务的内部标识（transaction ID）。

        何时申请：

        由 SELECT … FOR UPDATE（或其它需要交叉事务一致性检查的操作）隐式申请，针对所有“已引用目标父表行且尚未提交”的事务 XID。

        也用于死锁检测中，让等待关系通过此锁体现。

        主要作用：

        等待并检查“事务 X 是否已经完成（commit/rollback）”，保证外键或并发可见性一致性。

        与 ExclusiveLock on transactionid 互斥：如果事务 X 还没结束（持有排他锁），申请共享锁的一方就必须阻塞，直到 X 完成。

    兼容性摘要

    |锁|同类型共存|与 ExclusiveLock on transactionid 共存|与 ShareLock on transactionid 共存|
    |-|-|-|-|
    |RowExclusiveLock|可与自身共存|可|可|
    |tuple-level ExclusiveLock|不可与自身共存|——（不适用）|——（不适用）|
    |ExclusiveLock on transactionid|不可与自身共存|不可|不可|
    |ShareLock on transactionid|可与自身共存|不可|可|

### Solutions

1. 第一種方法： SELECT FOR NO KEY UPDATE

    SELECT … FOR UPDATE 在 accounts.id=1 那一行上取得了一个 tuple‐level ExclusiveLock（行级排他锁），同时因为外键一致性要对所有引用了 accounts.id=1 且还未提交的事务做检查，于是在 XID=2421 上申请了一把 ShareLock on transactionid。

    關鍵在於 PostgreSQL 預設在 SELECT … FOR UPDATE 時，會假設主鍵欄位（那些有唯一索引、可被外鍵參照的欄位）有可能被修改，
因此除了在該行上加行級排他鎖（FOR UPDATE），還要在 transaction‐ID 層級為外鍵檢查做額外的鎖定，才不會在跨表的外鍵一致性檢查時出現半提交的資料。

    我們看我們的 schema

    ```sql
    ALTER TABLE "transfers" ADD FOREIGN KEY ("from_account_id") REFERENCES "accounts" ("id");

    ALTER TABLE "transfers" ADD FOREIGN KEY ("to_account_id") REFERENCES "accounts" ("id");
    ```

    transfers.from_account_id 和 transfers.to_account_id 有參照 accounts.id

    我們再看看我們的 UpdateAccount 語句

    ```sql
    -- name: UpdateAccount :one
    UPDATE accounts
    set balance = $2
    WHERE id = $1
    RETURNING *;
    ```

    我們更新的是 accounts.balance 並不是外鍵 accounts.id ，如果能「告訴」Postgres：這次鎖定只是為了更新某些非鍵欄位，主鍵絕對不會變動，那麼它就可以省略那把用於外鍵檢查的 transaction‐ID 鎖（或與之相關的 Key-Share 鎖）。

    我們可以用 FOR NO KEY UPDATE 「告訴」Postgres 我要修改的只有「不參與唯一索引或外鍵參照」的欄位（例如 balance），不是主鍵／唯一鍵欄位。

    把 GetAccountForUpdate 語句改成

    ```sql
    -- name: GetAccountForUpdate :one
    SELECT * FROM accounts
    WHERE id = $1 LIMIT 1
    FOR NO KEY UPDATE;
    ```

    在執行 `make sqlc` regenerate golang code for this query

2. 第二種方法： UPDATE accounts …
    在 db/query/account.sql 加入

    ```sql
    -- name: AddAccountBalance :one
    UPDATE accounts
    set balance = balance + sqlc.arg(amount)
    WHERE id = sqlc.arg(id)
    RETURNING *;
    ```

    regenerate golang code exec `make sqlc`

    为何这么写能解决问题？

    避免 SELECT … FOR UPDATE 的 transaction-ID 锁

    SELECT … FOR UPDATE 不仅会在父表行上加一个行级排他锁，还会对所有引用该行且尚未提交的子表事务 XID 申请一个 ShareLock on transactionid，以保证外键一致性。

    这个 ShareLock 和子表事务本身持有的 transaction-ID ExclusiveLock 互斥，就会引发死锁。

    直接 UPDATE 只需行级锁

    UPDATE accounts SET balance = balance + … WHERE id = … 只会对这行做一个普通的行级排他锁（tuple-level ExclusiveLock），并且不会触发任何针对 transaction-ID 的锁申请。

    因此，不会再因外键检查去抢子表事务的 XID 锁，自然也就不会被 BLOCKED，granted = false。
3. SELECT FOR NO KEY UPDATE VS UPDATE accounts …

    | 方法| 优点| 缺点|
    | - | - | - |
    | `SELECT … FOR NO KEY UPDATE` + 后续 `UPDATE` | - 对非键字段加“轻量”排它锁，不会拦截子表的 KeyShareLock，能有效避开外键引起的死锁<br/>- 依然能拿到旧值 | - 只有 PostgreSQL 9.5+ 支持<br/>- 仍需两步：先 SELECT 再 UPDATE，事务稍复杂 |
    | 直接 `UPDATE … RETURNING`（如 SQLC 生成的 `AddAccountBalance`）| - 一步原子：只要更新并拿回新值<br/>- 只会使用行级排他锁，不会牵扯 transaction-ID 锁，最不易死锁<br/>- 代码简单易维护 | - 无法在 UPDATE 前拿到“旧余额”做复杂计算（只能在业务里自己算）|

    首选直接 UPDATE … RETURNING（即你给出的 AddAccountBalance），因为它最简单、最不容易因外键 transaction-ID 锁而死锁。
