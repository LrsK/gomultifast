/*
The gomultifast package allows for creation and searching of an Aho-Corasick search trie (Automaton).
*/
package gomultifast

import (
	"errors"
	"fmt"
)

const PATTERN_MAX_LENGTH = 5000
const PATTERN_STARTING_LENGTH = 2000

// The Aho-Corasick trie (automaton)
type Automaton struct {
	root           *Node   // The root of the Aho-Corasick trie
	all_nodes      []*Node // Pointers to all nodes
	automaton_open bool    // Automaton status. If false, no more patterns can be added
	current_node   *Node   // Pointer to current node while searching
	position       int     // The last searched position in a chunk.
	base_position  int     // Position of the current chunk related to whole input text
	total_patterns int     // Total patterns in the automaton
}

/*
Type that defines the callback used to handle matches
*/
type MatchCallback func(Match, string, string) bool

/*
Add a Pattern (search term and identifier) to an Automaton
*/
func (a *Automaton) Add(pattern *Pattern) (int, error) {
	if !a.automaton_open {
		return -1, errors.New("Error: Closed")
	}
	if len(pattern.Pstring) == 0 {
		return -1, errors.New("Error: Zero pattern")
	}
	if len(pattern.Pstring) > PATTERN_MAX_LENGTH {
		return -1, errors.New("Error: Pattern too long")
	}

	n := a.root
	var next *Node

	// Break down string into runes and store in nodes
	for _, alpha := range pattern.Pstring {
		// Does this alpha exist in the trie?
		next = n.find_next(alpha)
		if next != nil {
			n = next
			continue
		} else {
			// The alpha was not found, create a node that leads to it
			next = n.create_next(alpha)
			next.depth = n.depth + 1
			n = next
			// Register a pointer to the node in the automaton
			a.register_node(n)
		}
	}

	// If we got this far and the last node is set as "final", a duplicate has been encountered.
	if n.final {
		return -1, errors.New("Error: Duplicate pattern")
	}
	n.final = true
	n.register_pattern(pattern)
	a.total_patterns++
	return 0, nil
}

/*
Returns the number of nodes in the automaton
*/
func (a *Automaton) NumberOfNodes() int {
	return len(a.all_nodes)
}

/*
Add a pointer to a node to the automaton registry
*/
func (a *Automaton) register_node(node *Node) {
	a.all_nodes = append(a.all_nodes, node)
}

/*
Make the brand new trie/automaton and set the root node
*/
func NewAutomaton() *Automaton {
	a := Automaton{}
	a.root = node_create()

	a.register_node(a.root)
	a.reset()

	a.total_patterns = 0
	a.automaton_open = true
	return &a
}

/*
Create a new Pattern consisting of an identifier string and a string containing the search term
*/
func NewPattern(Ident string, Pstring string) *Pattern {
	p := Pattern{Ident: Ident, Pstring: Pstring}
	return &p
}

/*
Reset the automata. This makes the next search begin from the root node.
*/
func (a *Automaton) reset() {
	a.current_node = a.root
	a.base_position = 0
}

/*
Automaton finalization step.
The slice alphas is here the prefix of a search term in the trie.
Looks for a situation where the given node has an edge that leads to an
alpha that exists elsewhere in the trie and points to that node if found.
*/
func (a *Automaton) set_failure(node *Node, alphas []rune) {
	for i := 1; i < node.depth; i++ {
		m := a.root
		for j := i; j < node.depth && m != nil; j++ {
			m = m.find_next(alphas[j])
		}
		if m != nil {
			node.failure_node = m
			break
		}
	}
	if node.failure_node == nil {
		node.failure_node = a.root
	}
}

/*
Automaton finalization step.
Traverse the outgoing edges of a node in a DFS manner starting from the root.
Makes a recursive call to reach the end of the "trie branch".
At each iteration, look for a "failure node", i.e. the node where the search
can continue if the end of a trie branch is hit.
*/
func (a *Automaton) traverse_setfailure(node *Node, alphas []rune) {
	var next *Node

	for _, edge := range node.outgoing {
		alphas[node.depth] = edge.alpha
		next = edge.next
		// Look for failure node
		a.set_failure(next, alphas)
		// Recursive call reach next node in branch
		a.traverse_setfailure(next, alphas)
	}
}

/*
Here we finalize the automaton by adding failure nodes and sorting the edges of the nodes to allow for binary search.
No more search terms can be added after this is invoked.
*/
func (a *Automaton) Finalize() {
	alphas := make([]rune, PATTERN_STARTING_LENGTH)

	a.traverse_setfailure(a.root, alphas)

	for _, node := range a.all_nodes {
		a.collect_all_matched_patterns(node)
		node.sort_edges()
	}
	// Close the automaton to prevent further additions.
	a.automaton_open = false
}

/*
If a node has a failure node, add the failure node's
matched patterns to the given node's matched patterns.
*/
func (a *Automaton) collect_all_matched_patterns(node *Node) {
	m := node.failure_node
	for m != nil {
		for _, mp := range m.matched_patterns {
			node.register_pattern(&mp)
		}

		if m.final {
			node.final = true
		}
		m = m.failure_node
	}
}

/*
Search through "text" in a finished automaton. If a match is found,
a user defined callback function is called with the searched text, and string parameter in "param".
if keep_searching is true, the search will continue to the rest of the text after a match.
*/
func (a *Automaton) Search(text string, keep_searching bool, callback MatchCallback, param string) int {
	position := 0
	var current *Node
	var next *Node
	var match Match

	if a.automaton_open == true {
		/* automaton not ready for some reason */
		return -1
	}

	if !keep_searching {
		a.reset()
	}

	current = a.current_node

	// Search for the string in text, character by character
	for position < len(text) {
		var alpha rune = rune(text[position])
		next = current.binary_search_next(alpha)
		if next == nil {
			if current.failure_node != nil {
				current = current.failure_node
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
			match.position = position + a.base_position
			match.Patterns = current.matched_patterns
			// Match found run callback
			cb_res := callback(match, text, param)
			if cb_res {
				return 1
			}
		}
		if position >= len(text) {
			break
		}
	}

	// Save status variables in case we want to keep searching
	a.current_node = current
	a.base_position += position

	return 0
}

/*
Print out the automaton for debugging purposes.
*/
func (a *Automaton) Print() {
	var sid Pattern

	fmt.Printf("---------------------------------\n")
	for _, n := range a.all_nodes {
		var fid int
		if n.failure_node != nil {
			fid = n.failure_node.id
		} else {
			fid = 1
		}
		fmt.Printf("NODE(%3d)/----fail----> NODE(%3d)\n", n.id, fid)
		for _, e := range n.outgoing {
			fmt.Printf(" |----(")
			fmt.Printf("%c)---", e.alpha)
			fmt.Printf("--> NODE(%3d)\n", e.next.id)
		}
		if len(n.matched_patterns) != 0 {
			fmt.Printf("Accepted patterns: {")
			for j := 0; j < len(n.matched_patterns); j++ {
				sid = n.matched_patterns[j]
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
