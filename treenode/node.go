package treenode

const (
	Function    = "function"
	IF          = "if"
	SWITCH      = "switch"
	CASE        = "case"
	FOR         = "for"
	WHILE       = "while"
	DOWHILE     = "doWhile"
	INSTRUCTION = "instruction"
	TRY         = "try"
)

type TreeNode struct {
	Type      string     `json:"type"`
	Data      string     `json:"data"`
	Condition string     `json:"condition"`
	Nodes     []TreeNode `json:"nodes"`
}
