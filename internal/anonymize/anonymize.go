package anonymize

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/Jubblin/ip-anonymizer/internal/mapping"
)

var ipv4Pattern = regexp.MustCompile(
	`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`,
)

func AnonymizeString(s string, store *mapping.Store) (string, error) {
	var err error
	result := ipv4Pattern.ReplaceAllStringFunc(s, func(real string) string {
		if err != nil {
			return real
		}
		fake, allocErr := store.Allocate(real)
		if allocErr != nil {
			err = allocErr
			return real
		}
		return fake
	})
	return result, err
}

func WalkJSON(v any, store *mapping.Store) (any, error) {
	switch value := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, child := range value {
			transformed, err := WalkJSON(child, store)
			if err != nil {
				return nil, err
			}
			out[key] = transformed
		}
		return out, nil
	case []any:
		out := make([]any, len(value))
		for i, child := range value {
			transformed, err := WalkJSON(child, store)
			if err != nil {
				return nil, err
			}
			out[i] = transformed
		}
		return out, nil
	case string:
		return AnonymizeString(value, store)
	default:
		return value, nil
	}
}

func ProcessFile(inputPath, outputPath, mappingPath string) error {
	store, err := mapping.Load(mappingPath)
	if err != nil {
		return err
	}

	input, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer input.Close()

	decoder := json.NewDecoder(input)
	decoder.UseNumber()

	var root any
	if err := decoder.Decode(&root); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			if _, err := input.Seek(0, io.SeekStart); err != nil {
				return fmt.Errorf("rewind input for JSONL: %w", err)
			}
			return processJSONL(input, outputPath, store)
		}
		return fmt.Errorf("decode JSON: %w", err)
	}

	transformed, err := WalkJSON(root, store)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(transformed); err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}

	if err := store.Save(); err != nil {
		return err
	}

	if outputPath == "" {
		_, err = os.Stdout.Write(buf.Bytes())
		return err
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

func processJSONL(input *os.File, outputPath string, store *mapping.Store) error {
	var out io.Writer = os.Stdout
	var output *os.File
	if outputPath != "" {
		f, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("create output: %w", err)
		}
		defer f.Close()
		output = f
		out = f
	}

	scanner := bufio.NewScanner(input)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var record any
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.UseNumber()
		if err := decoder.Decode(&record); err != nil {
			return fmt.Errorf("decode JSONL line %d: %w", lineNum, err)
		}

		transformed, err := WalkJSON(record, store)
		if err != nil {
			return fmt.Errorf("anonymize JSONL line %d: %w", lineNum, err)
		}

		if err := encoder.Encode(transformed); err != nil {
			return fmt.Errorf("encode JSONL line %d: %w", lineNum, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read JSONL: %w", err)
	}

	if output != nil {
		if err := output.Sync(); err != nil {
			return fmt.Errorf("sync output: %w", err)
		}
	}

	return store.Save()
}
