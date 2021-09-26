package sockets

import (
	"bytes"
	"errors"
	"github.com/preludeorg/pneuma/util"
	"os"
	"path/filepath"
	"strings"
)

func requestPayload(target string) (string, error) {
	body, filename, code, err := requestHTTPPayload(target)
	if err != nil {
		return "", err
	}
	if code == 200 {
		workingDir := "./"
		path := filepath.Join(workingDir, filename)
		err = util.SaveFile(bytes.NewReader(body), path)
		if err != nil {
			return "", err
		}

		err = os.Chmod(path, 0755)
		if err != nil {
			return "", err
		}

		return path, nil
	}
	return "", errors.New("UNHANDLED PAYLOAD EXCEPTION")
}

func payloadErrorResponse(err error, agent *util.AgentConfig, link *util.Instruction) {
	link.Response = "Payload Error: " + strings.TrimSpace(err.Error())
	link.Status = 1
	link.Pid = agent.Pid
}
