package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

func GetConfig(ctx context.Context, addr string) ([]byte, Config, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "http://"+addr+"/api/config", nil)
	if err != nil {
		return nil, Config{}, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, Config{}, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, Config{}, err
	}
	res.Body.Close()

	var msg Config
	err = json.Unmarshal(body, &msg)
	if err != nil {
		return body, Config{}, err
	}
	return body, msg, nil
}

func GetStats(ctx context.Context, addr string) ([]byte, Stats, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "http://"+addr+"/api/stats", nil)
	if err != nil {
		return nil, Stats{}, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, Stats{}, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, Stats{}, err
	}
	res.Body.Close()

	var msg Stats
	err = json.Unmarshal(body, &msg)
	if err != nil {
		return body, Stats{}, err
	}
	return body, msg, nil
}

func GetLatestImage(ctx context.Context, addr, cameraName string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "http://"+addr+"/api/"+cameraName+"/latest.jpg", nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	res.Body.Close()

	return body, nil
}
