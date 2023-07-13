package workflows

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var jsonInp = `{
	"key2": {
		"key3": "{{.ingests.5.response.url}}",
		"key4": "value4"
	},
	"key1": "value1",
	"key5": "{{.ingests.1.response.url}}",
	"key6": ["foo", "bar"],
	"key7": [{
		"key3": "{{.ingests.7_5_0.response.url}}",
		"key4": "value7_4"
	}, {
		"key3": "{{.ingests.7_5.response.url}}",
		"key4": "value7_5"
	}]
}`

var expectedOut = map[string]string{
	"key1":         "value1",
	"key2.key3":    "{{.ingests.5.response.url}}",
	"key2.key4":    "value4",
	"key5":         "{{.ingests.1.response.url}}",
	"key7[0].key3": "{{.ingests.7_5_0.response.url}}",
	"key7[0].key4": "value7_4",
	"key7[1].key3": "{{.ingests.7_5.response.url}}",
	"key7[1].key4": "value7_5",
	"key6[0]":      "foo",
	"key6[1]":      "bar",
}

func Test1(t *testing.T) {
	c1 := map[string]interface{}{}
	json.Unmarshal([]byte(jsonInp), &c1)
	output := FindPathAndValuesWithPattern(regexp.MustCompile(".*"), c1, []string{}, []Match{})
	fmt.Println(output)
	fm := map[string]string{}
	for _, match := range output {
		k := strings.Join(match.pathArr, ".")
		v := match.value
		fm[k] = v
		fmt.Println(k, v)
	}
	assert.Equal(t, true, reflect.DeepEqual(fm, expectedOut))
}
