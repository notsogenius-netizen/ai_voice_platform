package paramsfixtures

// Grouped names must count individually: 3 params, not 1.
func Grouped(a, b, c int) int {
	return a + b + c
}

// Mixed grouped and typed: 5 params.
func Mixed(a, b int, c string, d, e bool) int {
	if d || e {
		return len(c) + a + b
	}
	return a + b
}

// Anonymous / interface-style params in a method-like func.
func WithError(x int, err error) error {
	if x < 0 {
		return err
	}
	return nil
}
