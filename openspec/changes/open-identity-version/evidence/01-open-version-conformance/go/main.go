package main

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
)

const schema = `
import "strings"
#V: string & =~"^\\d+\\.\\d+\\.\\d+$"
#IP: {
	ModulePath!: string
	Version!: #V
	Major: "v0"
	VersionMajor: "v" + strings.SplitN(Version, ".", 2)[0]
	VersionMajor: Major
}`

func main() {
	ctx := cuecontext.New()
	s := ctx.CompileString(schema).LookupPath(cue.ParsePath("#IP"))
	cases := map[string]string{
		"open":     `ModulePath: "x", Version: string`,
		"absent":   `ModulePath: "x"`,
		"concrete": `ModulePath: "x", Version: "0.1.0"`,
		"skew":     `ModulePath: "x", Version: "1.0.0"`,
		"badfmt":   `ModulePath: "x", Version: "abc"`,
	}
	for _, k := range []string{"open", "absent", "concrete", "skew", "badfmt"} {
		u := s.Unify(ctx.CompileString(cases[k]))
		e1 := u.Validate()
		e2 := u.Validate(cue.Concrete(true))
		fmt.Printf("== %s\n  plain: %v\n  concrete:", k, e1)
		for _, e := range cueerrors.Errors(e2) {
			fmt.Printf(" [%s]", e.Error())
		}
		fmt.Println()
	}
}
