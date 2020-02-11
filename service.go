package main

// Service performs operations on docker client
type Service struct{}

// NewService creates new upload service
func NewService() *Service {
	return &Service{}
}

// ListLanguages return a list of avalible languages
func (s *Service) ListLanguages() (error, error) {
	return nil, nil
}

// ListContainers return a list of avalible containers
func (s *Service) ListContainers() (error, error) {
	return nil, nil
}

// CreateContainer creates a new container
func (s *Service) CreateContainer(lang string) error {
	return nil
}

// Eval evaluates provided code
func (s *Service) Eval(language string, code string) (error, error) {
	return nil, nil
}

// Cleanup cleans up containers
func (s *Service) Cleanup() (error, error) {
	return nil, nil
}
