package cb

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/cloudfoundry-incubator/bbs/cellhandlers"
	"github.com/cloudfoundry-incubator/cf_http"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/tedsuo/rata"
)

//go:generate counterfeiter . CellClient

type CellClient interface {
	StopLRPInstance(cellURL string, key models.ActualLRPKey, instanceKey models.ActualLRPInstanceKey) error
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
	instanceKey models.ActualLRPInstanceKey,
) error {
	reqGen := rata.NewRequestGenerator(cellURL, cellhandlers.Routes)

	req, err := reqGen.CreateRequest(cellhandlers.StopLRPInstanceRoute, stopParamsFromLRP(key, instanceKey), nil)
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

func stopParamsFromLRP(
	key models.ActualLRPKey,
	instanceKey models.ActualLRPInstanceKey,
) rata.Params {
	return rata.Params{
		"process_guid":  key.ProcessGuid,
		"instance_guid": instanceKey.InstanceGuid,
		"index":         strconv.Itoa(key.Index),
	}
}
