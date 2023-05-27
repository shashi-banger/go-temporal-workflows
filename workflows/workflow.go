package workflows

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func GetResourceIfExists(ctx context.Context, resourceCollectionUrl string, workflowId string, activityName string) (bool, string, error) {
	log.Println("GetResourceIfExists: ", resourceCollectionUrl, workflowId, activityName)
	getUrl, _ := url.Parse(resourceCollectionUrl)
	params := url.Values{}
	params.Add("workflow_id", workflowId)
	params.Add("activity_name", activityName)
	getUrl.RawQuery = params.Encode()

	log.Println("GetResourceIfExists: ", getUrl.String())
	// Make http Get request to get resource
	resp, err := http.Get(getUrl.String())
	if err != nil {
		panic(err)
	}
	if err != nil {
		return false, "", err
	} else {
		if resp.StatusCode == 200 {
			// Read response
			respBody, _ := ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
			return true, string(respBody), nil
		} else if resp.StatusCode == 404 {
			return false, "", nil
		} else {
			return false, "", fmt.Errorf("GetResourceError")
		}
	}
}

func PostWithHeaders(url string, headers map[string]string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	return client.Do(req)
}

func ResolveValueExpressions(req map[string]interface{}, activityResponses map[string]string) {
	allMatches := []Match{}
	var obj map[string]interface{}
	allMatches = FindPathAndValuesWithPattern(regexp.MustCompile("{{.*}}"), req, []string{}, allMatches)
	log.Println("len allMatches: ", len(allMatches))

	for _, m := range allMatches {
		activityName := GetActivityNameFromValueExpression(m.value)
		log.Println("ResolveValueExpressions: ", activityName)
		activityResponse := activityResponses[activityName]
		log.Println("ResolveValueExpressions: ", activityResponse)
		response, _ := GetResourceWithRetries(activityResponse)
		responseBody, _ := ioutil.ReadAll(response.Body)
		respObj := map[string]interface{}{}
		json.Unmarshal(responseBody, &respObj)
		value := GetValue(respObj, m.value)
		obj = req
		for _, p := range m.pathArr[:len(m.pathArr)-1] {
			obj = obj[p].(map[string]interface{})
		}
		obj[m.pathArr[len(m.pathArr)-1]] = value
	}
	return
}

// Activity CreateResource
func CreateResourceActivity(ctx context.Context, activity *Activity, activityResponses map[string]string, workFlowId string) (string, error) {
	var respMap map[string]interface{}

	log.Println("CreateResourceActivity:", activity.ResourceRequestParams)

	// Resolve value expressions
	ResolveValueExpressions(activity.ResourceRequestParams, activityResponses)
	// Create json from activity.ResourceRequest
	reqJson, err := json.Marshal(activity.ResourceRequestParams)
	if err != nil {
		fmt.Errorf("RequestMarshalError")
		return "", err
	}

	// Get CAS server from environment variable
	cas_server := os.Getenv("CAS_SERVER")
	log.Println("CAS_SERVER:", cas_server)
	post_endpoint, _ := url.JoinPath(cas_server, activity.ResourcePath)

	resourceExists, resourceRep, err := GetResourceIfExists(ctx, post_endpoint, workFlowId, activity.ActivityName)
	log.Println("ResourceExists:", resourceExists, "ResourceRep:", resourceRep, "Error:", err)
	if !resourceExists && err == nil {
		// If resource does not exist in backend only then create it
		// Make http Post request to create resource
		resp, err := PostWithHeaders(post_endpoint, map[string]string{"x-workflow-id": workFlowId,
			"x-activity-name": activity.ActivityName}, reqJson)
		if err != nil {
			fmt.Errorf("Create Resource Error")
			return "", err
		} else {
			defer resp.Body.Close()
		}
		// Read response
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Errorf("RequestMarshalError")
			return "", err
		}
		// Unmarshal response into map[string]interface{}
		json.Unmarshal(respBody, &respMap)
	} else if resourceExists {
		json.Unmarshal([]byte(resourceRep), &respMap)
	} else {
		return "", fmt.Errorf("Unknown Server Error")
	}

	resource_id := respMap["meta"].(map[string]interface{})["resource_id"].(string)
	resource_url, _ := url.JoinPath(post_endpoint, resource_id)

	// Wait for the resource to be created and the status to become created
	sleepTime := 5
	for {
		var respMap map[string]interface{}
		resp, err := GetResourceWithRetries(resource_url)
		if err != nil {
			fmt.Errorf("GetResourceError")
			return "", err
		}
		respBody, err := ioutil.ReadAll(resp.Body)
		json.Unmarshal(respBody, &respMap)
		if respMap["meta"].(map[string]interface{})["status"].(string) == "created" {
			break
		} else {
			time.Sleep(time.Duration(sleepTime) * time.Second)
			sleepTime = sleepTime * 2
			if sleepTime > 700 {
				sleepTime = 5
			} else {
				sleepTime = sleepTime * 2
			}
		}
	}
	log.Println("CreateResourceActivity Completed:", activity.ResourceRequestParams)
	return resource_url, nil
}

func GetResourceWithRetries(resource_url string) (*http.Response, error) {
	client := retryablehttp.NewClient()
	client.RetryWaitMin = 3 * time.Second
	client.RetryWaitMax = 60 * time.Second
	client.RetryMax = 5

	req, err := retryablehttp.NewRequest("GET", resource_url, nil)

	resp, err := client.Do(req)
	return resp, err
}

func CleanupActivity(ctx context.Context, resourceUrls []string) (string, error) {
	for _, resourceUrl := range resourceUrls {
		// Make http Delete request to delete resource
		resp, err := http.NewRequest("DELETE", resourceUrl, nil)
		if err != nil {
			fmt.Errorf("ResourceDeleteError")
			return "", err
		} else {
			defer resp.Body.Close()
		}
	}
	return "Success", nil
}

func CasWorkflow(ctx workflow.Context, model *WorkflowModel) (string, error) {
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

	dag := CreateActivityDAG(model.Activities)

	// Apply the options.
	ctx = workflow.WithActivityOptions(ctx, options)

	workflowId := workflow.GetInfo(ctx).WorkflowExecution.ID

	for {
		activities := GetActivitiesForProcessing(dag)
		if len(activities) == 0 {
			break
		}
		for _, activityName := range activities {
			log.Println("Processing activity: ", activityName)
			activity := GetActivityFromID(dag, activityName)
			log.Println("Activity: ", activity)
			// Execute activity
			activityErr := workflow.ExecuteActivity(ctx, CreateResourceActivity, activity, activityResponses, workflowId).Get(ctx, &output)
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
