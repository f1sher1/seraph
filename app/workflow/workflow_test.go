package workflow

import (
	"encoding/json"
	"log"
	"seraph/app/db"
	"seraph/app/objects"
	"seraph/app/workflow/states"
	"seraph/pkg/contextx"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWorkflow_Start(t *testing.T) {
	cfg := &db.Config{
		Connection:  "mysql://root:123456@0.0.0.0:3306/seraph?charset=utf8&parseTime=True&loc=Local",
		Debug:       false,
		PoolSize:    10,
		IdleTimeout: 3600,
	}
	err := db.Init(cfg)

	asserter := assert.New(t)
	// if asserter.NoError(err) {
	// 	err = db.Migrate()

	if asserter.NoError(err) {
		ctx := contextx.NewContext()
		ctx.Set("project_id", "project_id_1")

		wfSpec := objects.WorkflowSpec{
			Name:        "test workflow",
			Description: "desc",
			Tags:        nil,
			Type:        "direct",
			Inputs: []interface{}{
				objects.Table{"name": "hehe1"},
			},
			Tasks: map[string]objects.TaskSpec{
				"task_a": {
					Name:   "task_a",
					Action: "std.echo output='{{ .name }}'",
					Parameters: []interface{}{
						objects.Table{"name": "hehe"},
					},
					OnSuccess: []interface{}{
						"task_b",
					},
					// OnComplete: []interface{}{
					// 	"task_c",
					// },
				},
				"task_b": {
					Name:   "task_b",
					Action: "std.echo output='aaa'",
					OnSuccess: []interface{}{
						"task_c",
					},
				},
				"task_c": {
					Name:   "task_c",
					Action: "std.echo output='complete'",
				},
			},
			TaskDefaults: nil,
		}

		wfDef := objects.NewWorkflowDefinition()
		wfDef.ID = uuid.NewString()
		wfDef.Name = "test workflow"
		specBytes, err := json.Marshal(wfSpec)
		if asserter.NoError(err) {
			wfDef.Spec = string(specBytes)
			wfDef.Deleted = 0
			wfDef.Namespace = ""
			wfDef.ProjectID = "project_id_1"

			s, err := wfDef.GetSpec()
			if asserter.NoError(err) {
				//log.Printf("%#v", s)

				wf := Workflow{
					Spec: s,
					Ctx:  ctx,
				}
				err = wf.Initialize()
				if asserter.NoError(err) {
					//log.Printf("%#v", wf)
					exID := uuid.NewString()

					err = wf.Start(exID, "exec description", "", nil, objects.Table{
						"name": "haha",
					}, nil)
					if asserter.NoError(err) {
						wfEx, err := objects.QueryWorkflowExecutionByID(ctx, exID)
						if asserter.NoError(err) {
							for {
								taskExs, err := wfEx.GetChildrenTaskExecutions()
								if asserter.NoError(err) {
									for _, ex := range taskExs {
										if states.IsSuccess(ex.State) {
											goto DOWN
										}
									}
								}
								time.Sleep(1 * time.Second)
							}
						}
					DOWN:
						log.Println(nil, "finished")
					}
				}

			}
		}

	}

	//}

}

func Test_WorkflowInstances(t *testing.T) {
	cfg := &db.Config{
		Connection:  "mysql://root:123456@0.0.0.0:3306/seraph?charset=utf8&parseTime=True&loc=Local",
		Debug:       false,
		PoolSize:    10,
		IdleTimeout: 3600,
	}
	err := db.Init(cfg)

	asserter := assert.New(t)
	if asserter.NoError(err) {
		ctx := contextx.NewContext()
		ctx.Set("project_id", "project_id_1")

		wfSpec := objects.WorkflowSpec{
			Name:        "three union",
			Description: "desc",
			Tags:        nil,
			Type:        "direct",
			// Inputs: []interface{}{
			// 	map[string]string{
			// 		"uuid": "6aad637c-7754-48e6-a589-327bb35528de",
			// 	},
			// 	map[string]interface{}{
			// 		"auth": map[string]string{
			// 			"tenantid": "8e772c1ca0bd45109c71694211ab4218",
			// 			"token":    "cee50245abd24ce78645a992dc232b0e:8e772c1ca0bd45109c71694211ab4218",
			// 		},
			// 	},
			// },
			Tasks: map[string]objects.TaskSpec{
				"intance_stop": {
					Name:   "intance_stop",
					Action: "nova.instance_stop",
					Parameters: []interface{}{
						map[string]interface{}{
							"uuid": "6aad637c-7754-48e6-a589-327bb35528de",
							"auth": map[string]string{
								"tenantid": "8e772c1ca0bd45109c71694211ab4218",
								"token":    "cee50245abd24ce78645a992dc232b0e:8e772c1ca0bd45109c71694211ab4218",
							},
							"boyd": map[string]interface{}{
								"force_stop": true,
							},
						},
					},
					OnSuccess: []interface{}{
						"get_instance_status_stop",
					},
					OnError: []interface{}{
						"task_c",
					},
				},
				"get_instance_status_stop": {
					Name:   "get_instance_status_stop",
					Action: "nova.check_instance_status",
					Parameters: []interface{}{
						map[string]interface{}{
							"uuid": "6aad637c-7754-48e6-a589-327bb35528de",
							"auth": map[string]string{
								"tenantid": "8e772c1ca0bd45109c71694211ab4218",
								"token":    "cee50245abd24ce78645a992dc232b0e:8e772c1ca0bd45109c71694211ab4218",
							},
							"retry":       10,
							"sleep":       2,
							"last_status": "SHUTOFF",
						},
					},
					OnSuccess: []interface{}{
						"instance_resize",
					},
					OnError: []interface{}{
						"task_c",
					},
				},
				"instance_resize": {
					Name:   "instance_resize",
					Action: "nova.instance_resize",
					Parameters: []interface{}{
						map[string]interface{}{
							"uuid": "6aad637c-7754-48e6-a589-327bb35528de",
							"auth": map[string]string{
								"tenantid": "8e772c1ca0bd45109c71694211ab4218",
								"token":    "cee50245abd24ce78645a992dc232b0e:8e772c1ca0bd45109c71694211ab4218",
							},
							"body": map[string]string{"flavorRef": "4"}, // 4, 29443e41-b7f4-43cc-897f-8d51f820235a
						},
					},
					OnSuccess: []interface{}{
						"get_instance_status_resize",
					},
					OnError: []interface{}{
						"task_c",
					},
				},
				"get_instance_status_resize": {
					Name:   "get_instance_status_resize",
					Action: "nova.check_instance_status",
					Parameters: []interface{}{
						map[string]interface{}{
							"uuid": "6aad637c-7754-48e6-a589-327bb35528de",
							"auth": map[string]string{
								"tenantid": "8e772c1ca0bd45109c71694211ab4218",
								"token":    "cee50245abd24ce78645a992dc232b0e:8e772c1ca0bd45109c71694211ab4218",
							},
							"retry":       10,
							"sleep":       1,
							"last_status": "SHUTOFF",
						},
					},
					OnSuccess: []interface{}{
						"instance_start",
					},
					OnError: []interface{}{
						"task_c",
					},
				},
				"instance_start": {
					Name:   "instance_start",
					Action: "nova.instance_start",
					Parameters: []interface{}{
						map[string]interface{}{
							"uuid": "6aad637c-7754-48e6-a589-327bb35528de",
							"auth": map[string]string{
								"tenantid": "8e772c1ca0bd45109c71694211ab4218",
								"token":    "cee50245abd24ce78645a992dc232b0e:8e772c1ca0bd45109c71694211ab4218",
							},
						},
					},
					OnError: []interface{}{
						"task_c",
					},
				},
				"task_c": {
					Name:   "task_c",
					Action: "std.echo output='error'",
				},
			},
			TaskDefaults: nil,
		}

		wfDef := objects.NewWorkflowDefinition()
		wfDef.ID = uuid.NewString()
		wfDef.Name = "test workflow"
		specBytes, err := json.Marshal(wfSpec)
		if asserter.NoError(err) {
			wfDef.Spec = string(specBytes)
			wfDef.Deleted = 0
			wfDef.Namespace = ""
			wfDef.ProjectID = "project_id_1"

			s, err := wfDef.GetSpec()
			if asserter.NoError(err) {
				wf := Workflow{
					Spec: s,
					Ctx:  ctx,
				}
				err = wf.Initialize()
				if asserter.NoError(err) {
					exID := uuid.NewString()

					err = wf.Start(exID, "exec description", "", nil, objects.Table{}, nil)
					if asserter.NoError(err) {
						wfEx, err := objects.QueryWorkflowExecutionByID(ctx, exID)
						if asserter.NoError(err) {
							for {
								taskExs, err := wfEx.GetChildrenTaskExecutions()
								if asserter.NoError(err) {
									for _, ex := range taskExs {
										if states.IsCompleted(ex.State) {
											log.Printf("wf is completed status is %v", ex.State)
											goto DOWN
										}
									}
								}
								time.Sleep(1 * time.Second)
							}
						}
					DOWN:
						log.Println(nil, "finished")
					}
				}

			}
		}
	}
}
