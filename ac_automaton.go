// Package gomultifast package allows for creation and searching of an Aho-Corasick search trie (Automaton).
package gomultifast

import (
	"errors"
	"fmt"
)

const patternMaxLength = 5000
const patternStartingLength = 2000

// Automaton contains the Aho-Corasick trie
type Automaton struct {
	root          *node   // The root of the Aho-Corasick trie
	allNodes      []*node // Pointers to all nodes
	open          bool    // Automaton status. If false, no more patterns can be added
	currentNode   *node   // Pointer to current node while searching
	position      int     // The last searched position in a chunk.
	basePosition  int     // Position of the current chunk related to whole input text
	totalPatterns int     // Total patterns in the automaton
}

// MatchCallback defines the callback used to handle matches
type MatchCallback func(Match, string, string) bool

// Add a Pattern (search term and identifier) to an Automaton
func (a *Automaton) Add(pattern *pattern) (int, error) {
	if !a.open {
		return -1, errors.New("Error: Closed")
	}
	if len(pattern.Pstring) == 0 {
		return -1, errors.New("Error: Zero pattern")
	}
	if len(pattern.Pstring) > patternMaxLength {
		return -1, errors.New("Error: Pattern too long")
	}

	n := a.root
	var next *node

	// Break down string into runes and store in nodes
	for _, alpha := range pattern.Pstring {
		// Does this alpha exist in the trie?
		next = n.findNext(alpha)
		if next != nil {
			n = next
			continue
		} else {
			// The alpha was not found, create a node that leads to it
			next = n.createNext(alpha)
			next.depth = n.depth + 1
			n = next
			// Register a pointer to the node in the automaton
			a.registerNode(n)
		}
	}

	// If we got this far and the last node is set as "final", a duplicate has been encountered.
	if n.final {
		return -1, errors.New("Error: Duplicate pattern")
	}
	n.final = true
	n.registerPattern(pattern)
	a.totalPatterns++
	return 0, nil
}

// NumberOfNodes returns the number of nodes in the automaton
func (a *Automaton) NumberOfNodes() int {
	return len(a.allNodes)
}

// Add a pointer to a node to the automaton registry
func (a *Automaton) registerNode(node *node) {
	a.allNodes = append(a.allNodes, node)
}

// NewAutomaton makes a brand new trie/automaton and set the root node
func NewAutomaton() *Automaton {
	a := Automaton{}
	a.root = nodeCreate()

	a.registerNode(a.root)
	a.reset()

	a.totalPatterns = 0
	a.open = true
	return &a
}

// NewPattern creates a new Pattern consisting of an identifier string and a string containing the search term
func NewPattern(Ident string, Pstring string) *pattern {
	p := pattern{Ident: Ident, Pstring: Pstring}
	return &p
}

// Reset the automata. This makes the next search begin from the root node.
func (a *Automaton) reset() {
	a.currentNode = a.root
	a.basePosition = 0
}

/*
Automaton finalization step.
The slice alphas is here the prefix of a search term in the trie.
Looks for a situation where the given node has an edge that leads to an
alpha that exists elsewhere in the trie and points to that node if found.
*/
func (a *Automaton) setFailure(node *node, alphas []rune) {
	for i := 1; i < node.depth; i++ {
		m := a.root
		for j := i; j < node.depth && m != nil; j++ {
			m = m.findNext(alphas[j])
		}
		if m != nil {
			node.failureNode = m
			break
		}
	}
	if node.failureNode == nil {
		node.failureNode = a.root
	}
}

/*
Automaton finalization step.
Traverse the outgoing edges of a node in a DFS manner starting from the root.
Makes a recursive call to reach the end of the "trie branch".
At each iteration, look for a "failure node", i.e. the node where the search
can continue if the end of a trie branch is hit.
*/
func (a *Automaton) traverseSetfailure(n *node, alphas []rune) {
	var next *node

	for _, edge := range n.outgoing {
		alphas[n.depth] = edge.alpha
		next = edge.next
		// Look for failure node
		a.setFailure(next, alphas)
		// Recursive call reach next node in branch
		a.traverseSetfailure(next, alphas)
	}
}

