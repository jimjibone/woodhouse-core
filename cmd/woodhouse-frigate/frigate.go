package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-frigate/api"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh/v1"
)

const (
	minBackoff = time.Second
	maxBackoff = 30 * time.Second
)

type Frigate struct {
	ServerAddr      string
	log             *log.Context
	lastBackoff     time.Time
	lastRestart     time.Time
	backoffDuration time.Duration
	client          *wh.Client
	connMu          sync.RWMutex
	conn            *websocket.Conn
	caneras         map[string]*FrigateCamera // key=camera name
}

func (frig *Frigate) Run(ctx context.Context, client *wh.Client) error {
	if frig.log == nil {
		frig.log = log.NewContext(log.DefaultLogger, "frigate", log.DebugLevel)
	}

	frig.client = client
	frig.caneras = make(map[string]*FrigateCamera)

	log.Infof("started")
	defer log.Infof("finished")

	for {
		// Try to connect.
		frig.log.Infof("starting connection")
		conn := frig.connect(ctx)
		if conn != nil {
			frig.connMu.Lock()
			frig.conn = conn
			frig.connMu.Unlock()

			// Close the websocket when the context is cancelled.
			go func() {
				<-ctx.Done()
				frig.log.Infof("closing connection")
				frig.connMu.Lock()
				if frig.conn != nil {
					frig.conn.Close()
					frig.conn = nil
				}
				frig.connMu.Unlock()
			}()

			// Receive updates and send requests.
			frig.recv(ctx, conn)
		}

		// Check if we're done.
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Backoff for a short while before the next connection attempt.
		frig.backoff(ctx)
	}
}

func (frig *Frigate) connect(ctx context.Context) *websocket.Conn {
	rawConfig, config, err := api.GetConfig(ctx, frig.ServerAddr)
	if err != nil {
		frig.log.Errorln("dial:", err)
		return nil
	}
	frig.log.Infof("config: %s", rawConfig)
	if debugSaveJson {
		api.SaveJSON("frigate-config.json", rawConfig)
	}

	// rawStats, stats, err := api.GetStats(frig.ServerAddr)
	// if err != nil {
	// 	frig.log.Errorln("dial:", err)
	// 	return nil
	// }
	// frig.log.Infof("----> recv stats: %s", stats)
	// api.SaveJSON("frigate-stats.json", rawStats)

	for name := range config.Cameras {
		if _, found := frig.caneras[name]; !found {
			frig.caneras[name] = NewFrigateCamera(frig.ServerAddr, name, frig.client)
		}
	}

	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/ws", frig.ServerAddr), nil)
	if err != nil {
		frig.log.Errorln("dial:", err)
		return nil
	}
	return conn
}

func (frig *Frigate) recv(ctx context.Context, conn *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		frame := &api.Frame{}
		err := conn.ReadJSON(frame)
		// _, message, err = conn.ReadMessage()
		if err != nil {
			frig.log.Errorf("read: %s", err)
			conn.Close()
			return
		}

		// frig.log.Debugf("----> recv frame: %s", frame)

		// frame := struct {
		// 	Topic   string          `json:"topic"`
		// 	Payload json.RawMessage `json:"payload"`
		// }{}
		// err = json.Unmarshal(message, &frame)
		// if err != nil {
		// 	frig.log.Errorln("unmarshal:", err)
		// 	conn.Close()
		// 	return
		// }

		switch frame.Topic {
		case "stats":
			frame.Payload = api.SanitiseJSON(frame.Payload)
			var message api.Stats
			err = json.Unmarshal(frame.Payload, &message)
			if err != nil {
				frig.log.Debugf("----> recv frame: %s", frame)
				frig.log.Errorln("unmarshal stats:", err)
				conn.Close()
				return
			}
			frig.log.Debugf("----> recv stats: %s", message)
			if debugSaveJson {
				api.SaveJSON("frigate-stats.json", frame.Payload)
			}

		default:
			if strings.HasSuffix(frame.Topic, "/motion") {
				name := strings.TrimSuffix(frame.Topic, "/motion")
				if camera, found := frig.caneras[name]; found {
					camera.HandleMotion(frame.Payload)
				} else {
					frig.log.Errorf("received motion for unknown camera: %s", frame)
				}
			} else {
				frig.log.Warnf("----> recv unknown: %s", frame)
				if debugSaveJson {
					name := strings.ReplaceAll(strings.TrimSpace(strings.TrimPrefix(frame.Topic, "/")), "/", "_")
					api.SaveJSON(fmt.Sprintf("frigate-unknown-%s.json", name), api.SanitiseJSON(frame.Payload))
				}
			}
		}
	}
}

func (frig *Frigate) backoff(ctx context.Context) {
	// Reset the backoff duration if the backoff has not been used for a
	// suitable amount of time.
	dt := time.Since(frig.lastRestart)
	if dt > frig.backoffDuration {
		frig.log.Debugf("backoff reset after %s", dt)
		frig.backoffDuration = minBackoff
	}
	frig.lastBackoff = time.Now()
	frig.log.Infof("starting backoff for %s", frig.backoffDuration)
	timer := time.NewTimer(frig.backoffDuration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
	frig.log.Debugf("backoff finished")
	frig.backoffDuration = frig.backoffDuration * 2
	if frig.backoffDuration > maxBackoff {
		frig.backoffDuration = maxBackoff
	}
	frig.lastRestart = time.Now()
}
