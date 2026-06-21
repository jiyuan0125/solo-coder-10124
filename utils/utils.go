package utils

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/bjesus/pipet/common"
)

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

func BashEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func StableFingerprint(data interface{}) string {
	normalized := normalize(data)
	jsonBytes, err := json.Marshal(normalized)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(jsonBytes)
	return fmt.Sprintf("%x", hash)
}

func normalize(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Map:
		m := make(map[string]interface{})
		keys := val.MapKeys()
		sortedKeys := make([]string, 0, len(keys))
		keyMap := make(map[string]reflect.Value)
		for _, k := range keys {
			ks := fmt.Sprintf("%v", k.Interface())
			sortedKeys = append(sortedKeys, ks)
			keyMap[ks] = k
		}
		sort.Strings(sortedKeys)
		for _, ks := range sortedKeys {
			m[ks] = normalize(val.MapIndex(keyMap[ks]).Interface())
		}
		return m
	case reflect.Slice, reflect.Array:
		s := make([]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			s[i] = normalize(val.Index(i).Interface())
		}
		return s
	case reflect.Ptr, reflect.Interface:
		if val.IsNil() {
			return nil
		}
		return normalize(val.Elem().Interface())
	case reflect.String:
		s := val.String()
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.ReplaceAll(s, "\t", " ")
		s = strings.Join(strings.Fields(s), " ")
		return s
	case reflect.Bool:
		return val.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint()
	case reflect.Float32, reflect.Float64:
		return val.Float()
	default:
		return fmt.Sprintf("%v", v)
	}
}
