package workflows

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"testing"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	yaml "gopkg.in/yaml.v3"
)

const TaskQueName = "casApiWorkflowQueue"

// To run all tests in this file from the root of the repo:
// go test -v ./workflows/
func Test_FindPathAndValuesWithPattern(t *testing.T) {
	tc1 := map[string]interface{}{
		"key1": "value1",
		"key2": map[string]interface{}{
			"key3": "{{ingests.1.response.url}}",
			"key4": "value4",
		},
		"key5": "{{ingests.1.response.url}}",
	}

	output := FindPathAndValuesWithPattern(regexp.MustCompile("{{.*}}"), tc1, []string{}, []Match{})
	fmt.Println(output)
}

func createWorkflowModel(t *testing.T, wf string) *WorkflowModel {
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
	wfModel := WorkflowModel{}

	// Create workflow model from wfData
	for k, v := range wfData["activities"].(map[string]interface{}) {
		activity := v.(map[string]interface{})
		activityValue := Activity{}
		activityValue.ActivityName = k
		activityValue.ActivityStatus = Pending

		activityParams := activity["params"].(map[string]interface{})

		activityValue.ResourceRequestParams = activityParams["resource_request_params"].(map[string]interface{})
		activityValue.ResourcePath = activityParams["resource_path"].(string)
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
	w.RegisterWorkflow(CasWorkflow)
	w.RegisterActivity(CreateResourceActivity)
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

	we, err := c.ExecuteWorkflow(context.Background(), options, CasWorkflow, wfModel)
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
