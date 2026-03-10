package releaseprocess

import (
	"strings"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/token"

	oerrors "github.com/opmodel/cli/pkg/errors"
)

const fieldNotAllowed = "field not allowed"

func ValidateConfig(schema cue.Value, values []cue.Value, context, name string) (cue.Value, *oerrors.ConfigError) {
	if !schema.Exists() || len(values) == 0 {
		return cue.Value{}, nil
	}

	merged := values[0]
	for _, v := range values[1:] {
		merged = merged.Unify(v)
		if err := merged.Err(); err != nil {
			return cue.Value{}, &oerrors.ConfigError{Context: context, Name: name, RawError: err}
		}
	}

	var combined cueerrors.Error
	combined = walkDisallowed(schema, merged, nil, combined)

	unified := schema.Unify(merged)
	if err := unified.Validate(cue.Concrete(true)); err != nil {
		for _, ce := range cueerrors.Errors(err) {
			f, _ := ce.Msg()
			if f == fieldNotAllowed {
				continue
			}
			combined = cueerrors.Append(combined, ce)
		}
	}

	if combined != nil {
		return cue.Value{}, &oerrors.ConfigError{Context: context, Name: name, RawError: combined}
	}

	return merged, nil
}

func walkDisallowed(schema, val cue.Value, pathPrefix []string, acc cueerrors.Error) cueerrors.Error {
	iter, err := val.Fields(cue.Optional(true))
	if err != nil {
		return acc
	}
	for iter.Next() {
		sel := iter.Selector()
		child := iter.Value()
		fieldPath := append(append([]string{}, pathPrefix...), sel.String())

		if !schema.Allows(sel) {
			acc = cueerrors.Append(acc, &fieldNotAllowedError{pos: child.Pos(), path: fieldPath})
			continue
		}

		if child.IncompleteKind() == cue.StructKind {
			childSchema := schema.LookupPath(cue.MakePath(sel))
			if !childSchema.Exists() {
				continue
			}
			acc = walkDisallowed(childSchema, child, fieldPath, acc)
		}
	}
	return acc
}

type fieldNotAllowedError struct {
	pos  token.Pos
	path []string
}

func (e *fieldNotAllowedError) Position() token.Pos         { return e.pos }
func (e *fieldNotAllowedError) InputPositions() []token.Pos { return nil }
func (e *fieldNotAllowedError) Error() string               { return fieldNotAllowed }
func (e *fieldNotAllowedError) Path() []string {
	return append([]string{"values"}, normalizeFieldPath(e.path)...)
}
func (e *fieldNotAllowedError) Msg() (msg string, args []interface{}) {
	return fieldNotAllowed, nil
}

func normalizeFieldPath(path []string) []string {
	if len(path) == 0 {
		return nil
	}
	joined := strings.Join(path, ".")
	joined = strings.TrimPrefix(joined, "#module.#config.")
	joined = strings.TrimPrefix(joined, "#module.#config")
	joined = strings.TrimPrefix(joined, "#config.")
	joined = strings.TrimPrefix(joined, "#config")
	joined = strings.TrimPrefix(joined, ".")
	if joined == "" {
		return nil
	}
	return strings.Split(joined, ".")
}
