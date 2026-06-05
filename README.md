# IP Anonymizer for JSON

A Go CLI that anonymizes IPv4 addresses in JSON files with consistent pseudonyms across runs.

## Usage

```bash
go build -o ip-anonymizer .

# Write anonymized JSON to a file
./ip-anonymizer -input data.json -output data.anon.json

# Write to stdout
./ip-anonymizer -input data.json

# Use a mapping file outside the repo (recommended)
./ip-anonymizer -input data.json -mapping ~/.config/ip-anonymizer/mapping.json

# Overwrite the input file in place
./ip-anonymizer -input data.json -in-place
```

## How it works

- Recursively walks all JSON string values
- Detects IPv4 addresses, including those embedded in URLs and log messages
- Replaces each real IP with a fake IP from the RFC 5737 documentation range (`198.51.100.0/24`)
- Persists a mapping file so the same real IP always maps to the same fake IP across executions

## Secret mapping file

On first run, a mapping file is created (default: `.ip-anonymizer-mapping.json` in the current directory).

**This file is a secret.** It is reversible — anyone with the mapping file can recover the original IP addresses. Do not:

- Commit it to git (it is listed in `.gitignore`)
- Share it publicly
- Back it up to untrusted storage

For production use, store the mapping file outside your repository:

```bash
mkdir -p ~/.config/ip-anonymizer
./ip-anonymizer -input data.json -mapping ~/.config/ip-anonymizer/mapping.json
```

The mapping file is created with permissions `0600` (owner read/write only).
