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
	CompleteTask(receptorURL string, task models.Task) error
}

type taskClient struct {
	httpClient *http.Client
}

func NewTaskClient() TaskClient {
	return &taskClient{
		httpClient: &http.Client{},
	}
}

func (c *taskClient) CompleteTask(receptorURL string, task models.Task) error {
	reqGen := rata.NewRequestGenerator(receptorURL, routes.CompleteTaskRoutes)

	payload, err := json.Marshal(task)
	if err != nil {
		return err
	}

	req, err := reqGen.CreateRequest(routes.CompleteTask, nil, bytes.NewBuffer(payload))
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
