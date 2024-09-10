package server

import (
	"code-parser/treenode"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	sitter "github.com/smacker/go-tree-sitter"
	clang "github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"io"
)

func (s *Server) generateTree(c *gin.Context) {
	language := c.Param("language")
	parser := sitter.NewParser()

	// Set the language for the parser
	switch language {
	case "cpp":
		parser.SetLanguage(cpp.GetLanguage())
	case "c":
		parser.SetLanguage(clang.GetLanguage())
	case "python":
		parser.SetLanguage(python.GetLanguage())
	case "javascript":
		parser.SetLanguage(javascript.GetLanguage())
	default:
		_ = c.AbortWithError(400, fmt.Errorf("language not available: %s", language))
		return
	}

	// Read the source code from the request body
	sourceCode, err := io.ReadAll(c.Request.Body)
	if err != nil {
		_ = c.AbortWithError(400, fmt.Errorf("couldn't read source code from body: %s", err.Error()))
		return
	}

	// Parse the source code
	tree, _ := parser.ParseCtx(context.Background(), nil, sourceCode)
	rootNode := tree.RootNode()

	// Recursively find function nodes and their children
	functionNodes := findFunctionNodes(rootNode, sourceCode)

	// Return the list of functions and their nodes as JSON
	c.JSON(200, functionNodes)
}

func findFunctionNodes(node *sitter.Node, sourceCode []byte) []treenode.TreeNode {
	var functions []treenode.TreeNode
	if isFunctionNode(node.Type()) {
		functionName := node.ChildByFieldName("declarator")
		functionSignature := string(sourceCode[functionName.StartByte():functionName.EndByte()])
		functionNode := treenode.TreeNode{
			Type:  "function",
			Data:  functionSignature,
			Nodes: extractNodes(node.ChildByFieldName("body"), sourceCode),
		}
		functions = append(functions, functionNode)
	}

	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child != nil {
			functions = append(functions, findFunctionNodes(child, sourceCode)...)
		}
	}

	return functions
}

