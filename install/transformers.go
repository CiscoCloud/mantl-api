package install

import (
	"fmt"
	"strings"
)

type valueTransformer func(interface{}, packageDefinition) string

func zookeeperHosts(v interface{}, d packageDefinition) string {
	zkVal := "zookeeper.service.consul:2181"
	strval := v.(string)
	if zkHosts, ok := getConfigVal(d.apiConfig, "mantl", "zookeeper", "hosts").(string); ok {
		if strings.Contains(strval, zkVal) {
			strval = strings.Replace(strval, zkVal, zkHosts, -1)
		}
	}
	return strval
}

func intTransformer(v interface{}, d packageDefinition) string {
	if strval, ok := v.(string); ok { // already been converted
		return strval
	}
	return fmt.Sprintf("%d", int(v.(float64)))
}

func numberTransformer(v interface{}, d packageDefinition) string {
	if strval, ok := v.(string); ok { // already been converted
		return strval
	}
	return fmt.Sprintf("%0.2f", v.(float64))
}

var valueTransformers = map[string][]valueTransformer{
	"string": []valueTransformer{
		zookeeperHosts,
	},
	"integer": []valueTransformer{
		intTransformer,
	},
	"number": []valueTransformer{
		numberTransformer,
	},
}
