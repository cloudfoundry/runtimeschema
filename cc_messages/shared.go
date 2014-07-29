package cc_messages

import "github.com/cloudfoundry-incubator/runtime-schema/models"

type EnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Environment []EnvironmentVariable

func (env Environment) BBSEnvironment() []models.EnvironmentVariable {
	bbsEnv := make([]models.EnvironmentVariable, len(env))
	for i, envVar := range env {
		bbsEnv[i] = models.EnvironmentVariable{Name: envVar.Name, Value: envVar.Value}
	}
	return bbsEnv
}
