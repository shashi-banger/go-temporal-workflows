package workflows

import (
	"log"
	"strings"
)

// A value expression is of the form `ingests.1.result.foo.boo.zoo`.
// This means that if A is map[string]interface{} representation of  `ingests.1`'s
// then the value of the expression is A["foo""]["boo"]["zoo]
// Another example is `ingests.1.result.foo.boo.zoo.1` which means that the value of the expression is A["foo""]["boo"]["zoo][1]
// Another example is `ingests.1.result.foo.boo.zoo.1.bar` which means that the value of the expression is A["foo""]["boo"]["zoo][1]["bar"]

func GetActivityNameFromValueExpression(ve string) string {
	ve = strings.TrimSpace(ve)
	ve = strings.TrimPrefix(ve, "{{")
	ve = strings.TrimSuffix(ve, "}}")
	ve = strings.TrimSpace(ve)
	s := strings.Split(ve, ".result.")
	return s[0]
}

func GetPathFromValueExpression(ve string) []string {
	ve = strings.TrimSpace(ve)
	ve = strings.TrimPrefix(ve, "{{")
	ve = strings.TrimSuffix(ve, "}}")
	ve = strings.TrimSpace(ve)
	s := strings.Split(ve, ".result.")
	return strings.Split(s[1], ".")
}

func GetValue(obj map[string]interface{}, ve string) interface{} {
	path := GetPathFromValueExpression(ve)
	for _, p := range path[:len(path)-1] {
		obj = obj[p].(map[string]interface{})
	}
	log.Println("GetValue:", path, path[len(path)-1], obj)
	return obj[path[len(path)-1]]
}