func extractNodes(node *sitter.Node, sourceCode []byte) []treenode.TreeNode {
	var bodyNodes []treenode.TreeNode

	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		switch child.Type() {
		case "compound_statement":
			fmt.Printf("extracting body nodes: %d", node.ChildCount())
			nodes := extractNodes(child, sourceCode)
			for j := 0; j < len(nodes); j++ {
				bodyNodes = append(bodyNodes, nodes[j])
			}
		case "return_statement":
			bodyNode := treenode.TreeNode{
				Type:  treenode.INSTRUCTION,
				Data:  string(sourceCode[child.StartByte():child.EndByte()]),
				Nodes: nil,
			}
			bodyNodes = append(bodyNodes, bodyNode)
		case "expression_statement":
			bodyNode := treenode.TreeNode{
				Type:  treenode.INSTRUCTION,
				Data:  string(sourceCode[child.StartByte():child.EndByte()]),
				Nodes: nil,
			}
			bodyNodes = append(bodyNodes, bodyNode)
		case "for_statement":
			declaration := child.ChildByFieldName("initializer")
			condition := child.ChildByFieldName("condition")
			update := child.ChildByFieldName("update")
			newCondition := string(sourceCode[declaration.StartByte():condition.EndByte()])
			if update != nil {
				newCondition = string(sourceCode[declaration.StartByte():update.EndByte()])
			}
			bodyNode := treenode.TreeNode{
				Type:      treenode.FOR,
				Condition: newCondition,
				Nodes:     []treenode.TreeNode{},
			}
			nodes := extractNodes(child.ChildByFieldName("body"), sourceCode)
			for j := 0; j < len(nodes); j++ {
				bodyNode.Nodes = append(bodyNode.Nodes, nodes[j])
			}
			bodyNodes = append(bodyNodes, bodyNode)
		case "while_statement":
			condition := child.ChildByFieldName("condition").ChildByFieldName("value")
			newCondition := string(sourceCode[condition.StartByte():condition.EndByte()])
			bodyNode := treenode.TreeNode{
				Type:      treenode.WHILE,
				Condition: newCondition,
				Nodes:     []treenode.TreeNode{},
			}
			nodes := extractNodes(child.ChildByFieldName("body"), sourceCode)
			for j := 0; j < len(nodes); j++ {
				bodyNode.Nodes = append(bodyNode.Nodes, nodes[j])
			}
			bodyNodes = append(bodyNodes, bodyNode)
		case "do_statement":
			condition := child.ChildByFieldName("condition")
			newCondition := string(sourceCode[condition.StartByte():condition.EndByte()])
			bodyNode := treenode.TreeNode{
				Type:      treenode.DOWHILE,
				Condition: newCondition,
				Nodes:     []treenode.TreeNode{},
			}
			nodes := extractNodes(child.ChildByFieldName("body"), sourceCode)
			for j := 0; j < len(nodes); j++ {
				bodyNode.Nodes = append(bodyNode.Nodes, nodes[j])
			}
			bodyNodes = append(bodyNodes, bodyNode)
		case "declaration":
			bodyNode := treenode.TreeNode{
				Type:  treenode.INSTRUCTION,
				Data:  string(sourceCode[child.StartByte():child.EndByte()]),
				Nodes: nil,
			}
			bodyNodes = append(bodyNodes, bodyNode)
		case "if_statement":
			condition := child.ChildByFieldName("condition").ChildByFieldName("value")
			conditionString := string(sourceCode[condition.StartByte():condition.EndByte()])
			bodyNode := treenode.TreeNode{
				Type:  treenode.IF,
				Data:  conditionString,
				Nodes: []treenode.TreeNode{},
			}
			nodes := extractNodes(child, sourceCode)
			for j := 0; j < len(nodes); j++ {
				nodes[j].Condition = "true"
				bodyNode.Nodes = append(bodyNode.Nodes, nodes[j])
			}
			alternative := child.ChildByFieldName("alternative")
			if alternative != nil {
				nodes := extractNodes(alternative, sourceCode)
				for j := 0; j < len(nodes); j++ {
					nodes[j].Condition = "false"
					bodyNode.Nodes = append(bodyNode.Nodes, nodes[j])
				}
			}
			bodyNodes = append(bodyNodes, bodyNode)
		case "case_statement":
			condition := child.ChildByFieldName("value")
			conditionString := "default"
			if condition != nil {
				conditionString = string(sourceCode[condition.StartByte():condition.EndByte()])
			}
			bodyNode := treenode.TreeNode{
				Type:      treenode.CASE,
				Condition: conditionString,
				Nodes:     []treenode.TreeNode{},
			}
			nodes := extractNodes(child, sourceCode)
			for j := 0; j < len(nodes); j++ {
				bodyNode.Nodes = append(bodyNode.Nodes, nodes[j])
			}
			bodyNodes = append(bodyNodes, bodyNode)
		case "switch_statement":
			condition := child.ChildByFieldName("condition").ChildByFieldName("value")
			conditionString := string(sourceCode[condition.StartByte():condition.EndByte()])
			bodyNode := treenode.TreeNode{
				Type:  treenode.SWITCH,
				Data:  conditionString,
				Nodes: []treenode.TreeNode{},
			}
			nodes := extractNodes(child, sourceCode)
			for j := 0; j < len(nodes); j++ {
				bodyNode.Nodes = append(bodyNode.Nodes, nodes[j])
			}
			bodyNodes = append(bodyNodes, bodyNode)
		//case "try_statement":
		//	condition := child.ChildByFieldName("condition").ChildByFieldName("value")
		//	conditionString := string(sourceCode[condition.StartByte():condition.EndByte()])
		//	bodyNode := treenode.TreeNode{
		//		Type:  treenode.SWITCH,
		//		Data:  conditionString,
		//		Nodes: []treenode.TreeNode{},
		//	}
		//	nodes := extractNodes(child, sourceCode)
		//	for j := 0; j < len(nodes); j++ {
		//		bodyNode.Nodes = append(bodyNode.Nodes, nodes[j])
		//	}
		//	bodyNodes = append(bodyNodes, bodyNode)
		default:
			fmt.Println("unhandled: " + node.Type())
		}
	}

	return bodyNodes
}

func isFunctionNode(nodeType string) bool {
	return nodeType == "function_definition" || nodeType == "function_declaration"
}
