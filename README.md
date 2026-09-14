# Chick

Chick is a Go command-line tool for querying CSV files with SQL. It reads a directory of CSVs, converts them into typed Parquet files using a YAML schema, and loads the result into a local DuckDB table named `data`. Query results can be displayed as a terminal table or exported as JSON.

```text
CSV files + YAML schema → Parquet files → DuckDB → SQL results
```

The repository includes sample LLM gateway usage data and a matching schema, so you can try filtering requests, aggregating token usage, and comparing costs immediately.

## Getting started

You need Go 1.27.1 or later (as declared in `go.mod`) and a working C/C++ toolchain for the DuckDB Go driver's CGO bindings. Make is optional.

From the repository root, build and run:

```sh
go build -o ./bin/chick .

./bin/chick \
  --location=./samples \
  --schema=./samples/schema.yaml \
  --destination=./output/first-run \
  --query="SELECT model, COUNT(*) AS requests FROM data GROUP BY model ORDER BY requests DESC;"
```

Use a new destination for each invocation. Chick creates both timestamped `output_*.parquet` files and a `chick.db` database in that directory. Reusing a destination with an existing `data` table fails, and leftover Parquet files can cause duplicate input to be loaded.

For the bundled Makefile demo:

```sh
make run
```

This builds the binary, **deletes the entire `./output` directory**, and runs a sample query for requests in `us-east-1`.

## Command-line options

| Flag | Default | Description |
| --- | --- | --- |
| `--location` | `./samples` | Directory to search recursively for files ending in `.csv`. |
| `--schema` | Empty | Path to the YAML schema. Required for a successful run. |
| `--destination` | `./output` | Directory for generated Parquet files and `chick.db`; created if needed. |
| `--query` | Empty | SQL query to execute against the `data` table. Supply a query that returns rows. |
| `--format` | `table` | Output format: `table` or uppercase `JSON`. Other values fall back to table output. |

Run `./bin/chick --help` to see the built-in flag descriptions.

Every invocation performs the full CSV → Parquet → DuckDB pipeline before executing its query; there is no query-only mode for an existing database.

## Query examples

These examples use the bundled sample data and separate destinations. Choose fresh destination paths if you run them again.

### Summarize usage and cost by model

```sh
./bin/chick \
  --schema=./samples/schema.yaml \
  --destination=./output/model-summary \
  --query="SELECT model, COUNT(*) AS requests, SUM(total_tokens) AS tokens, ROUND(SUM(cost_usd), 4) AS total_cost_usd FROM data GROUP BY model ORDER BY total_cost_usd DESC;"
```

### Export failed requests as JSON

```sh
./bin/chick \
  --schema=./samples/schema.yaml \
  --destination=./output/failed-requests \
  --format=JSON \
  --query="SELECT request_id, model, status_code, error_reason FROM data WHERE status_code >= 400 LIMIT 20;" \
  > failed-requests.json
```

JSON results are an array of objects keyed by column name. A query with no matching rows currently produces `null`.

## Using your own CSV files

Put CSV files with the same column layout in a directory and define their fields in a YAML schema. Chick skips the first record of each file as a header and maps values to schema fields **by position**, not by header name. Keep the schema's field order and count aligned with every CSV file.

For example, a CSV file:

```csv
created_at,customer,quantity,amount,paid
2026-09-01T12:00:00Z,Acme,3,19.95,true
```

With a matching `schema.yaml`:

```yaml
type: schema
version: v1
spec:
  compression: zstd
  fields:
    - name: created_at
      type: datetime
      encoding: delta
    - name: customer
      type: string
      encoding: dictionary
    - name: quantity
      type: integer
    - name: amount
      type: float
    - name: paid
      type: boolean
```

Run it with:

```sh
./bin/chick \
  --location=./my-csvs \
  --schema=./schema.yaml \
  --destination=./output/my-data \
  --query="SELECT * FROM data;"
```

### Field types and storage options

| Field type | CSV value | Parquet representation |
| --- | --- | --- |
| `string` | Text | String |
| `integer` | Whole number within the signed 32-bit range | 32-bit integer |
| `float` | Decimal number | 32-bit float |
| `boolean` | Exact lowercase `true`; all other values become false | Boolean |
| `datetime` | RFC 3339 timestamp, optionally with fractional seconds | Timestamp with millisecond precision |

Set `spec.compression` to `zstd` or `snappy`; omitting it leaves compression at the library default. Optional per-field encodings are `delta`, `prefix`, and `dictionary`. Choose encodings compatible with the field type, as in the example above and [the bundled schema](samples/schema.yaml).

## Current limitations

- `date` and `double` appear in the Parquet schema builder but do not have CSV value conversion implemented yet. Use the types listed above.
- Input validation is limited. Numeric and timestamp parse errors are not reported, and empty values are not treated as SQL nulls. Validate your data before ingestion.
- Empty CSV files are not handled safely. Each file should contain at least a header, with consistent record widths and a matching schema.
- Table output assumes at least one result row and can panic on an empty result. Use `--format=JSON` when a query may return no rows. Table column order is not guaranteed to match the SQL select list.
- CSV files are read concurrently, with up to 10 readers, and each file is loaded into memory in full. Query results are also collected in memory before rendering. Parquet output is split into files of up to 5,000 rows.
- Individual CSV open/read failures are logged and skipped, so a run can complete with only part of the input loaded.

## Development

| Command | Purpose |
| --- | --- |
| `make build` | Build `./bin/chick`. |
| `make run` | Build, remove `./output`, and execute the sample query. |
| `make clean` | Remove the entire `./output` directory. |
| `go test ./...` | Run Go package checks; no test files are currently included. |

The implementation is split into small packages:

```text
main.go          CLI flags and pipeline orchestration
pkg/csv/         Recursive CSV discovery and concurrent reading
pkg/schema/      YAML schema structures
pkg/parquet/     Type conversion, encoding, and batched Parquet writing
pkg/duckdb/      Database creation and SQL execution
pkg/rendering/   Terminal table and JSON output
samples/         Example CSV files and their schema
```

Generated `bin/` and `output/` directories are ignored by Git.
