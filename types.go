package main

type Data struct {
	WorkflowName string      `json:"workflow_name"`
	Number       int64       `json:"num"`
	SubNumber    int64       `json:"sub_num"`
	NodeName     string      `json:"node_name"`
	Stages       []StageData `json:"stages"`
}

type StageData struct {
	Jobs   []JobData `json:"jobs"`
	Name   string    `json:"name"`
	Status string    `json:"status"`
}

type JobData struct {
	Status string `json:"status"`
	Name   string `json:"name"`
}
