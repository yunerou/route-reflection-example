package muxrouter

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// TypeToSchema describes a reflect.Type as a JSON skeleton for lightweight docs.
//   - struct  -> {fieldName: <schema>} (json tag honored; empty struct -> null)
//   - slice   -> [<element schema>]
//   - pointer -> schema of pointed-to type
//   - map     -> {"map[keyKind]": <value schema>}
//   - other   -> kind name, e.g. "string", "int", "bool"
func TypeToSchema(t reflect.Type) json.RawMessage {
	if t == nil {
		return json.RawMessage("null")
	}
	switch t.Kind() {
	case reflect.Struct:
		if t.NumField() == 0 {
			return json.RawMessage("null")
		}
		var sb strings.Builder
		sb.WriteByte('{')
		first := true
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			name := field.Name
			if jsonTag := field.Tag.Get("json"); jsonTag != "" {
				tagName, _, _ := strings.Cut(jsonTag, ",")
				if tagName == "-" {
					continue
				}
				if tagName != "" {
					name = tagName
				}
			}
			if !first {
				sb.WriteByte(',')
			}
			first = false
			key, _ := json.Marshal(name)
			sb.Write(key)
			sb.WriteByte(':')
			sb.Write(TypeToSchema(field.Type))
		}
		sb.WriteByte('}')
		return json.RawMessage(sb.String())
	case reflect.Slice, reflect.Array:
		var sb strings.Builder
		sb.WriteByte('[')
		sb.Write(TypeToSchema(t.Elem()))
		sb.WriteByte(']')
		return json.RawMessage(sb.String())
	case reflect.Pointer:
		return TypeToSchema(t.Elem())
	case reflect.Map:
		var sb strings.Builder
		sb.WriteByte('{')
		key, _ := json.Marshal(fmt.Sprintf("map[%s]", t.Key().Kind()))
		sb.Write(key)
		sb.WriteByte(':')
		sb.Write(TypeToSchema(t.Elem()))
		sb.WriteByte('}')
		return json.RawMessage(sb.String())
	default:
		val, _ := json.Marshal(t.Kind().String())
		return json.RawMessage(val)
	}
}
