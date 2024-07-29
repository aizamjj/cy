package layout

import (
	"fmt"

	"github.com/cfoust/cy/pkg/janet"
	"github.com/cfoust/cy/pkg/mux/screen/tree"

	"github.com/charmbracelet/lipgloss"
)

type NodeType interface{}

type PaneType struct {
	Attached bool
	ID       *tree.NodeID
}

type Border struct {
	Style   lipgloss.Border
	Keyword janet.Keyword
}

type SplitType struct {
	Vertical bool
	Percent  *int
	Cells    *int
	Border   *Border
	A        NodeType
	B        NodeType
}

type MarginsType struct {
	Cols   int
	Rows   int
	Frame  *string
	Border *Border
	Node   NodeType
}

type Layout struct {
	root NodeType
}

func New(node NodeType) Layout {
	return Layout{root: node}
}

// getPaneType gets all of the panes that are descendants of the provided node,
// in essence all of the leaf nodes.
func getPaneType(tree NodeType) (panes []PaneType) {
	switch node := tree.(type) {
	case PaneType:
		return []PaneType{node}
	case SplitType:
		panes = append(panes, getPaneType(node.A)...)
		panes = append(panes, getPaneType(node.B)...)
		return
	case MarginsType:
		panes = append(panes, getPaneType(node.Node)...)
		return
	}
	return
}

// isAttached reports whether the node provided leads to a node that is
// attached.
func isAttached(tree NodeType) bool {
	switch node := tree.(type) {
	case PaneType:
		return node.Attached
	case SplitType:
		return isAttached(node.A) || isAttached(node.B)
	case MarginsType:
		return isAttached(node.Node)
	}
	return false
}

func attach(node NodeType, id tree.NodeID) NodeType {
	switch node := node.(type) {
	case PaneType:
		if node.Attached {
			return PaneType{
				Attached: true,
				ID:       &id,
			}
		}

		return node
	case SplitType:
		node.A = attach(node.A, id)
		node.B = attach(node.B, id)
		return node
	case MarginsType:
		node.Node = attach(node.Node, id)
		return node
	}

	return node
}

// Attach changes the currently attached tree node to the one specified by id.
func Attach(layout Layout, id tree.NodeID) Layout {
	return Layout{root: attach(layout.root, id)}
}

// validateTree inspects a tree and ensures that it conforms to all relevant
// constraints, namely there should only be one PaneType with Attached=true.
func validateTree(tree NodeType) error {
	numAttached := 0
	for _, pane := range getPaneType(tree) {
		if pane.Attached != true {
			continue
		}
		numAttached++
	}

	if numAttached > 1 {
		return fmt.Errorf("you may only attach to one pane at once")
	}

	if numAttached == 0 {
		return fmt.Errorf("you must attach to at least one pane")
	}

	return nil
}
