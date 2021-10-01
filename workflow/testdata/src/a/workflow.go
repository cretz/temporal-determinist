package a

import (
	"time"

	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func PrepWorkflow() {
	var wrk worker.Worker
	wrk.RegisterWorkflow(WorkflowNop)
	wrk.RegisterWorkflow(WorkflowCallTime)
}

func WorkflowNop(ctx workflow.Context) error {
	return nil
}

func WorkflowCallTime(ctx workflow.Context) error {
	time.Now()
	return nil
}
