// This piece of code test the declaration behavior inside a if loop

package main

import "fmt"

func main() {
	var i, j, k = 0, 0, 0
	b := true
	if b {
		j = 1
		k := 1       // Only here ! -> "k non used"
		r, i := 2, 2 // New declaration of i BECAUSE := used in a different scope
		fmt.Printf("i=%d, r=%d, k=%d\n", i, r, k)
	}
	fmt.Printf("i=%d, j=%d, k=%d\n", i, j, k)
}
