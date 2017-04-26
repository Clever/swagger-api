package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"gopkg.in/v1/yaml"
)

func main() {

	fullBytes, err := ioutil.ReadFile("./full.yml")
	if err != nil {
		log.Fatalf("Error reading full.yml: %s", err)
	}

	var swagger map[string]interface{}
	if err := yaml.Unmarshal(fullBytes, &swagger); err != nil {
		log.Fatalf("Error unmarshaling swagger yml: %s", err)
	}

	clientBytes, err := generateClientYml(swagger)
	if err != nil {
		log.Fatalf("Error generating client yml: %s", err)
	}
	fmt.Println(string(clientBytes))
}

func modifyDefinitionForV11(name string, def map[interface{}]interface{}) map[interface{}]interface{} {
	// TODO: implement me!!
	return def
}

func generateDataApiYml(i map[string]interface{}, version string) ([]byte, error) {
	m := deepCopyMap(i)

	info := m["info"].(map[interface{}]interface{})
	m["basePath"] = version

	paths := m["paths"].(map[interface{}]interface{})
	for path := range paths {
		if strings.Contains(path.(string), "/events") && !path.(string) == "/districts/{id}/status" {
			delete(paths, path)
			continue
		}
	}

	definitions := m["definitions"].(map[interface{}]interface{})
	for name, definition := range definitions {

		if strings.HasSuffix(name, ".created") ||
			strings.HasSuffix(name, ".updated") ||
			strings.HasSuffix(name, ".deleted") ||
			strings.HasSuffix(name, "Object") {
			delete(definitions, name)
			continue
		}

		if strings.HasPrefix(name, "Name") {
			delete(definitions, name)
			continue
		}

		if version == "/v1.1" {
			definitions[name] = modifyDefinitionForV11(name, definition)
		}
	}

	return yaml.Marshal(m)
}

func generateEventApiYml() error {
	// TODO: implement me!
	return nil
}

func generateClientYml(i map[string]interface{}) ([]byte, error) {
	m := deepCopyMap(i)

	delete(m, "x-sample-languages")
	info := m["info"].(map[interface{}]interface{})
	info["title"] = "Clever API"
	info["description"] = "The Clever API"
	m["basePath"] = "/v1.2"

	paths := m["paths"].(map[interface{}]interface{})
	for path, methodOp := range paths {

		if strings.HasPrefix(path.(string), "/districts/{id}/") && !path.(string) == "/districts/{id}/status" {
			delete(paths, path)
			continue
		}

		for _, o := range methodOp.(map[interface{}]interface{}) {
			// TODO: handle events in here...
			operation := o.(map[interface{}]interface{})
			operation["tags"] = []string{"Data"}

			params, ok := operation["parameters"].([]interface{})
			if !ok {
				continue
			}
			paramsToSet := make([]map[interface{}]interface{}, 0)

			// TODO: Add a nice comment!!
			for _, p := range params {
				param := p.(map[interface{}]interface{})
				include := true
				for key, value := range param {
					if key.(string) == "name" {
						if value.(string) == "show_links" || value.(string) == "include" ||
							value.(string) == "where" {
							include = false
						}
					}
				}
				if include {
					paramsToSet = append(paramsToSet, param)
				}
			}
			operation["parameters"] = paramsToSet
		}
	}

	// TODO: remove x-validated??? do we even need to???

	return yaml.Marshal(m)
}

// note that this doesn't handle arrays or other data types (which don't matter)
func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	ret := make(map[string]interface{})

	for k, v := range m {
		subMap, isMap := v.(map[string]interface{})
		if isMap {
			ret[k] = deepCopyMap(subMap)
		} else {
			ret[k] = v
		}
	}
	return ret
}
