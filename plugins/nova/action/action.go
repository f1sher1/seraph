package action

import (
	"io/ioutil"
	"net/http"
	"seraph/app/config"
	"seraph/app/objects"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"strings"
)

var (
	URL = config.Config.NovaClient.URL
)

type NovaInput struct {
}

type NovaOutput struct {
	Output objects.ActionResult
}

func NovaRunAPI(ctx *contextx.Context, method, url, token string, body string, workflowId string) (string, int) {
	client := &http.Client{}
	var req *http.Request
	var err error

	if body == "" {
		req, err = http.NewRequest(method, url, nil)
	} else {
		req, err = http.NewRequest(method, url, strings.NewReader(body))
	}

	if err != nil {
		log.Error(workflowId, err)
		return err.Error(), 999
	}
	var requestId string
	if rId, ok := ctx.GetMap()["requestId"]; !ok {
		requestId = "****"
	} else {
		requestId = rId.(string)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("X-KSC-REQUEST-ID", requestId)
	req.Header.Set("X-Openstack-Request-Id", requestId)
	resp, err := client.Do(req)
	if err != nil {
		log.Error(workflowId, err)
		return err.Error(), 999
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(nil, err)
		return err.Error(), 999
	}

	if b == nil {
		b = []byte("")
	}

	log.Debugf(ctx, "NOVA API result %v | %#v", resp.StatusCode, string(b))
	return string(b), resp.StatusCode
}
