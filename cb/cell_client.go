package cb

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/cloudfoundry-incubator/cf_http"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry-incubator/runtime-schema/routes"
	"github.com/tedsuo/rata"
)

//go:generate counterfeiter . CellClient

type CellClient interface {
	StopLRPInstance(cellURL string, key models.ActualLRPKey, containerKey models.ActualLRPContainerKey) error
	CancelTask(cellURL string, taskGuid string) error
}

type cellClient struct {
	httpClient *http.Client
}

func NewCellClient() CellClient {
	return &cellClient{
		httpClient: cf_http.NewClient(),
	}
}

func (c *cellClient) StopLRPInstance(
	cellURL string,
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
) error {
	reqGen := rata.NewRequestGenerator(cellURL, routes.StopLRPRoutes)

	req, err := reqGen.CreateRequest(routes.StopLRPInstance, stopParamsFromLRP(key, containerKey), nil)
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
		return fmt.Errorf("http error: status code %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return nil
}

func (c *cellClient) CancelTask(cellURL string, taskGuid string) error {
	reqGen := rata.NewRequestGenerator(cellURL, routes.CancelTaskRoutes)

	req, err := reqGen.CreateRequest(routes.CancelTask, rata.Params{"task_guid": taskGuid}, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("http error: status code %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return nil
}

func stopParamsFromLRP(
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
) rata.Params {
	return rata.Params{
		"process_guid":  key.ProcessGuid,
		"instance_guid": containerKey.InstanceGuid,
		"index":         strconv.Itoa(key.Index),
	}
}
