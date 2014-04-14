package models

const (
	ExecutorService = "executor"
)

type ServiceRegistration struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type ServiceRegistrations []ServiceRegistration

func (s ServiceRegistrations) ExecutorRegistrations() ServiceRegistrations {
	registrations := ServiceRegistrations{}
	for _, reg := range s {
		if reg.Name == ExecutorService {
			registrations = append(registrations, reg)
		}
	}
	return registrations
}
