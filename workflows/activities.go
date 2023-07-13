package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/davegardnerisme/deephash"
	resty "github.com/go-resty/resty/v2"
	"github.com/maja42/goval"
	"github.com/tidwall/gjson"
)

var errResourceNotFound = errors.New("ResourceNotFound")

type ResourceUrl string
type ActivityResult map[string]ResourceUrl

func GetResourceWithRetries(resource_url string) (*resty.Response, error) {
	client := resty.New().
		SetRetryCount(5).
		// Override initial retry wait time.
		// Default is 100 milliseconds.
		SetRetryWaitTime(1 * time.Second).
		// MaxWaitTime can be overridden as well.
		// Default is 2 seconds.
		SetRetryMaxWaitTime(20 * time.Second)
	resp, err := client.R().
		Get(resource_url)
	return resp, err
}

func GetResourceIfExists(resourceCollectionUrl string,
	workflowId string, activityName string) (string, error) {
	hv := deephash.Hash(map[string]string{"workflowId": workflowId, "activityName": activityName})
	hvs := fmt.Sprintf("%x", hv)
	fmt.Println("x-request-id: ", string(hvs))
	client := resty.New().
		SetRetryCount(5).
		// Override initial retry wait time.
		// Default is 100 milliseconds.
		SetRetryWaitTime(1 * time.Second).
		// MaxWaitTime can be overridden as well.
		// Default is 2 seconds.
		SetRetryMaxWaitTime(20 * time.Second)
	fmt.Println("GetResourceIfExists: ", resourceCollectionUrl)
	resp, err := client.R().
		AddRetryCondition(func(r *resty.Response, err error) bool {
			fmt.Println("GetResourceIfExists: statusCode=", r.StatusCode())
			return (r.StatusCode() == http.StatusTooManyRequests ||
				r.StatusCode() == http.StatusServiceUnavailable)
		}).
		SetHeader("x-request-id", string(hvs)).
		Get(resourceCollectionUrl)
	if err != nil {
		return "", err
	} else {
		if resp.StatusCode() == 200 {
			// Read response
			resourceBody := resp.Body()
			resourceId := gjson.Get(string(resourceBody), "meta.resource_id").String()
			resourceUrl, _ := url.JoinPath(resourceCollectionUrl, resourceId)
			return resourceUrl, nil
		} else if resp.StatusCode() == 404 {
			return "", errResourceNotFound
		} else {
			return "", fmt.Errorf("ResourceGetError: %d", resp.StatusCode())
		}
	}

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
		respObj := map[string]interface{}{}
		json.Unmarshal(response.Body(), &respObj)
		value := GetValue(respObj, m.value)
		obj = req
		for _, p := range m.pathArr[:len(m.pathArr)-1] {
			obj = obj[p].(map[string]interface{})
		}
		obj[m.pathArr[len(m.pathArr)-1]] = value
	}
	return
}

func getResourceServerUrl(resourcePath string) string {
	cas_server := os.Getenv("CAS_SERVER")
	post_endpoint, _ := url.JoinPath(cas_server, resourcePath)
	return post_endpoint
}

func createResource(ctx context.Context, activity *Activity,
	workFlowId string, reqJson []byte) (error, string) {
	fmt.Println("createResource: ", activity.RequestParams.Path)
	post_endpoint := getResourceServerUrl(activity.ActivityParams.RequestParams.Path)
	client := resty.New()
	hv := deephash.Hash(map[string]string{"workflowId": workFlowId, "activityName": activity.Name})
	hvs := fmt.Sprintf("%x", hv)
	resp, err := client.R().
		AddRetryCondition(func(r *resty.Response, err error) bool {
			return (r.StatusCode() == http.StatusTooManyRequests ||
				r.StatusCode() == http.StatusServiceUnavailable)
		}).
		SetHeader("x-request-id", string(hvs)).
		SetBody(reqJson).Post(post_endpoint)

	respMap := map[string]interface{}{}
	if err != nil {
		return fmt.Errorf("CreateResourceError: Post Failed"), ""
	} else {
		if resp.StatusCode() == http.StatusOK {
			// Read response
			respBody := resp.Body()
			json.Unmarshal(respBody, &respMap)
			resource_id := respMap["meta"].(map[string]interface{})["resource_id"].(string)
			resource_url, _ := url.JoinPath(post_endpoint, resource_id)
			return nil, resource_url
		} else {
			return fmt.Errorf("CreateResourceError"), ""
		}
	}
}

