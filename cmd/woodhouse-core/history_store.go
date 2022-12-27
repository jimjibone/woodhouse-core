package main

import (
	"context"
	"log"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/jimjibone/woodhouse-4/apitools"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/config"
)

type HistoryStore struct {
	wg     sync.WaitGroup
	cancel func()
}

func NewHistoryStore(ds *DeviceStore) *HistoryStore {
	ctx, cancel := context.WithCancel(context.Background())
	hm := &HistoryStore{
		cancel: cancel,
	}
	if config.LoadedConfig.InfluxDB.Enabled {
		hm.wg.Add(1)
		go hm.run(ctx, ds)
	}
	return hm
}

func (hm *HistoryStore) Close() {
	hm.cancel()
	hm.wg.Wait()
}

func (hm *HistoryStore) run(ctx context.Context, ds *DeviceStore) {
	defer hm.wg.Done()

	statesSub := ds.statesPub.NewSub()
	defer statesSub.Close()

	client := influxdb2.NewClient(config.LoadedConfig.InfluxDB.Addr, config.LoadedConfig.InfluxDB.Token)
	writeAPI := client.WriteAPIBlocking(config.LoadedConfig.InfluxDB.Org, config.LoadedConfig.InfluxDB.Bucket)

	for {
		select {
		case <-ctx.Done():
			return

		case state := <-statesSub.Sub():
			// Only write recent updates.
			lastSeen := apitools.TimestampToTime(state.LastSeen)
			if lastSeen.IsZero() || time.Since(lastSeen) < time.Minute {
				tags := map[string]string{
					"bridge_id": state.BridgeId,
					"device_id": state.DeviceId,
				}
				fields := map[string]interface{}{
					"online": state.Online,
				}
				for _, val := range state.Values {
					for name, field := range apitools.ValueFields(val.Name, val) {
						fields[name] = field
					}
				}
				point := write.NewPoint("state", tags, fields, time.Now())

				if err := writeAPI.WritePoint(ctx, point); err != nil {
					log.Printf("ERROR: failed to write data to influxdb: %s", err)
				}
			}
		}
	}
}
