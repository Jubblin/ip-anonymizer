# ip-anonymizer

Anonymize IPv4 addresses in JSON files while keeping pseudonyms stable across runs.

`ip-anonymizer` is a small Go CLI for scrubbing network flow exports, API payloads, and log dumps before sharing them outside trusted environments. It walks JSON recursively, replaces IPv4 addresses (including those embedded in URLs and messages), and stores a local mapping file so `10.1.2.3` always becomes the same fake address next time you run it.

Supports single JSON documents and JSONL (one JSON object per line).

## Features

- **Consistent pseudonyms** — the same real IP maps to the same fake IP across executions
- **Embedded IP detection** — replaces addresses inside strings such as `https://203.0.113.45/health`
- **JSON and JSONL** — auto-detects multi-line flow exports (e.g. GoFlow/IPFIX dumps)
- **Secret mapping file** — created on first run, chmod `0600`, gitignored by default
- **Container image** — published to GitHub Container Registry on release

## Quick start

### Docker (recommended)

```bash
docker pull ghcr.io/jubblin/ip-anonymizer:latest

docker run --rm \
  -v "$(pwd):/data" \
  -v ip-anonymizer-mapping:/mapping \
  ghcr.io/jubblin/ip-anonymizer:latest \
  -input /data/flows.json \
  -output /data/flows.anon.json \
  -mapping /mapping/mapping.json
```

Mount a persistent volume or host path for `/mapping`. That file is the secret that makes anonymization reversible.

### From source

```bash
git clone https://github.com/Jubblin/ip-anonymizer.git
cd ip-anonymizer
go build -o ip-anonymizer .

./ip-anonymizer -input flows.json -output flows.anon.json
```

Requires Go 1.24 or later.

## Usage

```bash
# Write anonymized output to a file
ip-anonymizer -input data.json -output data.anon.json

# Write to stdout
ip-anonymizer -input data.json

# Store the mapping file outside the repo (recommended)
ip-anonymizer -input data.json \
  -output data.anon.json \
  -mapping ~/.config/ip-anonymizer/mapping.json

# Overwrite the input file
ip-anonymizer -input data.json -in-place
```

### CLI reference

| Flag | Default | Description |
|------|---------|-------------|
| `-input` | *(required)* | Path to the input JSON or JSONL file |
| `-output` | stdout | Path for anonymized output |
| `-mapping` | `.ip-anonymizer-mapping.json` | Secret mapping file (created if missing) |
| `-in-place` | `false` | Replace the input file (cannot be combined with `-output`) |

### Example

**Input** (`events.json`):

```json
{
  "events": [
    {
      "ip": "203.0.113.45",
      "note": "hit https://203.0.113.45/health"
    },
    {
      "ip": "10.1.2.3",
      "note": "retry from 10.1.2.3"
    }
  ]
}
```

**Output** (first run):

```json
{
  "events": [
    {
      "ip": "198.51.100.1",
      "note": "hit https://198.51.100.1/health"
    },
    {
      "ip": "198.51.100.2",
      "note": "retry from 198.51.100.2"
    }
  ]
}
```

A second run against new data containing `203.0.113.45` still maps it to `198.51.100.1`.

### JSONL / flow exports

Files with one JSON object per line (common for GoFlow and similar exporters) are detected automatically. Blank lines are skipped.

```bash
ip-anonymizer -input goflow2.json -output goflow2.anon.json \
  -mapping ~/.config/ip-anonymizer/goflow-mapping.json
```

## How it works

1. Parse the input as JSON. If multiple top-level values are found, fall back to JSONL line-by-line processing.
2. Recursively walk every string value in the document.
3. Match IPv4 addresses with a word-boundary regex.
4. Look up each address in the mapping file, or allocate a new fake IP from the RFC 5737 documentation range `198.51.0.0/16`.
5. Write the anonymized output and persist any new mappings.

Fake IPs are allocated sequentially:

- `198.51.100.1` – `198.51.100.254` first
- then `198.51.1.1`, `198.51.1.2`, … up to ~65,000 unique addresses

Only string values are modified. JSON keys, numbers, booleans, and `null` are left unchanged.

## Mapping file (secret)

On first run, `ip-anonymizer` creates a mapping file at the path given by `-mapping` (default: `.ip-anonymizer-mapping.json` in the current working directory).

```json
{
  "version": 1,
  "next": 3,
  "mappings": {
    "10.1.2.3": "198.51.100.2",
    "203.0.113.45": "198.51.100.1"
  }
}
```

### Treat this file as a credential

The mapping is **reversible**. Anyone with the file can recover the original IP addresses.

- Do **not** commit it to git (patterns are listed in `.gitignore`)
- Do **not** share it publicly or store it in untrusted backups
- Do **not** mount it read-only into shared CI artifacts

For regular use, keep it outside your repository:

```bash
mkdir -p ~/.config/ip-anonymizer
ip-anonymizer -input data.json \
  -mapping ~/.config/ip-anonymizer/mapping.json \
  -output data.anon.json
```

The file is created with mode `0600` (owner read/write only).

## Container

Images are built and published to [GitHub Container Registry](https://github.com/Jubblin/ip-anonymizer/pkgs/container/ip-anonymizer) when changes land on `main` or when a `v*` tag is pushed.

```bash
docker pull ghcr.io/jubblin/ip-anonymizer:latest
```

### Run with Docker

```bash
docker run --rm \
  -v "$(pwd):/data" \
  -v ip-anonymizer-mapping:/mapping \
  ghcr.io/jubblin/ip-anonymizer:latest \
  -input /data/flows.json \
  -output /data/flows.anon.json \
  -mapping /mapping/mapping.json
```

| Mount | Purpose |
|-------|---------|
| `$(pwd):/data` | Input and output files |
| `ip-anonymizer-mapping:/mapping` | Persistent secret mapping (named volume) |

You can replace the named volume with a host path, e.g. `-v ~/.config/ip-anonymizer:/mapping`.

The image runs as UID `65532` and exposes `/mapping` as a volume with correct ownership for writes.

### Image tags

| Trigger | Tags published |
|---------|----------------|
| Push to `main` | `latest`, `main`, commit SHA |
| Tag `v1.2.3` | `1.2.3`, `1.2`, `1` |

GHCR packages are private by default. To make the image public: open the package in GitHub → **Package settings** → **Change visibility**.

## Development

```bash
# Build
go build -o ip-anonymizer .

# Run against a sample file
echo '{"src_addr":"10.0.0.1"}' > /tmp/sample.json
./ip-anonymizer -input /tmp/sample.json -mapping /tmp/test-mapping.json

# Build the container locally
docker build -t ip-anonymizer:local .
```

## Limitations

- **IPv4 only** — IPv6 addresses are not anonymized
- **String values only** — IPs stored as JSON numbers are not matched
- **In-memory JSON** — single JSON documents are loaded fully into memory; JSONL is streamed line by line
- **Pseudonymization, not encryption** — the mapping file can undo anonymization

## Releases

Container images are released automatically via [`.github/workflows/release.yml`](.github/workflows/release.yml).

To cut a versioned release:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This publishes `ghcr.io/jubblin/ip-anonymizer:1.0.0` (and related semver tags).
