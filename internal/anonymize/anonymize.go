package anonymize

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/richardw/ip-anonymizer/internal/mapping"
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
			return fmt.Errorf("decode JSON: multiple top-level values")
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
