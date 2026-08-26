package minorfixtures

// Function with 6 returns → MINOR for returns_per_function (minor=5, major=7).
func SixReturns(x int) int {
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
	return 5
}

// Complexity around 6–10 → MINOR.
func MildComplexity(a, b, c bool) int {
	n := 0
	if a {
		n++
	}
	if b {
		n++
	}
	if c {
		n++
	}
	if a && b {
		n++
	}
	return n
}

// Function length 21–30 → MINOR.
func MildLength() int {
	a := 1
	b := 2
	c := 3
	d := 4
	e := 5
	f := 6
	g := 7
	h := 8
	i := 9
	j := 10
	k := 11
	l := 12
	m := 13
	n := 14
	o := 15
	p := 16
	q := 17
	r := 18
	s := 19
	t := 20
	return a + b + c + d + e + f + g + h + i + j + k + l + m + n + o + p + q + r + s + t
}

// Six parameters → MINOR.
func SixParams(a, b, c, d, e, f int) int {
	return a + b + c + d + e + f
}
