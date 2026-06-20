package utils

import (
	"os"
	"testing"

	"github.com/bjesus/pipet/common"
	"github.com/stretchr/testify/assert" // Or use default `testing` if no external dependency needed
)

// Test GetSeparator
func TestGetSeparator(t *testing.T) {
	app := &common.PipetApp{
		Separator: []string{"-", ":"},
	}
	assert.Equal(t, "-", GetSeparator(app, 0))
	assert.Equal(t, ", ", GetSeparator(app, 5)) // Test default case
}

// Test FileExists
func TestFileExists(t *testing.T) {
	f, _ := os.CreateTemp("", "example")
	defer os.Remove(f.Name()) // Cleanup
	assert.True(t, FileExists(f.Name()))
	assert.False(t, FileExists("nonexistent.file"))
}

func TestRemoveUnnecessaryNesting(t *testing.T) {
	input := [][][]interface{}{{{"foo", "bar"}}}
	expected := []interface{}{"foo", "bar"}
	result := RemoveUnnecessaryNesting(input)
	assert.Equal(t, result, expected)
}

func TestBashQuote(t *testing.T) {
	assert.Equal(t, "hello", BashQuote("hello"))
	assert.Equal(t, "''", BashQuote(""))
	assert.Equal(t, "'hello world'", BashQuote("hello world"))
	assert.Equal(t, "'it'\\''s'", BashQuote("it's"))
	assert.Equal(t, "'$var'", BashQuote("$var"))
	assert.Equal(t, "'`cmd`'", BashQuote("`cmd`"))
	assert.Equal(t, "'line1\nline2'", BashQuote("line1\nline2"))
}

func TestStableFingerprint(t *testing.T) {
	data1 := []interface{}{"a", "b", "c"}
	data2 := []interface{}{"a", "b", "c"}
	assert.Equal(t, StableFingerprint(data1), StableFingerprint(data2))

	map1 := map[string]interface{}{"b": 2, "a": 1}
	map2 := map[string]interface{}{"a": 1, "b": 2}
	assert.Equal(t, StableFingerprint(map1), StableFingerprint(map2))

	nested1 := []interface{}{
		map[string]interface{}{"name": "alice", "age": 30.0},
		map[string]interface{}{"name": "bob", "age": 25.0},
	}
	nested2 := []interface{}{
		map[string]interface{}{"age": 30.0, "name": "alice"},
		map[string]interface{}{"age": 25.0, "name": "bob"},
	}
	assert.Equal(t, StableFingerprint(nested1), StableFingerprint(nested2))

	s1 := "hello   world"
	s2 := "hello world"
	assert.Equal(t, StableFingerprint([]interface{}{s1}), StableFingerprint([]interface{}{s2}))

	s3 := "hello"
	s4 := "world"
	assert.NotEqual(t, StableFingerprint([]interface{}{s3}), StableFingerprint([]interface{}{s4}))
}
