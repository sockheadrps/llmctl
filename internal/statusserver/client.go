package statusserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var pollClient = &http.Client{Timeout: 2 * time.Second}

// Poll fetches /status from a remote llmctl instance at host:port.
// Returns an error if the request fails or the response cannot be decoded.
func Poll(host string, port int) (Status, error) {
	return PollAddr(fmt.Sprintf("%s:%d", host, port))
}

// PollAddr fetches /status from a remote llmctl instance at addr ("host:port").
func PollAddr(addr string) (Status, error) {
	url := "http://" + addr + "/status"
	resp, err := pollClient.Get(url)
	if err != nil {
		return Status{}, err
	}
	defer resp.Body.Close()
	var st Status
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		return Status{}, err
	}
	return st, nil
}
