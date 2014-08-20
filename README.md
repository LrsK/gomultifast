# Gomultifast

This is a Go implementation of the Aho-Corasick search algorithm.

The code is a port from the great Multifast library written in C.
Please visit http://multifast.sourceforge.net/ to learn more about it.


## Usage example


``` go

package main

import (
        "fmt"
        "github.com/lrsk/gomultifast"
)

func main() {
        // Aho-Corasick automaton
        atm := gomultifast.NewAutomaton()
        
        // Add search terms, of course this can be done in a loop from a file etc.
        key := "term1"
        search_term := "golang"
        tmp_patt := gomultifast.NewPattern(key, search_term)
        atm.Add(tmp_patt)
        
        key = "term21"
        search_term = "example.com"
        tmp_patt := gomultifast.NewPattern(key, search_term)
        atm.Add(tmp_patt)

        // No more adding of terms after this
        atm.Finalize()

        fmt.Printf("Finished adding search terms\n")

        //atm.Print() // This will print out the nodes for debugging

        atm.Search("thisissometextwithgolanginit", false, match_handler, "")
        atm.Search("iloveexamplesthatuseexample.cominthem", false, match_handler, "")
        atm.Search("thisisjust some text here", false, match_handler, "")

}

// Callback function to run against matches
func match_handler(matchp gomultifast.Match, param string) int {
        for _, m := range matchp.Patterns {
                // Print search hit
                fmt.Printf("%s,%s\n", m.Ident, param)
        }
        return 0
}
```
