// Package suitefile implements strict YAML codecs and an atomic store for the
// three SpecTest state files: spectest.yaml, spectest.log.yaml, and
// spectest.track.yaml.
package suitefile

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/nguyentin05/spectest/internal/domain"
)

// decodeSuite decodes spectest.yaml strictly and enforces schema version 1.
// Legacy v1 files (metadata/base shape) and any unknown key are rejected rather
// than migrated.
func decodeSuite(data []byte) (*domain.Suite, error) {
	var suite domain.Suite
	if err := decodeStrict(data, &suite); err != nil {
		return nil, fmt.Errorf("%s: invalid schema (expected version: 1): %w", SuiteFileName, err)
	}
	if suite.Version != 1 {
		return nil, fmt.Errorf("%s: expected version: 1, got %d", SuiteFileName, suite.Version)
	}
	return &suite, nil
}

func encodeSuite(suite *domain.Suite) ([]byte, error) {
	return encodeYAML(suite)
}

// decodeLog decodes spectest.log.yaml strictly.
func decodeLog(data []byte) (*domain.LogFile, error) {
	var log domain.LogFile
	if err := decodeStrict(data, &log); err != nil {
		return nil, fmt.Errorf("%s: invalid schema (expected version: 1): %w", LogFileName, err)
	}
	return &log, nil
}

func encodeLog(log *domain.LogFile) ([]byte, error) {
	return encodeYAML(log)
}

// decodeTrack decodes spectest.track.yaml strictly.
func decodeTrack(data []byte) (*domain.TrackFile, error) {
	var track domain.TrackFile
	if err := decodeStrict(data, &track); err != nil {
		return nil, fmt.Errorf("%s: invalid schema (expected version: 1): %w", TrackFileName, err)
	}
	return &track, nil
}

func encodeTrack(track *domain.TrackFile) ([]byte, error) {
	return encodeYAML(track)
}

// decodeStrict rejects unknown fields so typos and legacy schemas fail loudly.
func decodeStrict(data []byte, out any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		return err
	}
	return nil
}

// encodeYAML produces two-space-indented YAML terminated by a newline, without
// a document separator.
func encodeYAML(value any) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(value); err != nil {
		return nil, fmt.Errorf("encode yaml: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("close yaml encoder: %w", err)
	}
	return buf.Bytes(), nil
}
