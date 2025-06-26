# 避免潛在的 db deadlock

## 測試有沒有可能發生 db deadlock

### 目前的測試情境

我們做了 10 次同時間從 A 帳戶轉帳 10 塊錢到 B 帳戶，測試通過

但是我們還沒測試 A 轉帳給 B ，同時 B 轉帳給 A 的情境

所以我們可以來模擬一次 A 轉帳給 B ，同時 B 轉帳給 A

### 測試 A 轉帳給 B ，同時 B 轉帳給 A

開啟兩個 pgql console

console1

```sql
BEGIN;
UPDATE accounts SET balance=balance-10 WHERE id=1 RETURNING*;
```

console2

```sql
BEGIN;
UPDATE accounts SET balance=balance-10 WHERE id=2 RETURNING*;
```

console1

```sql
UPDATE accounts SET balance=balance+10 WHERE id=2 RETURNING*;
```

這時這句 UPDATE 就會被 blocked

console2

```sql
UPDATE accounts SET balance=balance+10 WHERE id=1 RETURNING*;
```

發生 deadlock ，輸出

```sh
ERROR:  deadlock detected
DETAIL:  Process 28254 waits for ShareLock on transaction 2811; blocked by process 28312.
Process 28312 waits for ShareLock on transaction 2812; blocked by process 28254.
HINT:  See server log for query details.
CONTEXT:  while updating tuple (0,1) in relation "accounts"
```

所以 A 轉帳給 B ，同時 B 轉帳給 A 是會發生 deadlock 的

## 怎麼排除 A 轉帳給 B ，同時 B 轉帳給 A 發生的 deadlock

### 理解 deadlock 產生的原因

這回的死結與外鍵無關，純粹是兩個交易 (A、B) 在「同一張表的兩列」上拿鎖次序相反造成的典型交叉等待。

1. 步驟拆解

    | 時間線 | 交易 A (pid 28312, xid 2737) | 交易 B (pid 28254, xid 2738) |
    | --- | --- | --- |
    | ①   | `UPDATE accounts SET balance = balance-10 WHERE id = 1;`<br>→ **鎖住 id = 1** 的 tuple-level ExclusiveLock | `UPDATE accounts SET balance = balance-10 WHERE id = 2;`<br>→ **鎖住 id = 2** 的 tuple-level ExclusiveLock |
    | ②   | `UPDATE accounts SET balance = balance+10 WHERE id = 2;`<br>→ 想鎖 id = 2，但發現它已被 B 鎖住，所以<br> 嘗試對 **xid = 2738** 申請 **ShareLock on transactionid** 來「等 B 結束」 | `UPDATE accounts SET balance = balance+10 WHERE id = 1;`<br>→ 想鎖 id = 1，但被 A 鎖住，所以<br> 嘗試對 **xid = 2737** 申請 **ShareLock on transactionid** |
    | ③   | A 等 B 釋放 2738 | B 等 A 釋放 2737 |
    | ④   | **環狀等待 → PostgreSQL 偵測死結**，挑一方 (這裡是 B) 當犧牲者回滾 | A 被保留，B 回滾，鎖釋放 |

    DETAIL: Process 28254 waits for ShareLock on transaction 2737 …
    Process 28312 waits for ShareLock on transaction 2738 …

    ShareLock on transactionid 這裡不是外鍵檢查，而是 PostgreSQL 等待「持有那筆行鎖的交易」結束的內部機制；行鎖衝突時就是用「對持鎖方的 XID 申請 ShareLock」來實現等待。

2. 為什麼光更新 balance 也會用到 transactionid 鎖？

    行鎖 (tuple-level ExclusiveLock) 只標在資料列上。

    若另一交易已鎖住那列，PostgreSQL 必須等它完成；等待方式就是：

    1. 對對方 XID 申請一把 ShareLock on transactionid；

    2. 直到對方把自己的 ExclusiveLock on transactionid 釋放（commit/rollback），等待方才會被喚醒。

    因此訊息裡仍然看到 “ShareLock on transaction xxxx”，但這次純屬 行鎖等待，不是外鍵。

### 怎麼避免上拿鎖次序相反造成的典型交叉等待

固定鎖序：先更新較小的 id、再更新較大的 id（或固定其它排序），所有程式遵守同一順序，就不會交叉等待。

### 測試固定鎖序能不能排除 deadlock

開啟兩個 pgql console

console1

```sql
BEGIN;
UPDATE accounts SET balance=balance-10 WHERE id=1 RETURNING*;
```

console2

```sql
BEGIN;
UPDATE accounts SET balance=balance+10 WHERE id=1 RETURNING*;
```

此時 console2 的 UPDATE 被 blocked ，因為 accounts.id=1 被鎖住了

console1

```sql
UPDATE accounts SET balance=balance+10 WHERE id=2 RETURNING*;
COMMIT;
```

console1 COMMIT 之後， console2 的 UPDATE 就不會被 blocked

console2

```sql
UPDATE accounts SET balance=balance-10 WHERE id=2 RETURNING*;
COMMIT;
```

轉帳完成沒有發生 deadlock ，確定固定鎖序可以排除 deadlock

### 實現固定鎖序 in golang

如果 FromAccountID < ToAccountId

則 FromAccountID 先扣款， ToAccountId 再入金

如果 FromAccountID > ToAccountId

則 ToAccountId 先入金， FromAccountID 再扣款

這樣就能實現每次都是先更新 AccountId 小的再更新 AccountId 大的
