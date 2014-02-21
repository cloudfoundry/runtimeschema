package bbs

import (
	"github.com/cloudfoundry/storeadapter"
	"time"
)

func maintainPresence(store storeadapter.StoreAdapter, key string, value []byte, interval uint64) (chan bool, chan error, error) {
	err := store.SetMulti([]storeadapter.StoreNode{
		{
			Key:   key,
			Value: value,
			TTL:   interval,
		},
	})

	if err != nil {
		return nil, nil, err
	}

	stop := make(chan bool)
	errors := make(chan error)

	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second / 2)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				err := store.Update(storeadapter.StoreNode{
					Key:   key,
					Value: value,
					TTL:   interval,
				})

				if err != nil {
					errors <- err
					return
				}
			case <-stop:
				return
			}
		}
	}()

	return stop, errors, nil
}
