# Transaction isolation level & read phenomenon

1. read uncommited isolation level

    造成： dirty read phenomenon ： a 交易讀得到 b 交易的更改

    解釋：開啟 ab 兩個交易，b交易查詢id=1的張戶餘額是100，a交易把id=a的帳戶餘額從100改成90，沒有提交，b交易再查詢id=1的張戶餘額變成90

2. read commited isolation level

    造成： non-repeatable read phenomenon：a 交易讀不到 b 交易的更改，但是 b 交易提交（commit）後看得到 b 交易的更改

    解釋： 開啟 ab 兩個交易，b交易查詢id=1的張戶餘額是100，a交易把id=a的帳戶餘額從100改成90，沒有提交，b交易再查詢id=1的張戶餘額還是100，a提交交易後，b交易再查詢id=1的張戶餘額變成90

    造成： phantom-read phenomenon：a 交易讀不到 b 交易的更改，但是 b 交易提交（commit）後看得到 b 交易的更改

    解釋： 原本有id 1,2,3 帳戶，餘額都是100，開啟 ab 兩個交易，b交易查詢餘額>=100的帳戶結果為id 1,2,3，a交易把id=1的帳戶餘額從100改成90，a提交交易後，b交易再查詢餘額>=100的帳戶結果為id 2,3， id 1 帳戶消失了，因為 id 1 餘額已經被交易 a 改成90，<=100

3. repeatable read level

    造成： serialization anomaly：ab 交易讀不到彼此交易的更改，ab 去讀同一筆資料，ab 寫入相同的資料，則 ab 寫入的相同資料都會生效
    解釋：

    a

    ```sql
    begin;
    set transaction isolation level repeatable read;
    ```

    b

    ```sql
    begin;
    set transaction isolation level repeatable read;
    ```

    a

    ```sql
    select * from accounts where id=1;
    ```

    b

    ```sql
    select * from accounts where id=1;
    ```

    a

    ```sql
    insert into accounts(owner, balance, currency) values ('sum',12,'USD');
    commit;
    ```

    b

    ```sql
    insert into accounts(owner, balance, currency) values ('sum',12,'USD');
    commit;
    ```

    ```sql
    select * from accounts;
    id | owner | balance | currency |          created_at           
    ----+-------+---------+----------+-------------------------------
    7 | sum   |      12 | USD      | 2025-06-11 06:11:50.322658+00
    8 | sum   |      12 | USD      | 2025-06-11 06:10:43.57582+00
    ```

---

mysql 在 serializable isolation level 把 SELECT 轉換成 SELECT FOR SHARE，持有 SELECT FOR SHARE 的交易只允許其他事務讀取行，但不允許更新或刪除行。

---

postgresql 語法

`show transaction isolation level;` 顯示當前 isolation level

`set transaction isolation level [isolation level];` 設定 isolation level

---

<https://www.postgresql.org/docs/current/transaction-iso.html>

PostgreSQL's Read Uncommitted mode behaves like Read Committed. This is because it is the only sensible way to map the standard isolation levels to PostgreSQL's multiversion concurrency control architecture.

## 另一個角度的解釋

以下 4 種「並行異常」是資料庫教科書裡用來說明各隔離級別差異的經典案例。
先看定義，再配合簡短的 SQL 時序範例，最後用一張速查表對比 ANSI 隔離級別允許/避免哪些現象。

1. Dirty Read（髒讀）

    定義：一個交易 讀到了另一個尚未提交的修改。

    風險：若寫入方最後回滾，讀取方就基於「從未真正存在」的資料做決策。

    範例時序

    | 時間線 | Tx A                                        | Tx B                                                  |
    | --- | ------------------------------------------- | ----------------------------------------------------- |
    | 1   | `UPDATE accounts SET balance=0 WHERE id=1;` |                                                       |
    | 2   |                                             | `SELECT balance FROM accounts WHERE id=1;` → **讀到 0** |
    | 3   | `ROLLBACK;`                                 |                                                       |

    Tx B 看見的 0 其實並不存在於永久狀態，這就是髒讀。

