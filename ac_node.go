package gomultifast

import "sort"

// used as node id number, used to count nodes
var nodeID = 0

// edge indicating the next node
type edge struct {
	alpha rune  // Edge alpha. An alpha is a text character in the trie
	next  *node // Target node of the edge
}

// A pattern to be stored in the trie
type pattern struct {
	Pstring string // String to add to trie
	Ident   string // String identifier
}

// A node in the trie structure
type node struct {
	id              int       // Node id
	final           bool      // Is this a "final" node, meaning this node is the endpoint of a search
	failureNode     *node     // The "failure node", i.e. a node where the search can continue in case of a failed search
	depth           int       // Distance between this node and the root
	matchedPatterns []pattern // Slice of matched patterns at a node
	outgoing        []edge    // Slice of outgoing edges
}

// Match contains all found matches with some details
type Match struct {
	Patterns []pattern // Slice containing matched patterns in the text
	position int       // The end position of matching patterns in the text
}

// Alphabetical implements sort.Interface for []Edge based on alphabetical position of Edge.alpha
type Alphabetical []edge

func (e Alphabetical) Len() int           { return len(e) }
func (e Alphabetical) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e Alphabetical) Less(i, j int) bool { return e[i].alpha < e[j].alpha }

/*
If found, return next node lead to by an edge with a given alpha
*/
func (nd *node) findNext(alpha rune) *node {
	for _, edge := range nd.outgoing {
		if edge.alpha == alpha {
			return edge.next
		}
	}
	return nil
}

// If the alpha does not yet exist, return pointer to a new node with an outgoing edge with the given alpha
func (nd *node) createNext(alpha rune) *node {
	next := nd.findNext(alpha)
	if next != nil {
		// The edge already exists
		return nil
	}
	// Otherwise register new node and edge
	next = nodeCreate()
	nd.registerOutgoingEdge(next, alpha)

	return next
}

// Make a new Edge with an alpha and a pointer to a node
func (nd *node) registerOutgoingEdge(next *node, alpha rune) {
	newEdge := edge{alpha: alpha, next: next}
	nd.outgoing = append(nd.outgoing, newEdge)
}

// Make a new node, and return a pointer to it
func nodeCreate() *node {
	nd := node{id: nodeID}
	nodeID++
	return &nd
}

// Check if Pattern exists in a given node
func (nd *node) hasPattern(newPattern *pattern) bool {
	for _, mp := range nd.matchedPatterns {
		if mp.Pstring == newPattern.Pstring {
			return true
		}
	}
	return false
}

// If pattern doesn't already exist in the node, add it
func (nd *node) registerPattern(newPattern *pattern) {
	// Check if the new pattern already exists in the node
	if nd.hasPattern(newPattern) {
		return
	}
	nd.matchedPatterns = append(nd.matchedPatterns, *newPattern)
}

// Sort the outgoing edges of a node alphabetically to enable binary search
func (nd *node) sortEdges() {
	sort.Sort(Alphabetical(nd.outgoing))
}

// Perform a binary search for a given alpha among the outgoing edges of a node
func (nd *node) binarySearchNext(alpha rune) *node {
	i := sort.Search(len(nd.outgoing), func(i int) bool { return nd.outgoing[i].alpha >= alpha })
	if i < len(nd.outgoing) && nd.outgoing[i].alpha == alpha {
		// alpha was found
		return nd.outgoing[i].next
	}
	return nil
}
