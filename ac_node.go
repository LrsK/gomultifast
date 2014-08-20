package gomultifast

import "sort"

// used as node id number, used to count nodes
var Node_id = 0

// An edge indicating the next node
type Edge struct {
	alpha rune  // Edge alpha. An alpha is a text character in the trie
	next  *Node // Target node of the edge
}

// A pattern to be stored in the trie
type Pattern struct {
	Pstring string // String to add to trie
	Ident   string // String identifier
}

// A node in the trie structure
type Node struct {
	id               int       // Node id
	final            bool      // Is this a "final" node, meaning this node is the endpoint of a search
	failure_node     *Node     // The "failure node", i.e. a node where the search can continue in case of a failed search
	depth            int       // Distance between this node and the root
	matched_patterns []Pattern // Slice of matched patterns at a node
	outgoing         []Edge    // Slice of outgoing edges
}

// Matches with some details
type Match struct {
	Patterns []Pattern // Slice containing matched patterns in the text
	position int       // The end position of matching patterns in the text
}

// Alphabetical implements sort.Interface for []Edge based on alphabetical position of Edge.alpha
type Alphabetical []Edge

func (e Alphabetical) Len() int           { return len(e) }
func (e Alphabetical) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e Alphabetical) Less(i, j int) bool { return e[i].alpha < e[j].alpha }

/*
If found, return next node lead to by an edge with a given alpha
*/
func (nd *Node) find_next(alpha rune) *Node {
	for _, edge := range nd.outgoing {
		if edge.alpha == alpha {
			return edge.next
		}
	}
	return nil
}

/*
If the alpha does not yet exist, return pointer to a new node with an outgoing edge with the given alpha
*/
func (nd *Node) create_next(alpha rune) *Node {
	next := nd.find_next(alpha)
	if next != nil {
		// The edge already exists
		return nil
	}
	// Otherwise register new node and edge
	next = node_create()
	nd.register_outgoing_edge(next, alpha)

	return next
}

/*
Make a new Edge with an alpha and a pointer to a node
*/
func (nd *Node) register_outgoing_edge(next *Node, alpha rune) {
	new_edge := Edge{alpha: alpha, next: next}
	nd.outgoing = append(nd.outgoing, new_edge)
}

/*
Make a new node, and return a pointer to it
*/
func node_create() *Node {
	nd := Node{id: Node_id}
	Node_id++
	return &nd
}

/*
Check if Pattern exists in a given node
*/
func (nd *Node) has_pattern(new_pattern *Pattern) bool {
	for _, mp := range nd.matched_patterns {
		if mp.Pstring == new_pattern.Pstring {
			return true
		}
	}
	return false
}

/*
If pattern doesn't already exist in the node, add it
*/
func (nd *Node) register_pattern(new_pattern *Pattern) {
	// Check if the new pattern already exists in the node
	if nd.has_pattern(new_pattern) {
		return
	}
	nd.matched_patterns = append(nd.matched_patterns, *new_pattern)
}

/*
Sort the outgoing edges of a node alphabetically to enable binary search
*/
func (nd *Node) sort_edges() {
	sort.Sort(Alphabetical(nd.outgoing))
}

/*
Perform a binary search for a given alpha among the outgoing edges of a node
*/
func (nd *Node) binary_search_next(alpha rune) *Node {
	i := sort.Search(len(nd.outgoing), func(i int) bool { return nd.outgoing[i].alpha >= alpha })
	if i < len(nd.outgoing) && nd.outgoing[i].alpha == alpha {
		// alpha was found
		return nd.outgoing[i].next
	} else {
		// alpha was not found
		return nil
	}
}