2. Non-Repeatable Read（不可重複讀）

    定義：同一個交易內，對同一列重複查詢得到不同結果。

    原因：另一個交易在兩次查詢之間提交了對那列的更新或刪除。

    範例

    |Tx A (長交易) |            Tx B|
    |----------------------    |----------------------------|
    |SELECT salary WHERE id=1;  -- 讀到 100||
    |                           |UPDATE employees SET salary=120 WHERE id=1;<br/>COMMIT;|
    |SELECT salary WHERE id=1;  -- 又讀到 120（變了）| |

3. Phantom Read（幻影讀）

    定義：在同一個交易內，重複執行「範圍/條件查詢」時，行數改變（新的行出現或舊的行消失）。

    實質：另一個交易插入／刪除了滿足條件的新行。

    範例

    |Tx A |                         Tx B|
    |---------------------------  | ------------------------------|
    SELECT COUNT(*) FROM orders WHERE customer_id = 42;  -- 得 5 筆||
    ||INSERT INTO orders ... (customer_id=42);<br/>COMMIT;|
    |SELECT COUNT(*) FROM orders WHERE customer_id = 42;  -- 變成 6 筆（多了一個 phantom）||

4. Serialization Anomaly（序列化異常 / 寫偏差）

    定義：多個交易各自合法，但並行執行後的 整體結果，無法對應到任何「先後順序」的單一序列。

    常見類型：Write-Skew —— 雙方先根據舊快照判斷「條件成立」，再各自插入/更新不同列，造成系統狀態違反商業規則。

    範例（醫院值班排班）

    -- 規則：每晚至少要有 1 名醫師值班

    |Tx A|Tx B|
    |-|-|
    |BEGIN (RR) <br/>SELECT COUNT(*) FROM shifts_tonight WHERE on_call = TRUE;  -- 回 1 認為有人值班，自己可以休息<br/>UPDATE shifts_tonight SET on_call=FALSE WHERE doctor='Alice';<br/>COMMIT;|BEGIN (RR) <br/>SELECT COUNT(*) FROM shifts_tonight WHERE on_call = TRUE;  -- 回 1 也認為有人值班<br/>UPDATE shifts_tonight SET on_call=FALSE WHERE doctor='Bob';<br/>COMMIT;|

    結果：今晚 0 人值班，違反規則；但任何 單一順序（先 A 後 B、或先 B 後 A）都會留下 1 人值班，可見發生了序列化異常。

    ---

    RR = Repeatable Read

    這是 ANSI SQL 定義的 4 個主要隔離級別之一。

    效果：

    交易在 BEGIN 時會拿到一個「固定快照」──之後對同一筆列資料重複查詢，結果不會改變。

    只有「同一列被兩個交易同時修改」(update–update conflict) 時，第二個人 COMMIT 會失敗。

    仍 可能 出現「幻影讀」(phantom) 或「寫偏差 / 序列化異常」。
    在 PostgreSQL 裏，SET TRANSACTION ISOLATION LEVEL REPEATABLE READ; 就是把交易切成 RR，所以大家常簡寫成 “BEGIN (RR)”。

### 哪些隔離級別會出現哪種現象？

| ANSI 隔離級別            | Dirty Read | Non-Repeatable Read | Phantom Read | 可能序列化異常 |
| -------------------- | ---------- | ------------------- | ------------ | ------- |
| **READ UNCOMMITTED** | ✅ 可能       | ✅ 可能                | ✅ 可能         | ✅ 可能    |
| **READ COMMITTED**   | ❌ 避免       | ✅ 可能                | ✅ 可能         | ✅ 可能    |
| **REPEATABLE READ**  | ❌ 避免       | ❌ 避免                | ✅ 可能         | ✅ 可能¹   |
| **SERIALIZABLE**     | ❌ 避免       | ❌ 避免                | ❌ 避免         | ❌ 避免    |

