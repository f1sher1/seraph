package pluginx

type MessageArgs struct {
	Attributes map[string]interface{}
	Message    interface{}
}

type MessageReply struct {
	Message interface{}
}
