package majorfixtures

// Eight returns → MAJOR for returns_per_function.
func EightReturns(x int) int {
	if x == 0 {
		return 0
	}
	if x == 1 {
		return 1
	}
	if x == 2 {
		return 2
	}
	if x == 3 {
		return 3
	}
	if x == 4 {
		return 4
	}
	if x == 5 {
		return 5
	}
	if x == 6 {
		return 6
	}
	return 7
}

// High cyclomatic complexity → MAJOR.
func HighComplexity(a, b, c, d, e bool, n int) int {
	total := 0
	if a {
		total++
	}
	if b {
		total++
	}
	if c {
		total++
	}
	if d {
		total++
	}
	if e {
		total++
	}
	if a && b {
		total++
	}
	if c || d {
		total++
	}
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			total++
		}
	}
	switch n {
	case 1:
		total++
	case 2:
		total++
	case 3:
		total++
	default:
		total++
	}
	return total
}

// Nine parameters → MAJOR.
func NineParams(a, b, c, d, e, f, g, h, i int) int {
	return a + b + c + d + e + f + g + h + i
}

// Deep nesting → MAJOR (>5).
func DeepNesting(ok bool, items []int) int {
	sum := 0
	if ok {
		for range items {
			if true {
				switch {
				case true:
					if true {
						for j := 0; j < 1; j++ {
							if j == 0 {
								sum++
							}
						}
					}
				}
			}
		}
	}
	return sum
}
