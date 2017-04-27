package main

import (
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

	var swagger map[interface{}]interface{}
	if err := yaml.Unmarshal(fullBytes, &swagger); err != nil {
		log.Fatalf("Error unmarshaling swagger yml: %s", err)
	}

	clientBytes, err := generateClientYml(swagger)
	if err != nil {
		log.Fatalf("Error generating client yml: %s", err)
	}
	if err := ioutil.WriteFile("temp-client-v1.2.yml", clientBytes, 0644); err != nil {
		log.Fatalf("Error writing data v1.1 API: %s", err)
	}

	dataV11, err := generateDataApiYml(swagger, "v1.1")
	if err != nil {
		log.Fatalf("Error generating data v1.1 API: %s", err)
	}
	if err := ioutil.WriteFile("temp-data-v1.1.yml", dataV11, 0644); err != nil {
		log.Fatalf("Error writing data v1.1 API: %s", err)
	}

	dataV12, err := generateDataApiYml(swagger, "v1.2")
	if err != nil {
		log.Fatalf("Error generating data v1.2 API: %s", err)
	}
	if err := ioutil.WriteFile("temp-data-v1.2.yml", dataV12, 0644); err != nil {
		log.Fatalf("Error writing data v1.2 API: %s", err)
	}

}

// modifyDefinitions removes fields that don't apply to the particular version / client
// combination. For example, it remove students.schools from v1.1.
func modifyDefinitions(version string, isClient bool, name string, def map[interface{}]interface{}) {

	properties := def["properties"].(map[interface{}]interface{})

	switch name {
	case "Student":
		if version == "v1.1" {
			delete(properties, "schools")
		}
		if !isClient {
			delete(properties, "iep_status")
		}
	case "Teacher":
		if version == "v1.1" {
			delete(properties, "schools")
		}
	case "DistrictStatus":
		if version == "v1.1" {
			delete(properties, "pause_start")
			delete(properties, "pause_end")
			delete(properties, "launch_date")
			// TODO: add in enum modifications
		}
	default:
	}
}

func generateDataApiYml(i map[interface{}]interface{}, version string) ([]byte, error) {
	m := deepCopyMap(i)

	m["basePath"] = "/" + version

	paths := m["paths"].(map[interface{}]interface{})
	for path := range paths {
		if strings.Contains(path.(string), "/events") && path.(string) != "/districts/{id}/status" {
			delete(paths, path)
			continue
		}
	}

	definitions := m["definitions"].(map[interface{}]interface{})
	for nameInterface, definition := range definitions {

		name := nameInterface.(string)
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

		modifyDefinitions(version, false, name, definition.(map[interface{}]interface{}))
	}

	return yaml.Marshal(m)
}

func generateEventApiYml() error {
	// TODO: implement me!
	return nil
}

func generateClientYml(i map[interface{}]interface{}) ([]byte, error) {
	m := deepCopyMap(i)

	delete(m, "x-sample-languages")
	info := m["info"].(map[interface{}]interface{})
	info["title"] = "Clever API"
	info["description"] = "The Clever API"
	m["basePath"] = "/v1.2"

	paths := m["paths"].(map[interface{}]interface{})
	for path, methodOp := range paths {

		if strings.HasPrefix(path.(string), "/districts/{id}/") && path.(string) != "/districts/{id}/status" {
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

	return yaml.Marshal(m)
}

// deepCopyMap makes a copy of all the maps in the
func deepCopyMap(m map[interface{}]interface{}) map[interface{}]interface{} {
	ret := make(map[interface{}]interface{})

	for k, v := range m {
		subMap, isMap := v.(map[interface{}]interface{})
		if isMap {
			ret[k] = deepCopyMap(subMap)
		} else {
			ret[k] = v
		}
	}
	return ret
}
