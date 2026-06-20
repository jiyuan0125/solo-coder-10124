package utils

import (
	"crypto/sha256"
	"encoding/hex"
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

func BashQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func MatchWildcard(pattern, name string) bool {
	if pattern == "" {
		return true
	}
	if name == "" {
		name = "default"
	}
	patternParts := strings.Split(pattern, "*")
	if len(patternParts) == 1 {
		return pattern == name
	}
	if !strings.HasPrefix(name, patternParts[0]) {
		return false
	}
	current := name[len(patternParts[0]):]
	for i := 1; i < len(patternParts); i++ {
		part := patternParts[i]
		if part == "" {
			continue
		}
		idx := strings.Index(current, part)
		if idx == -1 {
			return false
		}
		current = current[idx+len(part):]
	}
	return true
}

func StableFingerprint(data interface{}) string {
	normalized := normalizeForFingerprint(data)
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", normalized)))
	return hex.EncodeToString(h.Sum(nil))
}

func normalizeForFingerprint(data interface{}) interface{} {
	if data == nil {
		return nil
	}
	v := reflect.ValueOf(data)
	switch v.Kind() {
	case reflect.String:
		s := v.String()
		s = strings.ToLower(s)
		s = strings.Join(strings.Fields(s), " ")
		return s
	case reflect.Slice, reflect.Array:
		var items []interface{}
		for i := 0; i < v.Len(); i++ {
			items = append(items, normalizeForFingerprint(v.Index(i).Interface()))
		}
		return items
	case reflect.Map:
		type kv struct {
			K string
			V interface{}
		}
		var pairs []kv
		for _, key := range v.MapKeys() {
			pairs = append(pairs, kv{
				K: strings.ToLower(fmt.Sprint(key.Interface())),
				V: normalizeForFingerprint(v.MapIndex(key).Interface()),
			})
		}
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].K < pairs[j].K
		})
		var result []interface{}
		for _, p := range pairs {
			result = append(result, []interface{}{p.K, p.V})
		}
		return result
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return normalizeForFingerprint(v.Elem().Interface())
	default:
		return data
	}
}
