package cb

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry-incubator/runtime-schema/routes"
	"github.com/tedsuo/rata"
)

//go:generate counterfeiter . TaskClient
type TaskClient interface {
	CompleteTasks(receptorURL string, tasks []models.Task) error
}

type taskClient struct {
	httpClient *http.Client
}

func NewTaskClient() TaskClient {
	return &taskClient{
		httpClient: &http.Client{},
	}
}

func (c *taskClient) CompleteTasks(receptorURL string, tasks []models.Task) error {
	reqGen := rata.NewRequestGenerator(receptorURL, routes.CompleteTasksRoutes)

	payload, err := json.Marshal(tasks)
	if err != nil {
		return err
	}

	req, err := reqGen.CreateRequest(routes.CompleteTasks, nil, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return errors.New("server error")
	}

	return nil
}
