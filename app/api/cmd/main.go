package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"seraph/app/config"
	"seraph/app/db"
	"seraph/app/objects"
	"seraph/pkg/contextx"
)

var (
	ctx      = contextx.NewContext()
	workflow = flag.String("w", "", "workflow id")
)

func init() {
	cfg := &db.Config{
		Connection:  config.Config.Database.Connection,
		Debug:       config.Config.Database.Debug,
		PoolSize:    config.Config.Database.PoolSize,
		IdleTimeout: config.Config.Database.IdleTimeout,
	}
	if err := db.Init(cfg); err != nil {
		panic(err)
	}
}

type MGraph struct {
	vertexs        []string
	arc            [][]bool
	numVer, numEgd int
	isTrav         []bool
}

func (graph *MGraph) arcProc(nodes interface{}, xIndex int, completeNodes []string) {
	switch n := nodes.(type) {
	case []interface{}:
		for _, v := range n {
			yIndex, has := isContain(graph.vertexs, v.(string))
			if !has {
				panic("error")
			}
			graph.arc[xIndex][yIndex] = true
			if len(completeNodes) > 0 {
				for _, val := range completeNodes {
					nIndex, has := isContain(graph.vertexs, val)
					if !has {
						panic("error")
					}
					graph.arc[yIndex][nIndex] = true
				}
			}
		}
	case []string:
		for _, v := range n {
			yIndex, has := isContain(graph.vertexs, v)
			if !has {
				panic("error")
			}
			graph.arc[xIndex][yIndex] = true
			if len(completeNodes) > 0 {
				for _, val := range completeNodes {
					nIndex, has := isContain(graph.vertexs, val)
					if !has {
						panic("error")
					}
					graph.arc[yIndex][nIndex] = true
				}
			}
		}
	default:
		fmt.Printf("Not find the format %T\n", n)
		panic("error")
	}
}

func (graph *MGraph) createGraph(tasks map[string]interface{}) {
	// 初始化numVer
	graph.numVer = len(tasks)
	// 初始化isTrav
	graph.isTrav = make([]bool, graph.numVer)
	// 初始化vertexs
	graph.vertexs = make([]string, graph.numVer)
	i := 0
	for k, _ := range tasks {
		graph.vertexs[i] = k
		i++
	}
	// 初始化arc大小
	graph.arc = make([][]bool, graph.numVer)
	for _i := 0; _i < graph.numVer; _i++ {
		graph.arc[_i] = make([]bool, graph.numVer)
	}

	for k, v := range tasks {
		xIndex, ok := isContain(graph.vertexs, k)
		if !ok {
			fmt.Printf("error")
			break
		}
		switch val := v.(type) {
		case map[string]interface{}:
			var completeNodes []string
			if nodes, ok := val["on_complete"]; ok {
				switch n := nodes.(type) {
				case []interface{}:
					for _, nVal := range n {
						completeNodes = append(completeNodes, nVal.(string))
					}
				case []string:
					completeNodes = append(completeNodes, n...)
				default:
					fmt.Printf("Not find the format %T\n", n)
					panic("format error")
				}

			}
			if nodes, ok := val["on_success"]; ok {
				graph.arcProc(nodes, xIndex, completeNodes)
			}
			if nodes, ok := val["on_error"]; ok {
				graph.arcProc(nodes, xIndex, completeNodes)
			}
		}
	}
}