/*
Finalize puts the automaton in a state where no more search terms can be added.
Failure nodes are added and node edges are sorted to allow for binary search.
*/
func (a *Automaton) Finalize() {
	alphas := make([]rune, patternStartingLength)

	a.traverseSetfailure(a.root, alphas)

	for _, node := range a.allNodes {
		a.collectAllMatchedPatterns(node)
		node.sortEdges()
	}
	// Close the automaton to prevent further additions.
	a.open = false
}

/*
If a node has a failure node, add the failure node's
matched patterns to the given node's matched patterns.
*/
func (a *Automaton) collectAllMatchedPatterns(node *node) {
	m := node.failureNode
	for m != nil {
		for _, mp := range m.matchedPatterns {
			node.registerPattern(&mp)
		}

		if m.final {
			node.final = true
		}
		m = m.failureNode
	}
}

/*
Search through "text" in a finished automaton. If a match is found,
a user defined callback function is called with the searched text, and string parameter in "param".
if keepSearching is true, the search will continue to the rest of the text after a match.
*/
func (a *Automaton) Search(text string, keepSearching bool, callback MatchCallback, param string) (bool, error) {
	position := 0
	var current *node
	var next *node
	var match Match

	if a.open == true {
		// automaton not ready for some reason
		return false, errors.New("Automaton not ready. Must be Finalized.")
	}

	if !keepSearching {
		a.reset()
	}

	current = a.currentNode

	// Search for the string in text, character by character
	for position < len(text) {
		next = current.binarySearchNext(rune(text[position]))
		if next == nil {
			if current.failureNode != nil {
				current = current.failureNode
			} else {
				position++
			}
		} else {
			current = next
			position++
		}

		if current.final && next != nil {
			/* We check 'next' to find out if we came here after a alphabet
			 * transition or due to a fail. in second case we should not report
			 * matching because it was reported in previous node */
			match.position = position + a.basePosition
			match.Patterns = current.matchedPatterns
			// Match found run callback
			cbRes := callback(match, text, param)
			if cbRes {
				return true, nil
			}
		}
		if position >= len(text) {
			break
		}
	}

	// Save status variables in case we want to keep searching
	a.currentNode = current
	a.basePosition += position

	return false, nil
}

/*
SearchConcurrent searches through "text" in a finished automaton. If a match is found,
a user defined callback function is called with the searched text, and string parameter in "param".
This function does not support using the state of the automaton, but can start at any position in the text input.
*/
func (a *Automaton) SearchConcurrent(text string, position int, callback MatchCallback, param string) (bool, error) {
	var current *node
	var next *node
	var match Match
	basePosition := 0

	if a.open == true {
		// automaton not ready for some reason
		return false, errors.New("Automaton not ready. Must be Finalized.")
	}

	current = a.root

	// Search for the string in text, character by character
	for position < len(text) {
		next = current.binarySearchNext(rune(text[position]))
		if next == nil {
			if current.failureNode != nil {
				current = current.failureNode
			} else {
				position++
			}
		} else {
			current = next
			position++
		}

		if current.final && next != nil {
			/* We check 'next' to find out if we came here after a alphabet
			 * transition or due to a fail. in second case we should not report
			 * matching because it was reported in previous node */
			match.position = position + basePosition
			match.Patterns = current.matchedPatterns
			// Match found run callback
			cbRes := callback(match, text, param)
			if cbRes {
				return true, nil
			}
		}
		if position >= len(text) {
			break
		}
	}

	return false, nil
}

// Print out the automaton for debugging purposes.
func (a *Automaton) Print() {
	var sid pattern

	fmt.Printf("---------------------------------\n")
	for _, n := range a.allNodes {
		var fid int
		if n.failureNode != nil {
			fid = n.failureNode.id
		} else {
			fid = 1
		}
		fmt.Printf("NODE(%3d)/----fail----> NODE(%3d)\n", n.id, fid)
		for _, e := range n.outgoing {
			fmt.Printf(" |----(")
			fmt.Printf("%c)---", e.alpha)
			fmt.Printf("--> NODE(%3d)\n", e.next.id)
		}
		if len(n.matchedPatterns) != 0 {
			fmt.Printf("Accepted patterns: {")
			for j := 0; j < len(n.matchedPatterns); j++ {
				sid = n.matchedPatterns[j]
				if j != 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%s", sid.Ident)
			}
			fmt.Printf("}\n")
		}
		fmt.Printf("---------------------------------\n")
	}
}
