package utils

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/bjesus/pipet/common"
)

func BashQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, "$`'\"\\\n\t ") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func FlattenNestedSlices(app *common.PipetApp, data interface{}, level int) string {
	v := reflect.ValueOf(data)

	if v.Kind() != reflect.Slice {
		return fmt.Sprint(data)
	}

	var result []string

	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i).Interface()
		flattened := FlattenNestedSlices(app, elem, level+1)
		result = append(result, flattened)
	}

	sep := GetSeparator(app, level)
	return strings.Join(result, sep)
}

func RemoveUnnecessaryNesting(data interface{}) interface{} {
	for {
		val := reflect.ValueOf(data)
		if val.Kind() == reflect.Slice && val.Len() == 1 {
			firstElem := val.Index(0).Interface()
			if reflect.ValueOf(firstElem).Kind() == reflect.Slice {
				data = firstElem
				continue
			}
		}
		break
	}
	return data
}

func GetSeparator(app *common.PipetApp, depth int) string {
	if depth < len(app.Separator) {
		sep, _ := strconv.Unquote(`"` + app.Separator[depth] + `"`)
		return sep
	}
	return ", "
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func StableFingerprint(data interface{}) string {
	var buf strings.Builder
	writeStableValue(&buf, data)
	return buf.String()
}

func writeStableValue(buf *strings.Builder, v interface{}) {
	switch val := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if val {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case float64:
		buf.WriteString(fmt.Sprintf("%g", val))
	case string:
		buf.WriteString("s:")
		buf.WriteString(normalizeString(val))
	case []interface{}:
		buf.WriteString("[")
		for i, elem := range val {
			if i > 0 {
				buf.WriteString(",")
			}
			writeStableValue(buf, elem)
		}
		buf.WriteString("]")
	case map[string]interface{}:
		buf.WriteString("{")
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if i > 0 {
				buf.WriteString(",")
			}
			buf.WriteString(k)
			buf.WriteString(":")
			writeStableValue(buf, val[k])
		}
		buf.WriteString("}")
	default:
		buf.WriteString(fmt.Sprint(v))
	}
}

func normalizeString(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\t", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s
}
