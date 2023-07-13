package workflows

import (
	"regexp"
	"strings"

	"github.com/heimdalr/dag"
)

func FindDependencies(activity *Activity) []string {
	dependencies := []string{}
	allMatches := FindPathAndValuesWithPattern(regexp.MustCompile("{{.*}}"),
		activity.RequestParams.Body, []string{}, []Match{})
	for _, match := range allMatches {
		matchVal := strings.TrimSpace(match.value[2 : len(match.value)-2])
		activityName := GetActivityNameFromValueExpression(matchVal)
		dependencies = append(dependencies, activityName)
	}

	return dependencies
}

func CreateActivityDAG(activities []Activity) *dag.DAG {
	d := dag.NewDAG()

	for i := 0; i < len(activities); i++ {
		d.AddVertexByID(activities[i].Name, &activities[i])
	}

	for _, activity := range activities {
		// Find Dependencies
		dependencies := FindDependencies(&activity)
		for _, dependency := range dependencies {
			d.AddEdge(dependency, activity.Name)
		}
	}

	return d
}

func GetActivityFromID(d *dag.DAG, id string) *Activity {
	vertex, _ := d.GetVertex(id)
	return vertex.(*Activity)
}

func GetActivitiesForProcessing(d *dag.DAG) []string {
	activities := []string{}
	for k, v := range d.GetVertices() {
		parents, _ := d.GetParents(k)
		kSchedulable := true
		for _, pv := range parents {
			if pv.(*Activity).ActivityStatus != Completed {
				kSchedulable = false
				break
			}
		}
		if kSchedulable && v.(*Activity).ActivityStatus == Pending {
			activities = append(activities, k)
		}
	}
	return activities
}
