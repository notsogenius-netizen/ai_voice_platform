package nestedfixtures

func NestedBlocks(flag bool, xs []int) int {
	total := 0
	if flag {
		for _, x := range xs {
			switch x {
			case 1:
				if x > 0 {
					total += x
				}
			default:
				total++
			}
		}
	}
	return total
}
