package message

type Target struct {
	Exchange string
	Topic    string
	Host     string
	Version  string
	Fanout   bool
	Notify   bool
	Priority string
}

func (t Target) Copy() Target {
	return Target{
		Exchange: t.Exchange,
		Topic:    t.Topic,
		Host:     t.Host,
		Version:  t.Version,
		Fanout:   t.Fanout,
		Notify:   t.Notify,
		Priority: t.Priority,
	}
}
