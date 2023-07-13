package workflows

// Create an enum of workflow types
type ActivityType string
type ActivityStatus string

const (
	ApiCall ActivityType = "api_call"
)

type RequestParams struct {
	Path   string
	Method string
	Body   map[string]interface{}
}

type ActivityParams struct {
	Name                  string
	Type                  ActivityType
	RequestParams         RequestParams
	CompletenessCondition string
}

type Workflow struct {
	NumActivities int
	Activities    []ActivityParams
}
