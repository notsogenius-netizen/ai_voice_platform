package methodsfixtures

type Service struct{}

func (s *Service) A() {}
func (s *Service) B() {}
func (s *Service) C() {}
func (s Service) D()  {}

func Helper1() {}
func Helper2() {}
