package cb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cloudfoundry-incubator/auctioneer"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/tedsuo/rata"
)

//go:generate counterfeiter . AuctioneerClient
type AuctioneerClient interface {
	RequestLRPStartAuction(auctioneerURL string, lrpStart models.LRPStart) error
	RequestTaskAuction(auctioneerURL string, task models.Task) error
}

type auctioneerClient struct {
	httpClient *http.Client
}

func NewAuctioneerClient() AuctioneerClient {
	return &auctioneerClient{
		httpClient: &http.Client{},
	}
}

func (c *auctioneerClient) RequestLRPStartAuction(auctioneerURL string, lrpStart models.LRPStart) error {
	reqGen := rata.NewRequestGenerator(auctioneerURL, auctioneer.Routes)

	payload, err := json.Marshal(lrpStart)
	if err != nil {
		return err
	}

	req, err := reqGen.CreateRequest(auctioneer.CreateLRPAuctionRoute, rata.Params{}, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("http error: status code %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return nil
}

func (c *auctioneerClient) RequestTaskAuction(auctioneerURL string, task models.Task) error {
	reqGen := rata.NewRequestGenerator(auctioneerURL, auctioneer.Routes)

	payload, err := json.Marshal(task)
	if err != nil {
		return err
	}

	req, err := reqGen.CreateRequest(auctioneer.CreateTaskAuctionRoute, rata.Params{}, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("http error: status code %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return nil
}
