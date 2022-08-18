package action

import (
	"encoding/json"
	"seraph/pkg/log"
	"seraph/plugins/plugin"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_InstanceStop(t *testing.T) {
	asserter := assert.New(t)

	input := map[string]interface{}{
		"uuid": "6aad637c-7754-48e6-a589-327bb35528de",
		"auth": map[string]interface{}{
			"tenantid": "8e772c1ca0bd45109c71694211ab4218",
			"token":    "cee50245abd24ce78645a992dc232b0e:8e772c1ca0bd45109c71694211ab4218",
		},
		// "enable_track_resource": 1, // 默认值： 1
		"body": map[string]interface{}{
			"enable_track_resource": 0,
		},
	}

	result, err := plugin.Call("nova.instance_stop", nil, input)

	if asserter.NoError(err) {
		log.Debugf(nil, "%#v", result)
	} else {
		log.Error(nil, err)
	}

}

func Test_InstanceStart(t *testing.T) {
	asserter := assert.New(t)

	input := InstanceStartInput{
		InstanceUUID: "6aad637c-7754-48e6-a589-327bb35528de",
		UserAuthorty: map[string]interface{}{
			"tenantid": "8e772c1ca0bd45109c71694211ab4218",
			"token":    "cee50245abd24ce78645a992dc232b0e:8e772c1ca0bd45109c71694211ab4218",
		},
		Body: map[string]interface{}{},
	}

	result, err := plugin.Call("nova.instance_start", nil, input)

	if asserter.NoError(err) {
		log.Debug(nil, result)
	} else {
		log.Error(nil, err)
	}

}

func Test_InstanceResize(t *testing.T) {
	asserter := assert.New(t)
	body := map[string]interface{}{
		"flavorRef": "4", // 29443e41-b7f4-43cc-897f-8d51f820235a
	}
	bodyStr, _ := json.Marshal(body)
	input := InstanceResizeInput{
		InstanceUUID: "6aad637c-7754-48e6-a589-327bb35528de",
		UserAuthorty: map[string]interface{}{
			"tenantid": "8e772c1ca0bd45109c71694211ab4218",
			"token":    "cee50245abd24ce78645a992dc232b0e:8e772c1ca0bd45109c71694211ab4218",
		},
		Body: string(bodyStr),
	}

	result, err := plugin.Call("nova.instance_resize", nil, input)

	if asserter.NoError(err) {
		log.Debug(nil, result)
	} else {
		log.Error(nil, err)
	}

}
func Test_InstanceDetail(t *testing.T) {
	asserter := assert.New(t)

	input := CheckInstanceStatusInput{
		InstanceUUID: "6aad637c-7754-48e6-a589-327bb35528de",
		UserAuthorty: map[string]interface{}{
			"tenantid": "8e772c1ca0bd45109c71694211ab4218",
			"token":    "cee50245abd24ce78645a992dc232b0e:8e772c1ca0bd45109c71694211ab4218",
		},
		// Retry:      5,
		// Sleep:      2,
		// LastStatus: "ACTIVE",
	}

	result, err := plugin.Call("nova.instance_detail", nil, input)

	if asserter.NoError(err) {
		log.Debug(nil, result)
	} else {
		log.Error(nil, err)
	}

}

func Test_InstanceChangePwd(t *testing.T) {
	asserter := assert.New(t)

	input := InstanceStartInput{
		InstanceUUID: "6aad637c-7754-48e6-a589-327bb35528de",
		UserAuthorty: map[string]interface{}{
			"tenantid": "8e772c1ca0bd45109c71694211ab4218",
			"token":    "cee50245abd24ce78645a992dc232b0e:8e772c1ca0bd45109c71694211ab4218",
		},
		Body: map[string]interface{}{
			"adminPass": "123456",
		},
	}

	result, err := plugin.Call("nova.instance_change_pwd", nil, input)

	if asserter.NoError(err) {
		log.Debug(nil, result)
	} else {
		log.Error(nil, err)
	}

}
