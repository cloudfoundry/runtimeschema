package cell_client

import (
	"net/http"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type Client interface {
	StopLRPInstance(cellAddr string, stopInstance models.StopLRPInstance)
}

type client struct {
	httpClient *http.Client
}

func NewCellClient() Client {
	return &client{
		httpClient: &http.Client{},
	}
}

func (c *client) StopLRPInstance(cellAddr string, stopInstance models.StopLRPInstance) {

}
