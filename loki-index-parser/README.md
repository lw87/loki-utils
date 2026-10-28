# Loki TSDB Index Reader

A Go utility for inspecting local TSDB index files produced by Grafana Loki 3.7.9, using schema v13 and a 24h index period.

## Features

- `chunks` (default): lists unique chunk object keys referenced by the index.
- `labels`: lists label names and their associated values.

## How Loki TSDB Indexes Work

Loki groups log entries into streams based on their label sets, writes log data into chunks, and maintains a separate index that maps labels and series to chunk metadata. The index does not contain the actual log lines.

With schema v13 and `period: 24h`, Loki uses daily index periods. Index objects and chunk objects are stored separately.

## Labels and Postings

Labels are key-value pairs that identify a stream. For example:

```text
{app="nginx", environment="prod", region="eu"}
```

A postings list maps a label name and value to the series containing that label. For example, postings for `app=nginx` identify all series whose `app` label has the value `nginx`. Loki uses postings to find matching streams efficiently.

## How the Program Works

### Chunk mode

1. Opens the local TSDB index.
2. Reads label names and values.
3. Uses postings to enumerate series references.
4. Reads each series and its chunk metadata.
5. Uses Loki's schema implementation to construct chunk object keys.
6. Deduplicates chunk references and prints one key per line.

### Labels mode

1. Opens the index.
2. Reads all label names and their values.
3. Prints one `name=value` pair per line.

## Requirements

- A Go version supported by Loki 3.7.9.
- A local TSDB index file.
- Loki v3.4.2 Go module dependencies.

## Build

```bash
go mod init loki-index-reader
go get github.com/grafana/loki/v3@v3.7.9
go mod tidy
go build -o loki-index-reader .
```

Save the program as `main.go` in the same directory before building.

## Usage

### List chunks (default)

```bash
./loki-index-reader ./index-file
```

Specify tenant and schema start date when needed:

```bash
./loki-index-reader -mode chunks -tenant tenant-a -schema-from 2024-01-01 ./index-file
```

### List labels and values

```bash
./loki-index-reader -mode labels ./index-file
```

Example output:

```text
app=nginx
environment=prod
region=eu
```

### Save output to files

```bash
./loki-index-reader ./index-file > chunks.txt
./loki-index-reader -mode labels ./index-file > labels.txt
```

## Output Format

A chunk object key has a structure conceptually similar to:

```text
<tenant-id>/<fingerprint>/<from>:<through>:<checksum>
```

Example:

```text
tenant-a/ab12cd34/18e2c4a0000:18e2c4b0000:12345678
```

