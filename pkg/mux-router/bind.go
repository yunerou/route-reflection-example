package muxrouter

import (
	"fmt"
	"reflect"
	"strconv"
)

// ParamGetter returns the raw value for a parameter at the given source and the
// presence flag. Adapters supply a getter backed by their request abstraction
// (net/http request, huma.Context, ...).
type ParamGetter func(source ParamSource, name string) (raw string, found bool)

// BindParams builds a fresh ReqParamT, filling its path/query/header tagged
// fields from the supplied getter. It is shared by every adapter so parameter
// binding semantics stay identical across backends.
func BindParams[ReqParamT any](get ParamGetter) (ReqParamT, error) {
	var param ReqParamT
	pv := reflect.ValueOf(&param).Elem()
	pt := pv.Type()
	if pt.Kind() == reflect.Pointer {
		pv.Set(reflect.New(pt.Elem()))
		pv = pv.Elem()
		pt = pt.Elem()
	}
	if pt.Kind() != reflect.Struct {
		return param, nil
	}
	for i := 0; i < pt.NumField(); i++ {
		field := pt.Field(i)
		if field.PkgPath != "" {
			continue
		}
		var (
			raw   string
			found bool
		)
		if name, ok := FieldTagName(field, string(SourcePath)); ok {
			raw, found = get(SourcePath, name)
		} else if name, ok := FieldTagName(field, string(SourceQuery)); ok {
			raw, found = get(SourceQuery, name)
		} else if name, ok := FieldTagName(field, string(SourceHeader)); ok {
			raw, found = get(SourceHeader, name)
		}
		if !found {
			continue
		}
		if err := SetFieldValue(pv.Field(i), raw); err != nil {
			return param, fmt.Errorf("parse parameter %q: %w", field.Name, err)
		}
	}
	return param, nil
}

// SetFieldValue parses raw into v according to v's kind (string/bool/int/uint/float).
func SetFieldValue(v reflect.Value, raw string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		v.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetFloat(f)
	default:
		return fmt.Errorf("unsupported field type %s", v.Type())
	}
	return nil
}

// JSONSchemaType maps a reflect.Kind to a JSON Schema primitive type name,
// used by adapters to document scalar path/query/header parameters.
func JSONSchemaType(k reflect.Kind) string {
	switch k {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	default:
		return "string"
	}
}
