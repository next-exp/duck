# Rate history collection

`rateReader` stores GDC counters and rates in the detector's acquisition database.
It continues updating the existing `rates` run summaries and appends history to
`rateSamples`. No GDC, API or frontend changes are required for collection.

## Deployment

Apply `database/migrations/20261104_add_rate_samples.sql` to the configured
acquisition schema before starting the new binary. For the current deployments,
these schemas are `duck_v2` for NEXT100, `duck` for DEMO and `duck_v1` for HDDemo.
HDDemo uses one database and one collector for both GDCs. Run only one collector
per detector. The program checks that the history table exists at startup; it
does not automatically modify the production schema.

Build with `go build -o bin/rateReader ./rateReader`, then use the existing
configuration and `-period 5` (the default). Deploy only the reader; do not restart
acquisition services. This change does not alter TOPI settings.

## Stored values and timing

Each history row contains the message's run number, GDC name, UTC timestamp with
microsecond precision, cumulative events/bytes, current trigger/byte rates,
whole-run average trigger/byte rates, and a counter-reset flag. Rates use Hz and
bytes/second. The current rate is preserved exactly as published by the GDC; it
is not recomputed into a five-second interval average.

The first valid metric for a run/GDC is queued immediately. Later samples are
selected when their source timestamp is at least `-period` seconds after the
previous selected sample. A detected counter decrease forces an immediate
sample with `counter_reset=true`. Thus spacing follows actual incoming metrics,
not fixed wall-clock bins. No messages means no new history rows: missing
telemetry is not represented as zero or as a fresh copy of an old measurement.
GDC metrics without a positive run number or source timestamp, negative values,
and duplicate/out-of-order messages are rejected or ignored. A delayed message
from an older run cannot replace newer run state for that GDC.

Counters allow a future history API to compute `delta(events)/delta(time)` within
a run/GDC and aggregate time bins. Never difference across a marked reset.
A restart that resets and catches up between messages cannot be detected without
additional source metadata. No historical data is backfilled.

History writes and run-summary updates share a transaction. Source timestamps
and the `(run, gdc, sample_time)` primary key make retries idempotent. A bounded
in-memory queue retains up to 4096 pending samples across temporary DB failures;
new samples beyond this limit are dropped with an explicit log message. Pending
data is lost if the reader crashes or is killed. SIGINT/SIGTERM attempts a final
flush with a ten-second deadline. The queue is not a durable message broker.

The reader preserves its existing summary definition: sums of GDC counters and
published run averages, grouped by each message's run. It skips summary updates
when a known member's measurement is older than three sampling periods. History
can still be saved for those source times. GDCs that have never reported are not
inferred to contribute zero; a future plot must evaluate expected-host coverage.

The additive table can be left in place when rolling back the reader binary.
Keep recorded samples unless their removal is explicitly intended.

## Single-run plots and indexes

The first plot will show the aggregate rate across GDCs. Keep per-GDC rows as the
source data so missing hosts, resets and asynchronous sample times remain
visible to the aggregation logic.

`rateSamples_run_time (run, sample_time)` supports reading one run and an
optional time range without scanning other runs, even after the table grows to
millions of rows. The `(run, gdc, sample_time)` primary key also supports
single-GDC history and idempotent insertion. The intended read shape is:

```sql
SELECT gdc, sample_time, events, bytes, current_trigger_rate,
       current_byte_rate, avg_trigger_rate, avg_byte_rate, counter_reset
FROM rateSamples
WHERE run = ? AND sample_time >= ? AND sample_time < ?
ORDER BY sample_time;
```

A future API should group samples into common time bins, select at most one
current-rate sample per GDC/bin, and sum the selected GDC rates. Do not group by
exact message timestamps: different GDCs publish asynchronously. Return coverage
alongside the total and mark incomplete bins rather than treating absent GDCs
as zero. Counter-derived interval averages must be calculated per GDC before
combining them. Indexing isolates the requested run but does not bound the size
of a very long run; a future API should impose a point budget and aggregate
larger bins, with persisted rollups considered if measurements show a need.

## Validation

Run `go test -race ./rateReader ./pkg/database` and `go vet ./rateReader`.
Use `go test ./api -run '^$'` to check compatibility with API database mocks.
For the real MySQL test, set `RATE_READER_TEST_DSN` to an isolated, disposable
schema and run `go test -race ./rateReader -run TestMySQLHistoryAndRunSummaries`.
That test creates and drops `rates` and `rateSamples`; never point it at a
production schema.
