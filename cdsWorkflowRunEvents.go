package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/patrickmn/go-cache"
	"io/ioutil"
	"time"
)

func main() {
	store := cache.New(24*time.Hour, 10*time.Minute)

	c := cdsclient.Config{
		Host:  "",
		User:  "",
		Token: "",
	}

	client := cdsclient.New(c)

	ctx := context.Background()
	chanSSE := make(chan cdsclient.SSEvent)
	go client.EventsListen(ctx, chanSSE)
	go computeEvent(ctx, chanSSE, store)

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.GET("/pipelines", func(c *gin.Context) {
		alls := store.Items()
		datas := make([]*Data, 0, len(alls))
		for _, item := range alls {
			data := item.Object.(*Data)
			datas = append(datas, data)
		}
		c.JSON(200, datas)
	})
	r.GET("/pipelines/count", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"count": store.ItemCount(),
		})
	})
	r.Run() // listen and serve on 0.0.0.0:8080

}

func computeEvent(ctx context.Context, chanSSE <-chan cdsclient.SSEvent, store *cache.Cache) {
	for {
		select {
		case <-ctx.Done():
			ctx.Err()
			fmt.Println("Context done")
			return
		case evt := <-chanSSE:
			var e sdk.Event
			content, _ := ioutil.ReadAll(evt.Data)
			_ = json.Unmarshal(content, &e)
			if e.EventType == "" {
				continue
			}
			switch e.EventType {
			case "sdk.EventRunWorkflow":
				var eventWR sdk.EventRunWorkflow
				if err := mapstructure.Decode(e.Payload, &eventWR); err != nil {
					fmt.Printf("unable to read payload of EventRunWorkflow: %v  %+v", err, e.Payload)
					continue
				}
			case "sdk.EventRunWorkflowNode":
				var eventNR sdk.EventRunWorkflowNode
				if err := mapstructure.Decode(e.Payload, &eventNR); err != nil {
					fmt.Printf("unable to read payload of EventRunWorkflowNode: %v  %+v", err, e.Payload)
					continue
				}
				cacheKey := fmt.Sprintf("%s-%d-%d-%s", e.WorkflowName, e.WorkflowRunNum, e.WorkflowRunNumSub, eventNR.NodeName)
				data := transform(e, eventNR)
				store.Set(cacheKey, &data, 24*time.Hour)
			}
		}
	}
}

func transform(e sdk.Event, eventNR sdk.EventRunWorkflowNode) Data {
	d := Data{
		WorkflowName: e.WorkflowName,
		Number:       e.WorkflowRunNum,
		SubNumber:    e.WorkflowRunNumSub,
		NodeName:     eventNR.NodeName,
		Stages:       make([]StageData, len(eventNR.StagesSummary)),
	}
	for i, ss := range eventNR.StagesSummary {
		sd := StageData{
			Name:   ss.Name,
			Status: ss.Status.String(),
			Jobs:   make([]JobData, len(ss.Jobs)),
		}
		for idx, j := range ss.Jobs {
			jd := JobData{
				Name: j.Action.Name,
			}
			for _, k := range ss.RunJobsSummary {
				if k.Job.PipelineActionID == j.PipelineActionID {
					jd.Status = k.Status
				}
			}
			sd.Jobs[idx] = jd
		}
		d.Stages[i] = sd
	}
	return d
}
