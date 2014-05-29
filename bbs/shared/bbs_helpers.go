package shared

import (
	"github.com/cloudfoundry/storeadapter"
	"time"
)

func RetryIndefinitelyOnStoreTimeout(callback func() error) error {
	for {
		err := callback()

		if err == storeadapter.ErrorTimeout {
			time.Sleep(time.Second)
			continue
		}

		return err
	}
}
