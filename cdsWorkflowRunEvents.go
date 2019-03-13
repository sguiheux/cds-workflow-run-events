package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"github.com/patrickmn/go-cache"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func main() {
	//debug := flag.Bool("debug", false, "debug mode")
	store := cache.New(24*time.Hour, 10*time.Minute)

	ctx := context.Background()
	chanSSE := make(chan cdsclient.SSEvent, 10)
	if true {
		fmt.Println("Creating CDS Client")
		c := cdsclient.Config{
			Host:  os.Getenv("CDS_HOST"),
			User:  os.Getenv("CDS_USER"),
			Token: os.Getenv("CDS_TOKEN"),
			InsecureSkipVerifyTLS: true,
		}
		client := cdsclient.New(c)
		fmt.Println("Connection to SSE")
		go client.EventsListen(ctx, chanSSE)
	} else {
		go mock(chanSSE)
	}
	go computeEvent(ctx, chanSSE, store)

	r := gin.Default()
	r.Use(CORSMiddleware())
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

	r.Run("0.0.0.0:8080") // listen and serve on 0.0.0.0:8080

}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
		} else {
			c.Next()
		}
	}
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
			fmt.Printf("Received: %s\n", e.EventType)
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
				cacheKey := fmt.Sprintf("%s-%d-%s", e.WorkflowName, e.WorkflowRunNum, e.WorkflowName)
				fmt.Printf("Cache key: %s\n", cacheKey)
				var eventNR sdk.EventRunWorkflowNode
				if err := mapstructure.Decode(e.Payload, &eventNR); err != nil {
					fmt.Printf("unable to read payload of EventRunWorkflowNode: %v  %+v", err, e.Payload)
					continue
				}

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
			ID:     ss.ID,
			Name:   ss.Name,
			Status: ss.Status.String(),
			Jobs:   make([]JobData, len(ss.Jobs)),
		}
		for idx, j := range ss.Jobs {
			jd := JobData{
				ID:   j.PipelineActionID,
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