¹ 在 PostgreSQL，Repeatable Read 採 Snapshot Isolation；仍可能出現寫偏差/幻影，需要升到 Serializable（SSI）或顯式鎖才能完全杜絕。

### 小結

- Dirty read：讀到未提交資料。

- Non-repeatable read：同列兩次結果不同。

- Phantom read：查詢集合行數變動。

- Serialization anomaly：系統最終狀態無法對應任何順序執行。

選擇隔離級別時就要決定 哪些風險能接受、哪些必須防堵；若業務邏輯不能容忍寫偏差，就得用 SERIALIZABLE（並實作 40001 重試）或加唯一約束/鎖來強制序列化。

## 如何在 Docker 裡跑 psql 時輸出完整錯誤訊息， 全域／永久 的 SQLSTATE 輸出打開

1. 在容器內直接建立使用者層 ~/.psqlrc

    適用情境：

    你只 sporadically 進到容器裡跑 psql，且每次都是同一個 OS 映像（不需要把設定保存在主機）

    重點指令/步驟：

    ```bash
    docker exec -it mypg bash # 進到容器  
    echo '\set VERBOSITY verbose' >> /root/.psqlrc  
    exit # 之後所有 psql 會自動讀到  
    ```

2. 用 **掛 volume** 的方式把設定檔帶進容器**

    適用情境：

    你想保留設定在本機 repo ；重建/升級容器時仍自動生效

    重點指令/步驟：

    ```bash
    # 在專案目錄放一份 psqlrc
    echo '\set VERBOSITY verbose' > .psqlrc
    # docker-compose.yml
    services:
    db:
        image: postgres:16
        volumes:
        - ./psqlrc:/root/.psqlrc:ro    # 如果容器用 root
    # 或者用非 root 映像 → 改成 /home/postgres/.psqlrc
    ```

3. 指定系統層 psqlrc 或直接用環變數**

    適用情境：

    你不是進 shell，而是在 CI / script 裡跑 `psql -h … -c …`

    重點指令/步驟：

    *A. 在 Dockerfile*

    ```dockerfile
    FROM postgres:16
    RUN echo '\set VERBOSITY verbose' > /usr/local/etc/psqlrc
    ENV PSQLRC=/usr/local/etc/psqlrc
    ```<br>*B. 直接在指令列*<br>```bash
    docker run --rm -e PSQLRC=/etc/psqlrc \
    -v $(pwd)/psqlrc:/etc/psqlrc postgres:16 \
    psql -U postgres -v ON_ERROR_STOP=1 …
    ```

    *B. 直接在指令列*

    ```bash
    docker run --rm -e PSQLRC=/etc/psqlrc \
    -v $(pwd)/psqlrc:/etc/psqlrc postgres:16 \
    psql -U postgres -v ON_ERROR_STOP=1 …
    ```

    > **環境變數 `PSQLRC`** 會覆寫搜尋流程，所以你可以把它指到容器內任何可讀取的檔案；  
    > `psql` 仍會在啟動時自動執行檔案中的 `\set VERBOSITY verbose`。

### 快速驗證

```bash
docker exec -it mypg psql -U postgres -c 'select 1/0;'
```

若設定成功，應該看到

```sh
ERROR:  division by zero
SQLSTATE: 22012
LOCATION:  int4div, int.c:792
```

（verbose 會連路徑和行號一起印；若只想代碼，改 sqlstate。）

### 如何輸出 SQLSTATE 小結

1. 容器並不限制你寫 psqlrc──只是預設沒有而已。

2. 建一個檔案加上 \set VERBOSITY verbose，然後

    放在使用者家目錄（直接編輯或 volume 掛載），或

    指給 PSQLRC，就能讓所有 psql 永久顯示 SQLSTATE。

3. 若只是腳本級需求，也可以在執行時加 -v VERBOSITY=verbose 而不改檔。
