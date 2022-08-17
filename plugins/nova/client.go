package nova

import (
	"seraph/pkg/pluginx"
	"seraph/plugins/nova/action"
)

var (
	endpoints = map[string]pluginx.EndpointCallable{
		"nova.instance_stop":               &action.InstanceStop{},
		"nova.instance_start":              &action.InstanceStart{},
		"nova.instance_resize":             &action.InstanceResize{},
		"nova.check_instance_status":       &action.CheckInstanceStatus{},
		"nova.instance_change_pwd":         &action.InstanceChangePwd{},
		"nova.send_unimq":                  &action.SendUniMQ{},
		"nova.instance_before_migrate":     &action.InstanceBeforeMigrate{},
		"nova.check_instance_storage_type": &action.CheckInstanceStorageType{},
	}
)

func NewPluginServer(addr string) (*pluginx.PluginServer, error) {
	server, err := pluginx.NewPluginServer(addr)
	if err != nil {
		return nil, err
	}
	for name, endpoint := range endpoints {
		err = server.Register(name, endpoint)
		if err != nil {
			return nil, err
		}
	}
	return server, nil
}

func GetEndpoints() map[string]pluginx.EndpointCallable {
	return endpoints
}
