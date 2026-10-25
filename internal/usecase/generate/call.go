package generate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/nguyentin05/spectest/internal/domain"
)

// maxLLMInputChars guards against pathological prompts that exceed the model's
// context window. The pipeline aborts before invoking the LLM.
const maxLLMInputChars = 200_000

// callValidated invokes the LLM with a strict JSON contract and runs the
// provided validator on the decoded value.
//
// On decode or semantic failure it performs exactly one repair call that
// includes the offending output and the validation error. A second failure is
// returned unchanged. The stage name is prefixed to every returned error so
// callers can attribute failures to a pipeline phase.
func callValidated[T any](
	ctx context.Context,
	llm domain.LLMService,
	stage, systemPrompt, userPrompt string,
	validate func(T) error,
) (T, error) {
	var zero T

	if err := checkInputLimit(systemPrompt, userPrompt); err != nil {
		return zero, fmt.Errorf("%s: %w", stage, err)
	}
	if err := ctx.Err(); err != nil {
		return zero, fmt.Errorf("%s: %w", stage, err)
	}

	raw, err := llm.GenerateStructured(ctx, systemPrompt, userPrompt, nil)
	if err != nil {
		return zero, fmt.Errorf("%s: llm call: %w", stage, err)
	}

	value, firstErr := decodeAndValidate[T](raw, validate)
	if firstErr == nil {
		return value, nil
	}

	repairSystem, repairUser := buildRepairPrompt(systemPrompt, userPrompt, raw, firstErr)
	if err := checkInputLimit(repairSystem, repairUser); err != nil {
		return zero, fmt.Errorf("%s: repair: %w", stage, err)
	}
	if err := ctx.Err(); err != nil {
		return zero, fmt.Errorf("%s: repair: %w", stage, err)
	}

	raw2, err := llm.GenerateStructured(ctx, repairSystem, repairUser, nil)
	if err != nil {
		return zero, fmt.Errorf("%s: repair llm call: %w", stage, err)
	}

	value, secondErr := decodeAndValidate[T](raw2, validate)
	if secondErr != nil {
		return zero, fmt.Errorf("%s: after repair: %w", stage, errors.Join(firstErr, secondErr))
	}
	return value, nil
}

func checkInputLimit(system, user string) error {
	if total := len(system) + len(user); total > maxLLMInputChars {
		return fmt.Errorf("input too large: %d chars exceeds limit %d", total, maxLLMInputChars)
	}
	return nil
}

func decodeAndValidate[T any](raw string, validate func(T) error) (T, error) {
	var value T
	if err := decodeStrictJSON([]byte(raw), &value); err != nil {
		return value, fmt.Errorf("decode json: %w", err)
	}
	if validate != nil {
		if err := validate(value); err != nil {
			return value, fmt.Errorf("validate: %w", err)
		}
	}
	return value, nil
}

// decodeStrictJSON rejects unknown fields and any trailing tokens after the
// first JSON value.
func decodeStrictJSON(data []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	if err := dec.Decode(out); err != nil {
		return err
	}

	// A second Decode must return io.EOF; anything else means trailing data.
	var extra json.RawMessage
	if err := dec.Decode(&extra); err == nil {
		return errors.New("multiple JSON values in response")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("trailing data: %w", err)
	}
	return nil
}

func buildRepairPrompt(system, user, previous string, validationErr error) (string, string) {
	var b strings.Builder
	b.WriteString(user)
	b.WriteString("\n\n<previous_response>\n")
	b.WriteString(previous)
	b.WriteString("\n</previous_response>\n\n<validation_errors>\n")
	b.WriteString(validationErr.Error())
	b.WriteString("\n</validation_errors>\n\n")
	b.WriteString("Your previous response was invalid. Return a corrected JSON value " +
		"that satisfies the required schema and the validation errors above. " +
		"Return JSON only.")
	return system, b.String()
}
