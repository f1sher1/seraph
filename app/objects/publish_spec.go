package objects

type PublishSpec struct {
	Branch map[string]interface{}
	Globals map[string]interface{}
}

func NewPublishSpec(branch map[string]interface{}, globals map[string]interface{}) *PublishSpec {
	return &PublishSpec{
		branch,
		globals,
	}
}
