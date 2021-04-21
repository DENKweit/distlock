package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/DENKweit/distlock/types"
)

type api struct {
	Url *url.URL
}

func NewClient(endpoint string) (*api, error) {
	url, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	ret := &api{
		Url: url,
	}

	return ret, nil
}

func (a *api) Status() (status types.StatusReturn, err error) {
	err = nil
	status = types.StatusReturn{}

	url := fmt.Sprintf("%s/status", a.Url.String())

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Error: %s", resp.Status)
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&status)
	if err != nil {
		return
	}

	return
}

func (a *api) Acquire(key string, value string, duration time.Duration) (success bool, sessionID string, err error) {
	err = nil
	success = false
	sessionID = ""

	url := fmt.Sprintf("%s/kv/acquire/%s/%d", a.Url.String(), key, duration)

	req, err := http.NewRequest("POST", url, nil)

	if err != nil {
		return
	}

	q := req.URL.Query()
	q.Add("value", value)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Error: %s", resp.Status)
		return
	}

	ret := &types.AcquireReturn{}

	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return
	}

	success = ret.Success
	sessionID = ret.SessionID

	return
}

func (a *api) Release(key string, sessionID string) (success bool, err error) {
	err = nil
	success = false

	url := fmt.Sprintf("%s/kv/release/%s/%s", a.Url.String(), key, sessionID)

	req, err := http.NewRequest("POST", url, nil)

	if err != nil {
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Error: %s", resp.Status)
		return
	}

	ret := &types.ReleaseReturn{}

	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return
	}

	success = ret.Success

	return
}

func (a *api) Set(key string, value string, sessionID string) (success bool, err error) {
	err = nil
	success = false

	url := fmt.Sprintf("%s/kv/set/%s", a.Url.String(), key)

	req, err := http.NewRequest("POST", url, nil)

	if err != nil {
		return
	}

	q := req.URL.Query()
	q.Add("value", value)
	q.Add("sessionId", sessionID)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Error: %s", resp.Status)
		return
	}

	ret := &types.ReleaseReturn{}

	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return
	}

	success = ret.Success

	return
}

func (a *api) Get(key string) (ret *types.GetReturn, err error) {
	err = nil

	url := fmt.Sprintf("%s/kv/get/%s", a.Url.String(), key)

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Error: %s", resp.Status)
		return
	}

	ret = &types.GetReturn{}

	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return
	}

	return
}

func (a *api) RenewSession(sessionID string, duration time.Duration) (err error) {
	err = nil

	url := fmt.Sprintf("%s/session/renew/%s/%d", a.Url.String(), sessionID, duration)

	req, err := http.NewRequest("POST", url, nil)

	if err != nil {
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Error: %s", resp.Status)
		return
	}

	return
}

func (a *api) DestroySession(sessionID string) (err error) {
	err = nil
	url := fmt.Sprintf("%s/session/destroy/%s", a.Url.String(), sessionID)

	req, err := http.NewRequest("POST", url, nil)

	if err != nil {
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Error: %s", resp.Status)
		return
	}

	return
}

func (a *api) RenewSessionPerdiodic(sessionID string, interval time.Duration, doneCh <-chan struct{}) error {

	err := a.RenewSession(sessionID, interval+time.Second)
	if err != nil {
		return err
	}
	timer := time.NewTimer(interval)

	for {
		select {
		case <-timer.C:
			err := a.RenewSession(sessionID, interval+time.Second)

			timer.Stop()

			if err != nil {
				return err
			}

			timer.Reset(interval)
		case <-doneCh:
			timer.Stop()
			err := a.DestroySession(sessionID)

			if err != nil {
				return err
			}
		}
	}
}

func (a *api) Keys(prefix string) (keys []string, err error) {
	err = nil
	keys = []string{}

	url := fmt.Sprintf("%s/kv/keys", a.Url.String())

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return
	}

	q := req.URL.Query()
	q.Add("prefix", prefix)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Error: %s", resp.Status)
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&keys)
	if err != nil {
		return
	}

	return
}
