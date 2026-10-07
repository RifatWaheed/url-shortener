how to read a plan node (cost, rows, width, actual, buffers):
Index Scan using urls_short_code_key on urls shows the operation, the index it uses and the table it reads. Common types: Seq Scan, Index Scan, Index Only Scan, Bitmap Heap/Index Scan, Nested Loop, Hash Join, Merge Join, Sort, Aggregate, Limit. 

Example : 
Index Scan using urls_short_code_key on urls  (cost=0.29..8.31 rows=1 width=72) (actual time=0.015..0.016 rows=1 loops=1)
  Index Cond: (short_code = 'abc123'::text)
  Buffers: shared hit=3 read=1

cost=0.29..8.31 === > Values are in arbitrary units, not ms. Baseline is seq_page_cost = 1.0, the cost of 
                    one sequential page read

rows = 1 ===> This is rows emitted by the node after filtering, not rows scanned.

  actual time = 0.015..0.016 rows=1 loops=1 ===> actual time = startup...total in ms
                                                rows = actual rows per loop
                                                loops = how many times the node ran

                                                ** Estimated rows vs Actual rows : A gap of 10x or more means bad statstics , so need to Analyze table;
Buffers: shared hit=3 read=1 ===> shared hit = pages were found in Postgres's shared_buffers cache,
                                                so no disk read.
                                  read = pages from OS cache or disk

READING ORDER : 
Read the tree inside-out and bottom-up. The most-indented node runs first.
Find the node with the biggest actual time (after multiplying by loops).
Check estimated vs actual rows.
Look for Seq Scans on big tables, read counts much higher than hit, temp buffers, and a large Rows Removed by Filter.

why a 1-row table used the index before ANALYZE : 
when reltuples = -1 and the table's file is tiny, the planner assumes the table is at least 10 pages. At that size a Seq Scan costs more than 8.17, so the index scan wins

your stale-stats experiment and what it proved : 
But at plan time the planner checks the file's actual current page count, takes the stale density (2 rows per page), and multiplies it up. So it estimated a few hundred rows, and a Seq Scan over ~60 pages lost to the index. Stale stats aren't frozen stats. What actually goes stale is the per-page density and data distribution

Autovaccum Threshold Formula :
Trigger Threshold = autovacuum_analyze_threshold + (scale_factor * rows)

WHERE lower(short_code) = 'sc42': does the unique index get used? 
Ans : I think the index won't get used in this case as each row needs to be lower cased first,

WHERE short_code LIKE '%42': does the index help?
Ans : I think it doesn't help , because the starting character can be anything there is no clue where to jump to, and as the short_code is indexed as a normal B- tree nodes a prefix scan is also not available so postgres will sequentially scan it.

 An index on users.is_verified, where 95% of rows are true, queried with WHERE is_verified = true: index or seq scan?
Ans : Seq scan , when planner will analyze it seeing it is doing more work jumping memory locations on top of going through 95 percent of the rows a sequential I/O scan would be faster .

You add five more indexes to links. What happens to POST /shorten, and why?
Ans : every index created increases the Write operation cost as, besides writing in the table we need to write in all the indexes as well after each insert