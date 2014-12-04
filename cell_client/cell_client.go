package cell_client

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry-incubator/runtime-schema/routes"
	"github.com/tedsuo/rata"
)

type Client interface {
	StopLRPInstance(cellAddr string, stopInstance models.StopLRPInstance) error
}

type client struct {
	httpClient *http.Client
}

func New() Client {
	return &client{
		httpClient: &http.Client{},
	}
}

func (c *client) StopLRPInstance(cellURL string, stopInstance models.StopLRPInstance) error {
	reqGen := rata.NewRequestGenerator(cellURL, routes.StopLRPRoutes)
	req, err := reqGen.CreateRequest(routes.StopLRPInstance, paramsFromStopLRPInstance(stopInstance), nil)
	if err != nil {
		return err
	}

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

func paramsFromStopLRPInstance(s models.StopLRPInstance) rata.Params {
	return rata.Params{
		"process_guid":  s.ProcessGuid,
		"instance_guid": s.InstanceGuid,
		"index":         strconv.Itoa(s.Index),
	}
}
