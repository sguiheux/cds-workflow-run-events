package main

type Data struct {
	WorkflowName string
	Number       int64
	SubNumber    int64
	NodeName     string
	Stages       []StageData
}

type StageData struct {
	Jobs   []JobData
	Name   string
	Status string
}

type JobData struct {
	Status string
	Name   string
}
