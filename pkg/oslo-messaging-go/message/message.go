package message

import (
	"encoding/json"
	"errors"
	"fmt"
	"seraph/pkg/contextx"
	"strings"

	"github.com/google/uuid"
)

const (
	contextMessagePrefix = "_context_"
)

type MessageBody struct {
	RawBody    []byte
	Body       map[string]interface{}
	MessageId  string
	ReplyQueue string
	UniqueID   string
	Ctx        *contextx.Context

	err error

	OlsoVersion string
}

func (m MessageBody) GetReply() (interface{}, error) {
	if m.HasError() {
		return nil, m.GetError()
	}

	if result, ok := m.Body["result"]; ok {
		return result, nil
	}

	return nil, fmt.Errorf("invalid reply message '%s'", m.Body)
}

func (m MessageBody) BuildReply(reply interface{}, err error, ending bool) (MessageBody, error) {
	body := map[string]interface{}{
		"result": reply,
		"ending": ending,
	}

	replyBody := MessageBody{
		Body:        body,
		err:         err,
		MessageId:   m.MessageId,
		OlsoVersion: m.OlsoVersion,
		ReplyQueue:  m.ReplyQueue,
		Ctx:         m.Ctx,
	}

	return replyBody, nil
}

func (m *MessageBody) Initialize() {
	if olsoVersion, ok := m.Body["oslo.version"]; ok {
		m.OlsoVersion = olsoVersion.(string)
		m.Body = m.Body["oslo.message"].(map[string]interface{})
	}

	if failure, ok := m.Body["failure"]; ok && failure.(string) != "" {
		m.err = errors.New(failure.(string))
	}

	if msgId, ok := m.Body["_msg_id"]; ok && msgId.(string) != "" {
		m.MessageId = msgId.(string)
	}

	if replyQueue, ok := m.Body["_reply_q"]; ok && replyQueue.(string) != "" {
		m.ReplyQueue = replyQueue.(string)
	}

	if uniqueId, ok := m.Body["_unique_id"]; ok && uniqueId.(string) != "" {
		m.UniqueID = uniqueId.(string)
	}

	ctx := map[string]interface{}{}
	for name, value := range m.Body {
		if strings.HasPrefix(name, contextMessagePrefix) {
			realName := name[len(contextMessagePrefix):]
			ctx[realName] = value
		}
	}
	m.Ctx = contextx.NewContextFromMap(ctx)
}

func (m MessageBody) ToBytes() ([]byte, error) {
	data := m.Body
	if m.err != nil {
		data["failure"] = m.err.Error()
	}
	if m.MessageId != "" {
		data["_msg_id"] = m.MessageId
	}
	if m.ReplyQueue != "" {
		data["_reply_q"] = m.ReplyQueue
	}
	if m.UniqueID != "" {
		data["_unique_id"] = m.UniqueID
	}

	for name, value := range m.Ctx.GetMap() {
		data[contextMessagePrefix+name] = value
	}

	if m.OlsoVersion != "" {
		data = map[string]interface{}{
			"oslo.version": "2.0",
			"oslo.message": data,
		}
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (m MessageBody) HasError() bool {
	return m.err != nil
}

func (m MessageBody) GetError() error {
	return m.err
}

func (m MessageBody) GetMethod() string {
	if method, ok := m.Body["method"]; ok && method.(string) != "" {
		return method.(string)
	}
	return ""
}

func (m MessageBody) GetArguments() map[string]interface{} {
	if arguments, ok := m.Body["args"]; ok {
		return arguments.(map[string]interface{})
	}
	return map[string]interface{}{}
}

func (m MessageBody) GetContext() *contextx.Context {
	return m.Ctx
}

func NewMessageBody(ctx *contextx.Context, body map[string]interface{}, osloVersion string) MessageBody {
	return MessageBody{
		Body:        body,
		UniqueID:    uuid.NewString(),
		Ctx:         ctx,
		OlsoVersion: osloVersion,
	}
}

func ParseMessageBody(body []byte) (*MessageBody, error) {
	b := map[string]interface{}{}
	err := json.Unmarshal(body, &b)
	if err != nil {
		return nil, err
	}

	m := &MessageBody{
		RawBody: body,
		Body:    b,
	}
	m.Initialize()
	return m, nil
}
