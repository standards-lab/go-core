package graphtest

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/graph"
)

// refType is the type every node and group handle implements, whatever its
// value type: a field whose type implements it is a handle field.
var refType = reflect.TypeFor[graph.Ref]()

// CheckNames reports, as an error on t, each failure [NameErrors] finds in
// nodes. It does not stop the test, so one run reports every field.
func CheckNames(t testing.TB, nodes any) {
	t.Helper()
	for _, err := range NameErrors(nodes) {
		t.Error(err)
	}
}

// NameErrors checks each exported handle field of nodes, a struct or a
// pointer to one, whose fields are [graph.Node] and [graph.Group] handles
// or [graph.Ref] values. It returns an error for each handle field that
// holds no handle, a nil one or a zero-value literal, and for each whose
// node's or group's name, lowercased with its spaces removed, is not the
// field name lowercased: "meterprovider" and "meter provider" both name a
// MeterProvider field. Unexported fields and fields of other types are
// skipped. A nodes that is not a struct or a non-nil pointer to one is a
// single error. NameErrors returns nil when every handle field passes.
func NameErrors(nodes any) []error {
	v := reflect.ValueOf(nodes)
	if v.Kind() == reflect.Pointer && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return []error{fmt.Errorf("graphtest: %T is not a struct or a pointer to one", nodes)}
	}
	var errs []error
	for i := range v.NumField() {
		f := v.Type().Field(i)
		if !f.IsExported() || !f.Type.Implements(refType) {
			continue
		}
		fv := v.Field(i)
		if isNil(fv) {
			errs = append(errs, fmt.Errorf("%s.%s: no handle", v.Type().Name(), f.Name))
			continue
		}
		name := fv.Interface().(graph.Ref).Name()
		if name == "" {
			// Only a zero-value literal has the empty name; Define,
			// Value, Fresh, and Group reject it.
			errs = append(errs, fmt.Errorf("%s.%s: no handle", v.Type().Name(), f.Name))
			continue
		}
		if strings.ReplaceAll(strings.ToLower(name), " ", "") != strings.ToLower(f.Name) {
			errs = append(errs, fmt.Errorf("%s.%s: named %q", v.Type().Name(), f.Name, name))
		}
	}
	return errs
}

// isNil reports whether v, a handle field, holds no value: a nil pointer
// or a nil interface.
func isNil(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		return v.IsNil()
	}
	return false
}
