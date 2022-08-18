package handles

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

type Res struct {
	Code int    `json:"code"`
	Msg  string `json:"message"`
}

var Counter int64 = 0

func BaseHandles(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	res := new(Res)
	var token string
	var requestId string
	ctx := contextx.NewContext()
	requestIdArray, ok := r.Header["X-Ksc-Request-Id"]
	if !ok {
		requestId = fmt.Sprintf("wf-req-%s", uuid.NewString())
	} else {
		requestId = requestIdArray[0]
	}
	ctx.Set("requestId", requestId)

	tokenArray, ok := r.Header["X-Auth-Token"]
	if !ok {
		w.Header().Set("X-Openstack-Request-Id", requestId)
		w.WriteHeader(403)
		res.Code = 403
		res.Msg = "No find the token"
		res_json, _ := json.Marshal(res)
		w.Write(res_json)
		return
	} else {
		token = tokenArray[0]
		if token == "" {
			w.Header().Set("X-Openstack-Request-Id", requestId)
			w.WriteHeader(403)
			res.Code = 403
			res.Msg = "No find the token"
			res_json, _ := json.Marshal(res)
			w.Write(res_json)
			return
		}
	}

	instance_uuid := ps.ByName("uuid")
	tenantid := ps.ByName("tenantid")
	ctx.Set("project_id", tenantid)
	// get body data
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf(nil, "Body decode error %#v", err)
		w.Header().Set("X-Openstack-Request-Id", requestId)
		w.WriteHeader(400)
		res.Code = 400
		res.Msg = err.Error()
		res_json, _ := json.Marshal(res)
		w.Write(res_json)
		return
	}
	map_body := make(map[string]map[string]interface{})
	if err := json.Unmarshal(b, &map_body); err != nil {
		log.Errorf(nil, "Body to map error %#v", err)
		w.Header().Set("X-Openstack-Request-Id", requestId)
		w.WriteHeader(400)
		res.Code = 400
		res.Msg = err.Error()
		res_json, _ := json.Marshal(res)
		w.Write(res_json)
		return
	}
	if len(map_body) != 1 {
		log.Errorf(nil, "Body error %#v", map_body)
		w.Header().Set("X-Openstack-Request-Id", requestId)
		w.WriteHeader(400)
		res.Code = 400
		res.Msg = "No find the action to run workflow!"
		res_json, _ := json.Marshal(res)
		w.Write(res_json)
		return
	}
	var handle string
	for key := range map_body {
		handle = key
	}
	var code int
	var msg string
	switch handle {
	case "ChangeFlavor":
		code, msg = ChangeFlavorYmlHandle(ctx, token, instance_uuid, tenantid, map_body[handle])
		if code == 200 {
			msg = fmt.Sprintf("Workflow ID: %s", msg)
		} else {
			msg = fmt.Sprintf("Run Workflow Error: %s", msg)
		}
	case "OnlineChangePwd":
		code, msg = OnlineChangePwdYmlHandle(ctx, token, instance_uuid, tenantid, map_body[handle])
		if code == 200 {
			msg = fmt.Sprintf("Workflow ID: %s", msg)
		} else {
			msg = fmt.Sprintf("Run Workflow Error: %s", msg)
		}
	case "JustTest":
		code, msg = JustTest(ctx, token, instance_uuid, tenantid, map_body[handle])
		atomic.AddInt64(&Counter, 1)
		if code == 200 {
			msg = fmt.Sprintf("WorkFlow ID: %s", msg)
		} else {
			msg = fmt.Sprintf("Run Workflow Error: %s", msg)
		}

	default:
		code = 400
		msg = fmt.Sprintf("No find the action %s", handle)
	}
	w.Header().Set("X-Openstack-Request-Id", requestId)
	w.WriteHeader(code)
	res.Code = code
	res.Msg = msg
	res_json, _ := json.Marshal(res)
	w.Write(res_json)
}
