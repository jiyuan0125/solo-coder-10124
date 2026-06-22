package app

import (
	"testing"

	"github.com/bjesus/pipet/common"
	"github.com/stretchr/testify/assert"
)

func TestFilterBlocks_LeadingComma(t *testing.T) {
	e := &common.PipetApp{
		Blocks: []common.Block{
			{Name: "weather"},
			{Name: "news"},
			{Name: "other"},
		},
	}

	FilterBlocks(e, ",weather")

	assert.Equal(t, 1, len(e.Blocks))
	assert.Equal(t, "weather", e.Blocks[0].Name)
}

func TestFilterBlocks_TrailingComma(t *testing.T) {
	e := &common.PipetApp{
		Blocks: []common.Block{
			{Name: "weather"},
			{Name: "news"},
			{Name: "other"},
		},
	}

	FilterBlocks(e, "weather,")

	assert.Equal(t, 1, len(e.Blocks))
	assert.Equal(t, "weather", e.Blocks[0].Name)
}

func TestFilterBlocks_MultiPattern(t *testing.T) {
	e := &common.PipetApp{
		Blocks: []common.Block{
			{Name: "a"},
			{Name: "b"},
			{Name: "c"},
		},
	}

	FilterBlocks(e, "a,b")

	assert.Equal(t, 2, len(e.Blocks))
	assert.Equal(t, "a", e.Blocks[0].Name)
	assert.Equal(t, "b", e.Blocks[1].Name)
}

func TestFilterBlocks_EmptyPattern_NoChange(t *testing.T) {
	e := &common.PipetApp{
		Blocks: []common.Block{
			{Name: "a"},
			{Name: "b"},
		},
	}

	FilterBlocks(e, "")

	assert.Equal(t, 2, len(e.Blocks))
}

func TestFilterBlocks_OnlyCommas_NoChange(t *testing.T) {
	e := &common.PipetApp{
		Blocks: []common.Block{
			{Name: "a"},
			{Name: "b"},
		},
	}

	FilterBlocks(e, ",,,")

	assert.Equal(t, 2, len(e.Blocks))
}

func TestFilterBlocks_NonexistentAndValid(t *testing.T) {
	e := &common.PipetApp{
		Blocks: []common.Block{
			{Name: "nonexistent"},
			{Name: "weather"},
			{Name: "news"},
		},
	}

	FilterBlocks(e, "nonexistent,weather")

	assert.Equal(t, 2, len(e.Blocks))
	assert.Equal(t, "nonexistent", e.Blocks[0].Name)
	assert.Equal(t, "weather", e.Blocks[1].Name)
}

func TestFilterBlocks_Wildcard(t *testing.T) {
	e := &common.PipetApp{
		Blocks: []common.Block{
			{Name: "weather"},
			{Name: "weekly"},
			{Name: "news"},
		},
	}

	FilterBlocks(e, "w*")

	assert.Equal(t, 2, len(e.Blocks))
	assert.Equal(t, "weather", e.Blocks[0].Name)
	assert.Equal(t, "weekly", e.Blocks[1].Name)
}
