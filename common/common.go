package common

type Block struct {
	Type     string
	Name     string
	Command  interface{}
	Queries  []string
	NextPage string
}

type PipetApp struct {
	Blocks           []Block
	Data             []interface{}
	MaxPages         int
	Separator        []string
	CSVHeader        []string
	StableFingerprint bool
}
