package unimq

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"seraph/app/config"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"sort"
	"strconv"
	"strings"
)

type Msg struct {
	ContextRequestId           string                 `json:"_context_request_id"`     // workflow id
	ContextRequestAction       string                 `json:"_context_request_action"` // action ex name
	ContextSeriveCatalog       string                 `json:"_context_service_catalog"`
	ContextAuthToken           string                 `json:"_context_auth_token"`
	ContextUserId              string                 `json:"_context_user_id"`
	ContextRequestActionData   string                 `json:"_context_request_action_data"`
	ContextEnableTraceLog      bool                   `json:"_context_enble_trace_log"`
	ContextIsAdmin             bool                   `json:"_context_is_admin"`
	ContextTimestamp           string                 `json:"_context_timestamp"`
	ContextRemoteAddress       string                 `json:"_context_remote_address"`
	ContextRoles               []string               `json:"_context_roles"`
	ContextProjectName         string                 `json:"_context_project_name"`
	ContextReadDeleted         string                 `json:"_context_read_deleted"`
	ContextInstanceLockChecked bool                   `json:"_context_instance_lock_checked"`
	ContextProjectId           string                 `json:"_context_project_id"`
	ContextUserName            string                 `json:"_context_user_name"`
	EventType                  string                 `json:"event_type"`
	Payload                    map[string]interface{} `json:"payload"`
	Priority                   string                 `json:"priority"`
	PublisherId                string                 `json:"publisher_id"`
	MessageId                  string                 `json:"message_id"`
	Timestamp                  string                 `json:"timestamp"`
	UniqueId                   string                 `json:"_unique_id"`
	WorkflowId                 string                 `json:"workflow_id"`
}

func DeliverMessage(ctx *contextx.Context, message Msg) (string, int) {
	dt, _ := json.Marshal(&message)
	msg := make(map[string]interface{})
	json.Unmarshal(dt, &msg)

	method := "POST"
	path := []string{config.Config.UniMQClient.URL, "message", "topic", "send"}
	url := strings.Join(path, "/")

	params := map[string]interface{}{
		"topic_name":     config.Config.UniMQClient.TopicName,
		"msg_id":         msg["_unique_id"],
		"msg_action":     msg["event_type"],
		"msg_timestamp":  msg["timestamp"],
		"msg_request_id": msg["_context_request_id"],
		"app_key":        config.Config.UniMQClient.AppKey,
		"routing_key":    config.Config.UniMQClient.RoutingKey,
	}
	byte_msg, _ := json.Marshal(msg)
	params["msg"] = string(byte_msg)
	params["sign"] = func(params map[string]interface{}) string {
		var s string
		var keys []string

		for k := range params {
			keys = append(keys, k)
		}

		sort.Sort(sort.StringSlice(keys))

		for _, k := range keys {
			var build strings.Builder
			build.WriteString(s)
			build.WriteString(k)
			build.WriteString("=")
			switch v := params[k].(type) {
			case string:
				build.WriteString(v)
			case []byte:
				build.WriteString(string(v))
			case int:
				build.WriteString(strconv.Itoa(v))
			}
			s = build.String()
		}
		s += config.Config.UniMQClient.SecretKey

		h := md5.New()
		h.Write([]byte(s))
		re := h.Sum(nil)
		return fmt.Sprintf("%x", re)
	}(params)
	// for k, v := range params {
	// 	fmt.Printf("%#v|%#v\n", k, v)
	// }
	code, resp := requests(ctx, method, url, params)
	log.Debugf(ctx, "Send message to unimq params is %v | return code %v, response %v", params, code, resp)
	return resp, code
}

func requests(ctx *contextx.Context, method, _url string, params map[string]interface{}) (int, string) {
	client := &http.Client{}
	DataUrlVal := url.Values{}
	for k, v := range params {
		switch value := v.(type) {
		case string:
			DataUrlVal.Add(k, value)
		default:
			fmt.Printf("%v | %v", k, value)
			panic(value)
		}
	}
	payload := strings.NewReader(DataUrlVal.Encode())
	request, err := http.NewRequest(method, _url, payload)
	if err != nil {
		log.Errorf(ctx, "Create request error : %v", err)
		return 999, err.Error()
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		log.Errorf(ctx, "Request UniMQ Error :%v", err)
		return 999, err.Error()
	}
	defer response.Body.Close()

	b, _ := ioutil.ReadAll(response.Body)
	return response.StatusCode, string(b)
}

// func main() {
// 	now := time.Now()
// 	msg := Msg{
// 		ContextRequestId:     uuid.NewString(),
// 		ContextRequestAction: "test",
// 		ContextRoles:         []string{"admin"},
// 		ContextIsAdmin:       true,
// 		ContextReadDeleted:   "no",
// 		ContextProjectId:     "",
// 		EventType:            "seraph.test",
// 		Priority:             "INFO",
// 		Payload: map[string]interface{}{
// 			"instance_id": "uuid",
// 			"status":      "test",
// 			"task_name":   "task_name",
// 			"task_id":     "1",
// 		},
// 		Timestamp: now.Format("2006-01-01T15:04:05.000"),
// 		UniqueId:  fmt.Sprintf("wf-%v", uuid.NewString()),
// 	}
// 	DeliverMessage(msg)
// }
