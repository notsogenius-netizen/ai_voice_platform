package commentsfixtures

import (
	"fmt"
)

// This file checks that comments, blanks, package, and imports are excluded from LOC.

/*
Block comment that should not count toward file LOC.
Line two of block comment.
*/

func Documented() int {
	// leading comment inside function
	x := 1

	// blank line above should not count
	y := 2
	/* inline-ish block */
	return x + y
}

func Also() {
	fmt.Println("ok")
}
