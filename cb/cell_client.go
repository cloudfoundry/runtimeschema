package cb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry-incubator/runtime-schema/routes"
	"github.com/tedsuo/rata"
)

type CellClient interface {
	StopLRPInstance(cellAddr string, lrp models.ActualLRP) error
}

type cellClient struct {
	httpClient *http.Client
}

func NewCellClient() CellClient {
	return &cellClient{
		httpClient: &http.Client{},
	}
}

func (c *cellClient) StopLRPInstance(cellURL string, lrp models.ActualLRP) error {
	reqGen := rata.NewRequestGenerator(cellURL, routes.StopLRPRoutes)

	payload, err := json.Marshal(lrp)
	if err != nil {
		return err
	}

	req, err := reqGen.CreateRequest(routes.StopLRPInstance, stopParamsFromLRP(lrp), bytes.NewBuffer(payload))
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

func stopParamsFromLRP(lrp models.ActualLRP) rata.Params {
	return rata.Params{
		"process_guid":  lrp.ProcessGuid,
		"instance_guid": lrp.InstanceGuid,
		"index":         strconv.Itoa(lrp.Index),
	}
}
