package nodetree

import (
	"strings"
)

// Node implements a single node in a recursive tree.
// For space reasons no values are stored, only the fact that there was a value.
// Not thread-safe.
type Node struct {
	Key      string
	next     map[string]*Node
	HasValue bool
}

// NewNode should be used to create a node.
func NewNode(key string) *Node {
	return &Node{Key: key}
}

// Count returns the number of sub-nodes.
func (n *Node) Count() int {
	if n.next == nil {
		return 0
	}
	return len(n.next)
}

// Children returns the sub-nodes.
func (n *Node) Children() map[string]*Node {
	return n.next
}

// getNext retrieves a subnode.
func (n *Node) getNext(key string) *Node {
	if n.next == nil {
		return nil
	}
	return n.next[key]
}

// setNext adds a subnode.
func (n *Node) setNext(key string, value *Node) {
	if n.next == nil {
		n.next = make(map[string]*Node)
	}
	n.next[key] = value
}

func splitPath(key *string) []string {
	return strings.FieldsFunc(*key, func(c rune) bool {
		return c == '/'
	}) // no strings.Split() to avoid empty tokens
}

// GetNode retrieves a node by path.
func (n *Node) GetNode(path string) *Node {
	str := splitPath(&path)
	root := n
	for _, el := range str {
		root = root.getNext(el)
		if root == nil {
			break
		}
	}
	return root
}

// AddNode adds a new node by path.
func (n *Node) AddNode(path string) *Node {
	str := splitPath(&path)
	root := n
	for _, el := range str {
		next := root.getNext(el)
		if next == nil {
			next = NewNode(el)
			root.setNext(el, next)
		}
		root = next
	}
	root.HasValue = true
	return root
}

// DeleteNode removes a node by path.
func (n *Node) DeleteNode(path string) {
	str := splitPath(&path)
	if len(str) == 0 {
		return
	}
	root := n
	next := root
	var base *Node
	var baseKey string
	var lastKey string
	for i, el := range str {
		root = next
		next = root.getNext(el)
		if next == nil {
			return
		}
		if (next.next == nil || len(next.next) <= 1) && !next.HasValue {
			if base == nil {
				base = root
				baseKey = el
			}
		} else if i < len(str)-1 {
			base = nil
		}
		lastKey = el
	}
	if base != nil {
		root = base // delete the entire empty subtree
		lastKey = baseKey

	}
	delete(root.next, lastKey)
	if len(root.next) == 0 {
		root.next = nil // invariant: no children = no map
	}
}
