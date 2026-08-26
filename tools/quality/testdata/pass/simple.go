package passfixtures

// Simple helpers that stay within all quality thresholds.

func Add(a, b int) int {
	return a + b
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
