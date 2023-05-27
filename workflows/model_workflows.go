package workflows

// Create an enum of workflow types
type ActivityType string
type ActivityStatus string

const (
	Create ActivityType = "resource_create"
	Update ActivityType = "update"
)

const (
	Pending   ActivityStatus = "pending"
	Scheduled ActivityStatus = "scheduled"
	Completed ActivityStatus = "completed"
)

type Activity struct {
	//ActivityName is of the form {resourceName}.{id} id can be 1, 2, ... such that even
	//the workflow can express multiple instances of same resource
	ActivityName   string
	ActivityStatus ActivityStatus
	ActivityType   ActivityType
	//DependsOnActivities []string // Array of activity names
	ResourcePath          string
	ResourceRequestParams map[string]interface{}
}

type WorkflowModel struct {
	NumActivities int
	Activities    []Activity
}
