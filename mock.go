package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatih/structs"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"time"
)

func mock(c chan cdsclient.SSEvent) {
	cpt := 0
	for {
		time.Sleep(1 * time.Second)
		cpt++
		e, w := createEvent(cpt)

		eventToSend := cdsclient.SSEvent{}
		e.Payload = structs.Map(w)
		b, _ := json.Marshal(e)
		eventToSend.Data = bytes.NewBuffer(bytes.TrimSpace(b))

		fmt.Println("Send")
		c <- eventToSend

		go continueWorkflow(cpt, w, c)

		if cpt == 10 {
			break
		}
	}
}

func continueWorkflow(id int, rwn sdk.EventRunWorkflowNode, c chan cdsclient.SSEvent) {
	cpt := 0
	for {
		switch cpt {
		case 0:
			// Start stage 1
			rwn.StagesSummary[0].Status = "Waiting"
		case 1:
			// Start stage 1 job 1
			rwn.StagesSummary[0].Status = "Building"
			rwn.StagesSummary[0].RunJobsSummary = append(rwn.StagesSummary[0].RunJobsSummary, sdk.WorkflowNodeJobRunSummary{
				Status: "Building",
				Job: sdk.ExecutedJobSummary{
					PipelineActionID: 1,
				},
			})
		case 2:
			// End stage 1 job 1
			rwn.StagesSummary[0].RunJobsSummary[0].Status = "Success"
			rwn.StagesSummary[0].RunJobsSummary[0].Job.PipelineActionID = 1
		case 3:
			// Start stage 1 job 2
			rwn.StagesSummary[0].RunJobsSummary = append(rwn.StagesSummary[0].RunJobsSummary, sdk.WorkflowNodeJobRunSummary{
				Status: "Building",
				Job: sdk.ExecutedJobSummary{
					PipelineActionID: 2,
				},
			})
		case 4:
			// End stage 1 job 2
			rwn.StagesSummary[0].RunJobsSummary[1].Status = "Success"
			rwn.StagesSummary[0].RunJobsSummary[1].Job.PipelineActionID = 2
		case 5:
			// End stage 1
			rwn.StagesSummary[0].Status = "Success"
		case 6:
			// Start stage 2
			rwn.StagesSummary[1].Status = "Waiting"
		case 7:
			// Start stage 2 job 1
			rwn.StagesSummary[1].Status = "Building"
			rwn.StagesSummary[1].RunJobsSummary = append(rwn.StagesSummary[1].RunJobsSummary, sdk.WorkflowNodeJobRunSummary{
				Status: "Building",
				Job: sdk.ExecutedJobSummary{
					PipelineActionID: 1,
				},
			})
		case 8:
			// End stage 2 job 1
			rwn.StagesSummary[1].RunJobsSummary[0].Status = "Success"
			rwn.StagesSummary[1].RunJobsSummary[0].Job.PipelineActionID = 1
		case 9:
			// Start stage 2 job 2
			rwn.StagesSummary[1].RunJobsSummary = append(rwn.StagesSummary[1].RunJobsSummary, sdk.WorkflowNodeJobRunSummary{
				Status: "Building",
				Job: sdk.ExecutedJobSummary{
					PipelineActionID: 2,
				},
			})
		case 10:
			// End stage 2 job 2
			rwn.StagesSummary[1].RunJobsSummary[1].Status = "Success"
			rwn.StagesSummary[1].RunJobsSummary[1].Job.PipelineActionID = 2
		case 11:
			// End stage 2
			rwn.StagesSummary[1].Status = "Success"
		}
		cpt++

		e := sdk.Event{
			WorkflowRunNumSub: 0,
			WorkflowRunNum:    1,
			WorkflowName:      fmt.Sprintf("w%d", id),
			EventType:         "sdk.EventRunWorkflowNode",
		}
		eventToSend := cdsclient.SSEvent{}
		e.Payload = structs.Map(rwn)
		b, _ := json.Marshal(e)
		eventToSend.Data = bytes.NewBuffer(bytes.TrimSpace(b))
		fmt.Println("Send continue")
		c <- eventToSend

		if cpt == 11 {
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func createEvent(id int) (sdk.Event, sdk.EventRunWorkflowNode) {

	e := sdk.Event{
		WorkflowRunNumSub: 0,
		WorkflowRunNum:    1,
		WorkflowName:      fmt.Sprintf("w%d", id),
		EventType:         "sdk.EventRunWorkflowNode",
	}
	workflowEvent := createRunWorkflowNodeEvent()

	return e, workflowEvent
}

func createRunWorkflowNodeEvent() sdk.EventRunWorkflowNode {
	erwn := sdk.EventRunWorkflowNode{
		StagesSummary: make([]sdk.StageSummary, 0),
		NodeName:      "node",
		SubNumber:     0,
		Number:        1,
	}
	erwn.StagesSummary = append(erwn.StagesSummary, createStageSummary(1))
	erwn.StagesSummary = append(erwn.StagesSummary, createStageSummary(2))
	return erwn
}

func createStageSummary(id int) sdk.StageSummary {
	ss := sdk.StageSummary{
		ID:     int64(id),
		Status: "",
		Name:   fmt.Sprintf("Stage%d", id),
		Jobs: []sdk.Job{
			{
				PipelineActionID: 1,
				Action: sdk.Action{
					Name: "Action1",
				},
			},
			{
				PipelineActionID: 2,
				Action: sdk.Action{
					Name: "Action2",
				},
			},
		},
		RunJobsSummary: make([]sdk.WorkflowNodeJobRunSummary, 0, 2),
	}

	return ss
}
