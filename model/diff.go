package model

type DiffConfig struct {
	Left       string
	Right      string
	Delimiter  rune
	WithHeader bool
	Output     bool
}
type DiffContent struct {
	Content string
	Type    string
}

const (
	JsonType = "json"
	XmlType  = "xml"
)
