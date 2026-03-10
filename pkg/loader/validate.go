package loader

import (
	"cuelang.org/go/cue"

	oerrors "github.com/opmodel/cli/pkg/errors"
	"github.com/opmodel/cli/pkg/releaseprocess"
)

func ValidateConfig(schema, values cue.Value, context, name string) *oerrors.ConfigError {
	_, err := releaseprocess.ValidateConfig(schema, []cue.Value{values}, context, name)
	return err
}
