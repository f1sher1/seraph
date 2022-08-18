package objects

import "fmt"

type ActionResult struct {
	Data   interface{}
	Err    string
	Cancel bool
}

func (r ActionResult) IsSuccess() bool {
	return !r.IsError() && !r.IsCancel()
}

func (r ActionResult) IsCancel() bool {
	return r.Cancel
}

func (r ActionResult) IsError() bool {
	return r.Err != "" && !r.IsCancel()
}

func (r ActionResult) String() string {
	return fmt.Sprintf("Result [data=%#v, error=%s, cancel=%v]", r.Data, r.Err, r.Cancel)
}

func (r ActionResult) ToMap() map[string]interface{} {
	if r.IsSuccess() {
		return map[string]interface{}{
			"result": fmt.Sprintf("%v", r.Data),
		}
	} else {
		return map[string]interface{}{
			"result": fmt.Sprintf("%v:[%v]", r.Err, r.Data),
		}
	}
}