func WaitForCompletenessConditionCriteria(completenessCondition string, resourceUrl string) error {

	sleepTime := 5
	for {
		var respMap map[string]interface{}
		resp, err := GetResourceWithRetries(resourceUrl)
		if err != nil {
			// Wraps error with custom error
			return fmt.Errorf("GetResourceError: %w", err)
		}

		json.Unmarshal(resp.Body(), &respMap)
		// Replace any single quoted string literals to double quoted string literals
		// e.g " {{ foo.boo }} == 'created' " will be transformed to
		//  {{ foo.boo }} == "created"
		var re = regexp.MustCompile(`'([A-Za-z0-9\.\-_]*)'`)
		s := re.ReplaceAllString(completenessCondition, `"$1"`)

		re1 := regexp.MustCompile(`{{\s*([A-Za-z0-9_\-\.]*)\s*}}`)
		ms := re1.FindAllStringSubmatch(s, -1)
		for _, v := range ms {
			value := GetValue(respMap, v[1])
			strVal, ok := value.(string)
			if ok {
				strVal = fmt.Sprintf("%q", strVal)
				s = strings.Replace(s, v[0], strVal, -1)
			} else {
				// TODO: Handle non string values
				return fmt.Errorf("NonStringValuesNotSupported")
			}
		}

		fmt.Println("+++++++++++++++", s)

		eval := goval.NewEvaluator()
		result, _ := eval.Evaluate(s, nil, nil) // Returns <true, nil>
		resultB := result.(bool)
		if resultB {
			return nil
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
	// TODO: Handle timeout
	return fmt.Errorf("Workflow completeness check control should never reach here")
}

//  1. Check if the resource request body has any value expressions depending on other activities
//  2. If yes, then resolve the value expressions and create the request body
//  3. Check if the resource exists. This is important for idempotency of activity. Suppose a
//     resource has been created and the acitvity was in post condition wait state. Now the activity
//     is retried. In this case, the activity should not create the resource again
//
// 4. If the resource does not exist, then create the resource
// 5. If the resource exists, then check for post condition criteria to be met
func ActivityProcessAPICall(ctx context.Context, activity *Activity,
	activityResults map[string]string, workFlowId string) (string, error) {

	var resourceUrl string
	ResolveValueExpressions(activity.RequestParams.Body, activityResults)

	reqJson, err := json.Marshal(activity.RequestParams.Body)
	if err != nil {
		fmt.Errorf("RequestMarshalError")
		return "", err
	}

	switch activity.RequestParams.Method {
	case "POST":
		// Check if resource exists
		resourceUrl, err = GetResourceIfExists(
			getResourceServerUrl(activity.ActivityParams.RequestParams.Path), workFlowId, activity.Name)
		if err != nil {
			fmt.Println("GetResourceError error:", err)
		}
		if err == errResourceNotFound {
			err, resourceUrl = createResource(ctx, activity, workFlowId, reqJson)
			if err != nil {
				fmt.Println("CreateResourceError error:", err)
				return "", fmt.Errorf("ActivityProcessAPICall failed: %w", err)
			} else {
				fmt.Println("CreateResource success")
			}
		} else if err != nil {
			return "", fmt.Errorf("ActivityProcessAPICall failed: %w", err)
		} else {
			fmt.Println("Resource already exists")
		}
	case "GET":
		fmt.Println("GET: To be implemented")
	}

	// Check if post condition criteria is met
	if activity.CompletenessCondition != "" {
		// Check if post condition criteria is met
		err := WaitForCompletenessConditionCriteria(activity.CompletenessCondition, resourceUrl)
		if err == nil {
			activityResults[activity.Name] = resourceUrl
			return resourceUrl, nil
		} else { // Post condition criteria not met
			return "", fmt.Errorf("WaitForCompletenessConditionCriteriaError")
		}
	}

	return "", fmt.Errorf("Activity completion criteria not met")
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
