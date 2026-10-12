# Postgres Architecture :
    follows a client <=> server model
    clients (application / API / psql CLI) connect to the server process (Postmaster) via TCP (default port 5432) or a Unix socket
    The Postmaster forks a new `Backend Process` for every client connection (process per connection, not threads)
        - each backend parses, plans and executes the SQL of that one connection
        - this is why connections are expensive and connection pools (pgbouncer / pgxpool) are used
    When we start Postgres it also starts several background processes

    `Background Processes`: 1. Checkpointer
                            2. Background Writer
                            3. WAL (Write Ahead Log) Writer
                            4. Autovacuum Launcher (spawns Autovacuum Workers)

                            (All above work with Disk Storage)

                            5. Archiver (copies completed WAL files to an archive / secondary storage, used for backups & PITR)
                            6. Logger (writes server logs)
                            7. WAL Sender / WAL Receiver (only when Replication is configured)

                        - Backend Processes (not background processes) are 1 per active connection and execute SQL statements
                        - Background processes handle `Vacuuming` (autovacuum) and `Checkpointing` (checkpointer)
                        - Replication is handled by WAL Sender (primary) and WAL Receiver (replica), not by the checkpointer
                        - WAL Writer + Checkpointer work for persistence and recovery
                        - Background Writer writes dirty pages to disk ahead of time so backends always find free buffers

    `Shared Memory` (shared by all processes):
                        - Shared Buffers = cache of data pages in RAM (all reads/writes go through here first)
                        - WAL Buffers = WAL records waiting to be written to the WAL file

    - The `Data Cluster` is the data directory on disk (PGDATA) managed by one Postgres server, it contains
                                - Databases, Tables, Schemas, Indexes, Write Ahead Logs (pg_wal folder), config files

# Data Storage Model :
        - Every Table in Postgres is stored as 1 or more Heap files (split into 1GB segment files)
            ` A heap file is a physical representation of the logical table`
            - each table also has a Free Space Map (FSM) and Visibility Map (VM) file
        - Each Heap file is a collection of Pages / Blocks (typically size of 8KB)
        - Each Page can hold multiple rows of data, a page contains :
            - Page Header (metadata about the page, e.g. LSN of last WAL change)
            - Item Pointers / Line Pointers (array pointing to the actual rows inside the page)
            - Free Space
            - Tuples (rows) stored from the end of the page towards the start
        - The Row is represented as a Tuple which contains a header with
                - ctid = (block_number, tuple_index)  (physical location of the row / tuple)
                - xmin = Transaction Id that created/inserted the row
                - xmax = Transaction Id that updated/deleted (or locked) the row, 0 if still live
        `As these rows / Tuples are in the Heap they are in no fixed order on disk (no clustering by primary key)`
        - Indexes (B-Tree by default) are separate files which store key -> ctid

# TOAST (The Oversized-Attribute Storage Technique) :
        An automatic mechanism for large data values, a row cannot span multiple pages (8KB)
        TOAST kicks in when a row is larger than ~2KB (TOAST_TUPLE_THRESHOLD), not only when it exceeds 8KB
        Data types like TEXT, BYTEA, JSONB, VARCHAR are variable length so they are TOAST-able
        (fixed length types like INT, BIGINT, TIMESTAMP are never toasted)

        2 strategies to handle large data :
            1. Compression ( pglz (default) / lz4 - internal algorithms for compression)
            2. When compression is not enough Postgres moves these big values out of line into a TOAST table associated with the main table
               (main row keeps a pointer to the TOAST table, value is split into ~2KB chunks)

        - Max size of a single field value is 1GB
        - Per column storage strategy : PLAIN, MAIN, EXTERNAL, EXTENDED (default for most variable length types)

# Transaction and MVCC (Multi Version Concurrency Control):
        - When an UPDATE statement is executed Postgres doesn't overwrite the existing row, here's what happens :

            - Existing row, xmin = initial transaction id, xmax = 0
            - when UPDATE query is executed, old row xmin = unchanged, xmax = current transaction id (marking it as old)
                this also locks the row for other writers. Readers can still read this old row in the meantime.
            - a new version of the row is created where, xmin = current transaction id, xmax = 0 (new updated row)
            - the old row's ctid now points to the new updated row (version chain)
            - DELETE only sets xmax, no new row is created
            - Because of this UPDATE = DELETE + INSERT internally, which creates dead tuples (table bloat)
            - HOT (Heap Only Tuple) update : if no indexed column changes and the page has free space,
                the new version is put in the same page and indexes are not updated

            * Readers never block Writers, Writers never block Readers
              Writers DO block other Writers on the same row (second one waits until the first commits / rolls back)
              Reader Transactions read based on their snapshot (which tuples are visible is decided using xmin / xmax)
                  - READ COMMITTED (default) : new snapshot for each statement
                  - REPEATABLE READ / SERIALIZABLE : one snapshot for the whole transaction
              Writing Transactions create a new version of the row without affecting any on-going Reads *

