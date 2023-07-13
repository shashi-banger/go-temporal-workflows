package workflows

import (
	"fmt"
	"log"
	"time"

	"github.com/heimdalr/dag"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	Pending   ActivityStatus = "pending"
	Scheduled ActivityStatus = "scheduled"
	Completed ActivityStatus = "completed"
)

type Activity struct {
	ActivityParams
	ActivityStatus ActivityStatus
}

type WorkflowCtxt struct {
	ActivityDag   *dag.DAG
	NumActivities int
}

func CreateWorkflowCtxt(workflow *Workflow) *WorkflowCtxt {
	wfCtxt := WorkflowCtxt{}
	allActivities := []Activity{}
	for _, activity := range workflow.Activities {
		a := Activity{}
		a.ActivityParams = activity
		a.ActivityStatus = Pending
		allActivities = append(allActivities, a)
	}
	wfCtxt.ActivityDag = CreateActivityDAG(allActivities)
	return &wfCtxt
}

func ApiWorkflow(ctx workflow.Context, model *Workflow) (string, error) {
	var output string
	activityResponses := make(map[string]string, model.NumActivities)

	// RetryPolicy specifies how to automatically handle retries if an Activity fails.
	retrypolicy := &temporal.RetryPolicy{
		InitialInterval:        time.Second,
		BackoffCoefficient:     2.0,
		MaximumInterval:        100 * time.Second,
		MaximumAttempts:        2, // unlimited retries
		NonRetryableErrorTypes: []string{"RequestMarshalError"},
	}

	options := workflow.ActivityOptions{
		// Timeout options specify when to automatically timeout Activity functions.
		StartToCloseTimeout: 1 * time.Minute,
		// Optionally provide a customized RetryPolicy.
		// Temporal retries failed Activities by default.
		RetryPolicy: retrypolicy,
	}

	wfCtxt := CreateWorkflowCtxt(model)

	// Apply the options.
	ctx = workflow.WithActivityOptions(ctx, options)

	workflowId := workflow.GetInfo(ctx).WorkflowExecution.ID

	for {
		activities := GetActivitiesForProcessing(wfCtxt.ActivityDag)
		if len(activities) == 0 {
			break
		}
		for _, activityName := range activities {
			log.Println("Processing activity: ", activityName)
			activity := GetActivityFromID(wfCtxt.ActivityDag, activityName)
			log.Println("Activity: ", activity)
			// Execute activity
			activityErr := workflow.ExecuteActivity(ctx, ActivityProcessAPICall, activity, activityResponses, workflowId).Get(ctx, &output)
			if activityErr != nil {
				// Cleanup
				activityErr := workflow.ExecuteActivity(ctx, CleanupActivity, activityResponses).Get(ctx, &output)
				if activityErr != nil {
					return "",
						fmt.Errorf("Failed to cleanup resources: %w", activityErr)
				}
				return "", activityErr
			}
			activityResponses[activityName] = output
			activity.ActivityStatus = Completed
		}
	}

	return "Success", nil
}
