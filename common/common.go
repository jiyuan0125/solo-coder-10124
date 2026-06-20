package common

type Block struct {
	Type     string
	Command  interface{}
	Queries  []string
	NextPage string
	Name     string
}

type PipetApp struct {
	Blocks    []Block
	Data      []interface{}
	MaxPages  int
	Separator []string
	CSVHeader []string
	BlockName string
}
