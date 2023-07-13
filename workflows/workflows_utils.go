package workflows

import (
	"fmt"
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
			if pattern.MatchString(v.(string)) {
				//fmt.Println("appending")
				path = append(path, k)
				dst := make([]string, len(path))

				copy(dst, path)

				output = append(output, Match{dst, strings.TrimSpace(v.(string))})
				path = path[:len(path)-1]
			}
		} else if reflect.TypeOf(v).Kind() == reflect.Slice || reflect.TypeOf(v).Kind() == reflect.Array {
			path = append(path, k)
			//s := reflect.ValueOf(v)
			for i, item := range v.([]interface{}) {
				path[len(path)-1] = fmt.Sprintf("%s[%d]", k, i)
				if reflect.TypeOf(item).Kind() == reflect.Map {
					output = FindPathAndValuesWithPattern(pattern, reflect.ValueOf(item).Interface().(map[string]interface{}), path, output)
				} else if reflect.TypeOf(item).Kind() == reflect.String {
					if pattern.MatchString(reflect.ValueOf(item).Interface().(string)) {
						dst := make([]string, len(path))
						copy(dst, path)
						output = append(output, Match{dst, strings.TrimSpace(reflect.ValueOf(item).Interface().(string))})
					}
				}
			}
			path = path[:len(path)-1]
		}
	}
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
