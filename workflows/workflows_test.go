package workflows

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	yaml "gopkg.in/yaml.v3"
)

const TaskQueName = "casApiWorkflowQueue"

func createWorkflowModel(t *testing.T, wf string) *Workflow {
	// open wf file and parse yaml file into map[string]interface{}
	wfData := make(map[string]interface{})
	// Read data from wf files
	wfBytes, err := os.ReadFile(wf)
	if err != nil {
		t.Fatalf("Failed to read workflow file: %v", err)
	}
	// Unmarshal yaml data into map[string]interface{}
	log.Println(string(wfBytes))
	err = yaml.Unmarshal(wfBytes, &wfData)
	if err != nil {
		t.Fatalf("Failed to unmarshal workflow file: %v", err)
	}

	// Create workflow model
	wfModel := Workflow{}

	// Create workflow model from wfData
	for _, v := range wfData["activities"].([]interface{}) {
		activity := v.(map[string]interface{})
		activityValue := ActivityParams{}
		activityValue.Name = activity["name"].(string)
		activityValue.Type = ActivityType(activity["type"].(string))
		activityValue.CompletenessCondition = activity["completeness_condition"].(string)

		activityValue.RequestParams = RequestParams{}
		activityValue.RequestParams.Path = activity["request_params"].(map[string]interface{})["path"].(string)
		activityValue.RequestParams.Method = activity["request_params"].(map[string]interface{})["method"].(string)
		activityValue.RequestParams.Body = activity["request_params"].(map[string]interface{})["body"].(map[string]interface{})

		wfModel.Activities = append(wfModel.Activities, activityValue)
	}
	wfModel.NumActivities = len(wfModel.Activities)
	return &wfModel

}

func temporalWorker() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("Unable to create Temporal client.", err)
	}
	defer c.Close()

	w := worker.New(c, TaskQueName, worker.Options{})

	// This worker hosts both Workflow and Activity functions.
	w.RegisterWorkflow(ApiWorkflow)
	w.RegisterActivity(ActivityProcessAPICall)
	w.RegisterActivity(CleanupActivity)

	// Start listening to the Task Queue.
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("unable to start Worker", err)
	}
}

func startWorkflow(t *testing.T) {
	wfModel := createWorkflowModel(t, "testdata/eg_workflow.yaml")

	c, err := client.Dial(client.Options{})

	if err != nil {
		log.Fatalln("Unable to create Temporal client:", err)
	}

	defer c.Close()

	options := client.StartWorkflowOptions{
		ID:        "liv-hooks-mstabr",
		TaskQueue: TaskQueName,
	}

	log.Printf("Starting Workflow: %s\n", options.ID)

	we, err := c.ExecuteWorkflow(context.Background(), options, ApiWorkflow, wfModel)
	if err != nil {
		log.Fatalln("Unable to start the Workflow:", err)
	}

	log.Printf("WorkflowID: %s RunID: %s\n", we.GetID(), we.GetRunID())

	var result string

	err = we.Get(context.Background(), &result)

	if err != nil {
		log.Fatalln("Unable to get Workflow result:", err)
	}

	log.Println(result)

}

func Test_Workflow1(t *testing.T) {
	os.Setenv("CAS_SERVER", "http://localhost:9200")
	// Start a mock server
	go initMockServer()

	time.Sleep(2 * time.Second) // Wait for mock server to start

	go temporalWorker()

	startWorkflow(t)
}
