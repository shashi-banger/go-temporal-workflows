package workflows

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"
)

type Match struct {
	pathArr []string
	value   string
}

func FindPathAndValuesWithPattern(pattern *regexp.Regexp, obj map[string]interface{}, path []string, output []Match) []Match {
	for k, v := range obj {

		if reflect.TypeOf(v).Kind() == reflect.Map {
			path = append(path, k)
			output = FindPathAndValuesWithPattern(pattern, v.(map[string]interface{}), path, output)
			path = path[:len(path)-1]
		} else if reflect.TypeOf(v).Kind() == reflect.String {
			fmt.Println(v, pattern)
			if pattern.MatchString(v.(string)) {
				fmt.Println("appending")
				output = append(output, Match{append(path, k), strings.TrimSpace(v.(string))})
			}
		}
	}
	log.Println("FindPathAndValuesWithPattern: ", len(output))
	return output
}

/*
func main() {
	c1 := map[string]interface{}{
		"key1": "value1",
		"key2": map[string]interface{}{
			"key3": "{{.ingests.1.response.url}}",
			"key4": "value4",
		},
		"key5": "{{.ingests.1.response.url}}",
	}

	output := FindPathAndValuesWithPattern(regexp.MustCompile("{{.*}}"), c1, []string{}, []Match{})
	fmt.Println(output)
}*/
