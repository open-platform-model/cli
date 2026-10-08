package config

import (
	"errors"
	"fmt"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCueErrorUnder(t *testing.T) {
	err := cuecontext.New().CompileString("a: b: c: 1 & 2\n").Validate()
	require.Error(t, err)
	wrapped := fmt.Errorf("building: %w", err)

	assert.True(t, cueErrorUnder(wrapped, "a"))
	assert.True(t, cueErrorUnder(wrapped, "a", "b"))
	assert.True(t, cueErrorUnder(wrapped, "a", "b", "c"))
	assert.False(t, cueErrorUnder(wrapped, "b"), "a selector deeper in the path is not a prefix")
	assert.False(t, cueErrorUnder(wrapped, "a", "c"))
	assert.False(t, cueErrorUnder(wrapped, "a", "b", "c", "d"), "a prefix longer than the path")
	assert.False(t, cueErrorUnder(wrapped), "no prefix names no place")
	assert.False(t, cueErrorUnder(errors.New("a.b.c: conflicting values 1 and 2"), "a"), "a plain error carries no path, whatever its text")
	assert.False(t, cueErrorUnder(nil, "a"))
}