# Vacuum :
        When tuples are dead (deleted / old versions) and not visible to any active Transaction,
        VACUUM marks that space in the table file (on disk) as free so it can be reused by new rows
        - Normal VACUUM does not give the space back to the OS, it only makes it reusable (no table lock)
        - VACUUM FULL rewrites the whole table and returns space to the OS, but takes an exclusive lock
        - Also updates the Visibility Map and Free Space Map
        - Freezes old tuples to prevent `Transaction ID Wraparound` (transaction ids are 32 bit)
        - ANALYZE (often run together with it) updates table statistics used by the query planner
        - Autovacuum runs this automatically based on number of dead tuples

# WAL (Write Ahead Log) :
        Any change to a data page must be recorded first in a log, the WAL. A `Sequential` append-only log on disk (pg_wal)
        On COMMIT only the WAL needs to be flushed (fsync) to disk, the changed data pages can stay in Shared Buffers
        If Postgres crashes before flushing the changed data pages, the WAL is replayed from the last checkpoint to recover
        - Ensures Durability and Performance as writing logs sequentially is much faster than random writes on data pages
        - WAL is also used for Replication (streaming WAL to replicas) and Point In Time Recovery (PITR)

# Checkpointer :
        Flushes modified data pages gradually (not all at once) from Shared Buffers to physical disk
        - Identifies the modified pages (dirty buffers)
        - Writes these pages to disk slowly and continuously to prevent sudden spikes in disk I/O (checkpoint_completion_target)
        - calls fsync so the OS ensures the data is physically written to disk
        - WAL files older than the completed checkpoint are not needed for crash recovery anymore,
          so they are removed / recycled (frees disk space in pg_wal, not memory)
        - Runs every checkpoint_timeout (default 5 min) or when WAL grows beyond max_wal_size, or on manual CHECKPOINT

    After flushing is done it writes a checkpoint record (`Safe Point`) to the `WAL`, so for crash recovery Postgres doesn't need to replay the entire WAL,
    it can start replaying from the last checkpoint.


# Isolation Levels :
            Means : When several transactions run at the same time how much of each other's work they can see
            Example : Bank Account with a balance 100

            `Dirty Read` : You see a transaction's change before it's committed. Transaction A makes the balance 50 then rolls back , but in the meantime Transaction B read that change and works on the changed value, it couldn't know the transaction A was rolled back.

            `Non Repeatable Read` : Reading the same row twice in a single transaction but getting different results . You read balance 100, B commits balance = 50, you read again and its 50

            `Phantom Read` : You run same query Twice but get different number of rows
                            SELECT * FROM accounts WHERE balance > 80 returns 3 rows, B inserts and commits a new matching account, and the same query now returns 4.

            `Serialization Anomaly` : each transaction looks fine on its own , but both scenario should not exist if the transaction happened one after another. 
                            - Rule is atleast 1 doctor must be vacant
                            - Two doctors are on call 
                            - Doctor 1 sees 2 is on call and 2 is 1 also on call
                            - both cut the call and now both of them are vacant

            LEVELS :

                    1. Read Un-Committed :  basically means read before its committed , this causes dirty 
                                            reads. In Postgres it doesn't happen.

                    2. Read Committed (the default): Read after committed, two transactions may see different
                                                    as one reads after another is committed.

                    3. Repeatable Read : Whole transaction sees one Frozen snapshot. No changes can be seen
                                        by you if some else transaction makes a change. Trying to make change
                                        on a row that has already been changed causes a serialization error by the Porstgres and aborts the transaction

                    4. Serializable : Repeatable Read + Postgres looks for a pattern like the Doctors call example
                                       and aborts any 1 transaction so the one - at- a time ordering maintains and your app needs be ready for a retry 