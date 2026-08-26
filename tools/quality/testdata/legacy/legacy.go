package legacyfixtures

// LegacyService intentionally sits in the MINOR band for methods_per_file
// when thresholds are lowered in tests. Production defaults use 25/35.

type LegacyService struct{}

func (s *LegacyService) M01() {}
func (s *LegacyService) M02() {}
func (s *LegacyService) M03() {}
func (s *LegacyService) M04() {}
func (s *LegacyService) M05() {}
func (s *LegacyService) M06() {}
func (s *LegacyService) M07() {}
func (s *LegacyService) M08() {}

func Mild(x int) int {
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
