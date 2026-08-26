package multifxtures

// Multiple violation types in one file.

func TooManyReturns(v int) string {
	if v == 0 {
		return "a"
	}
	if v == 1 {
		return "b"
	}
	if v == 2 {
		return "c"
	}
	if v == 3 {
		return "d"
	}
	if v == 4 {
		return "e"
	}
	if v == 5 {
		return "f"
	}
	if v == 6 {
		return "g"
	}
	return "h"
}

func TooManyParams(a, b, c, d, e, f, g, h, i int) int {
	return a + b + c + d + e + f + g + h + i
}