func (graph *MGraph) drawGraphMap() {
	// 找开始节点
	var mq []string
	for i := 0; i < graph.numVer; i++ {
		isStartNode := true
		for j := 0; j < graph.numVer; j++ {
			if graph.arc[j][i] {
				isStartNode = false
				break
			}
		}
		if isStartNode {
			mq = append(mq, graph.vertexs[i])
		}
	}
	fmt.Printf("START NODEs:%v\n", mq)
	// 画二维数组结构
	for i := -1; i < graph.numVer; i++ {
		for j := -1; j < graph.numVer; j++ {
			if i == -1 {
				//表格打o印
				if j == -1 {
					fmt.Printf("|%-30v", "")
				} else {
					if _, ok := isContain(mq, graph.vertexs[j]); ok {
						fmt.Printf("|%-30v", fmt.Sprintf("%v *", graph.vertexs[j]))
					} else {
						fmt.Printf("|%-30v", graph.vertexs[j])
					}
				}
			} else {
				if j == -1 {
					if _, ok := isContain(mq, graph.vertexs[i]); ok {
						fmt.Printf("|%-30v", fmt.Sprintf("%v *", graph.vertexs[i]))
					} else {
						fmt.Printf("|%-30v", graph.vertexs[i])
					}
				} else {
					if graph.arc[i][j] {
						fmt.Printf("|%-30v", 1)
					} else {
						fmt.Printf("|%-30v", 0)
					}
				}
			}
		}
		fmt.Print("\n")
		for s := -1; s < graph.numVer; s++ {
			fmt.Printf(" %-30v", "----------------------------")
		}
		fmt.Print("\n")
	}
}

func (graph *MGraph) runWorkflowGraph(wf string) {
	tasks, _ := objects.QueryTaskExecutions(ctx, wf, nil, nil, nil, nil)
	for i := 1; i < len(tasks); i++ {
		for j := 0; j < i; j++ {
			if tasks[j].CreatedAt.After(tasks[i].CreatedAt) {
				tmp := tasks[j]
				tasks[j] = tasks[i]
				tasks[i] = tmp
			}
		}
	}
	termimalOpTasks(tasks)
}

func termimalOpTasks(tasks []*objects.TaskExecution) {
	for {
		for i, v := range tasks {
			fmt.Printf("+ %d ID: %-30v|NAME: %-30v|STATUS: %-10v\n", i, v.ID, v.Name, v.State)
		}
		fmt.Print("输入想要查询task详情编码:")
		var flag int
		if _, err := fmt.Scanln(&flag); err != nil {
			goto BREAK
		} else {
			if flag >= len(tasks) {
				goto BREAK
			} else {
				if tasks[flag].Type == "ACTION" {
					action, _ := objects.QueryActionExecutionsByTaskID(ctx, tasks[flag].ID)
					fmt.Printf("- ID: %v\n-NAME: %v\n- STATUS: %v\n- INPUT: %v\n- OUTPUT: %v\n", action[0].ID, action[0].Name, action[0].State, action[0].Inputs, action[0].Outputs)
				} else {
					fmt.Print("敬请期待!")
				}
			}
		}
		// fmt.Print("是否继续（y/n）")
		// var sflag string
		// if _, err := fmt.Scanln(&sflag); err != nil {

		// }
	}
BREAK:
	fmt.Println("退出详情!")
	fmt.Scanln()
}

func isContain(array []string, target string) (int, bool) {
	for i, v := range array {
		if v == target {
			return i, true
		}
	}
	return 0, false
}

func tasksToMap(tasks string, ch chan MGraph) {
	taskMap := make(map[string]interface{})
	json.Unmarshal([]byte(tasks), &taskMap)
	graph := MGraph{}
	graph.createGraph(taskMap["tasks"].(map[string]interface{}))
	ch <- graph
}

func main() {
	flag.Parse()
	wf, err := objects.QueryWorkflowExecutionByID(ctx, *workflow)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("WORKFLOW:")
	fmt.Printf("  %-20v %v\n", "startTime:", wf.CreatedAt)
	fmt.Printf("  %-20v %v\n", "finishedTime:", wf.FinishedAt)
	fmt.Printf("  %-20v %v\n", "status:", wf.State)
	fmt.Printf("  %-20v %v\n", "statusInfo:", wf.StateInfo)

	ch := make(chan MGraph)
	go tasksToMap(wf.Spec, ch)
	graph := <-ch
	for {
		fmt.Print("1. 绘制工作流图\n")
		fmt.Print("2. 打印运行的task\n")
		fmt.Print("输入一下操作编码(其他则退出):")
		var flag int
		if _, err := fmt.Scanln(&flag); err != nil {
			goto END
		} else {
			switch flag {
			case 1:
				fmt.Printf("TASK Graph:\n")
				graph.drawGraphMap()
			case 2:
				graph.runWorkflowGraph(*workflow)
			default:
				goto END
			}
		}
	}

END:
	fmt.Println("退出")
}
