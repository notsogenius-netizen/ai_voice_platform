package complexfixtures

func Decide(mode string, a, b int, enabled bool) int {
	result := 0
	if enabled {
		result++
	}
	for i := 0; i < a; i++ {
		if i%2 == 0 || i%3 == 0 {
			result += i
		}
	}
	switch mode {
	case "add":
		result += b
	case "sub":
		result -= b
	case "mul":
		result *= b
	default:
		if a > b && enabled {
			result += a - b
		}
	}
	return result
}
