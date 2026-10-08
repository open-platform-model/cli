package inventory

import (
	"github.com/open-platform-model/library/opm/k8s/ownership"
)

// AdmitSet is an explicit set of objects the apply guard admits: the
// library's verdict then lifts its foreign-object refusal for them, and
// nothing else.
type AdmitSet map[ownership.Object]struct{}

// Has reports whether the set admits the object; a nil set admits nothing.
func (s AdmitSet) Has(o ownership.Object) bool {
	_, ok := s[o]
	return ok
}
