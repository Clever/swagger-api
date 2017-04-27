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
	if err := ioutil.WriteFile("v1.2-client.yml", clientBytes, 0644); err != nil {
		log.Fatalf("Error writing data v1.1 API: %s", err)
	}

	dataV11, err := generateDataApiYml(swagger, "v1.1")
	if err != nil {
		log.Fatalf("Error generating data v1.1 API: %s", err)
	}
	if err := ioutil.WriteFile("v1.1.yml", dataV11, 0644); err != nil {
		log.Fatalf("Error writing data v1.1 API: %s", err)
	}

	dataV12, err := generateDataApiYml(swagger, "v1.2")
	if err != nil {
		log.Fatalf("Error generating data v1.2 API: %s", err)
	}
	if err := ioutil.WriteFile("v1.2.yml", dataV12, 0644); err != nil {
		log.Fatalf("Error writing data v1.2 API: %s", err)
	}

	eventsV11, err := generateEventsApiYml(swagger, "v1.1")
	if err != nil {
		log.Fatalf("Error generating events v1.1 API: %s", err)
	}
	if err := ioutil.WriteFile("v1.1-events.yml", eventsV11, 0644); err != nil {
		log.Fatalf("Error writing events v1.1 API: %s", err)
	}

	eventsV12, err := generateEventsApiYml(swagger, "v1.2")
	if err != nil {
		log.Fatalf("Error generating events v1.2 API: %s", err)
	}
	if err := ioutil.WriteFile("v1.2-events.yml", eventsV12, 0644); err != nil {
		log.Fatalf("Error writing events v1.2 API: %s", err)
	}

}

// modifyDefinitions removes fields that don't apply to the particular version / client
// combination. For example, it remove students.schools from v1.1.
func modifyDefinitions(version string, isClient bool, name string, def map[interface{}]interface{}) {

	properties, ok := def["properties"].(map[interface{}]interface{})
	if !ok {
		// Polymorphic sub-types, like students.updated, don't have their own properties
		return
	}

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

			// The state enum doesn't have "pause" in v1.1
			state := properties["state"].(map[interface{}]interface{})
			state["enum"] = []interface{}{"running", "pending", "error"}
		}
	default:
	}
}

// generateDataApiYml generates the data API from the base yml for a specific version. It does
// this by removing things from the yml, for example the /events endpoints.
func generateDataApiYml(i map[interface{}]interface{}, version string) ([]byte, error) {
	m := deepCopyMap(i)

	m["basePath"] = "/" + version
	info := m["info"].(map[interface{}]interface{})
	info["title"] = "Events API"
	info["description"] = "The Clever Events API"

	paths := m["paths"].(map[interface{}]interface{})
	for path := range paths {
		if strings.Contains(path.(string), "/events") && path.(string) != "/districts/{id}/status" {
			delete(paths, path)
			continue
		}
	}

	definitions := m["definitions"].(map[interface{}]interface{})
	// Remove any definitions that are used only for Events
	for nameInterface, definition := range definitions {

		name := nameInterface.(string)
		if strings.HasSuffix(name, ".created") ||
			strings.HasSuffix(name, ".updated") ||
			strings.HasSuffix(name, ".deleted") ||
			strings.HasSuffix(name, "Object") {
			delete(definitions, nameInterface)
			continue
		}

		if strings.HasPrefix(name, "Event") {
			delete(definitions, nameInterface)
			continue
		}

		modifyDefinitions(version, false, name, definition.(map[interface{}]interface{}))
	}

	return yaml.Marshal(m)
}

// generateEventsApiYml generates the events API from the base yml for a specific version. It does
// this by removing things from the yml, for example the non /events endpoints.
func generateEventsApiYml(i map[interface{}]interface{}, version string) ([]byte, error) {
	m := deepCopyMap(i)

	m["basePath"] = "/" + version

	paths := m["paths"].(map[interface{}]interface{})
	for path := range paths {
		if !strings.Contains(path.(string), "/events") {
			delete(paths, path)
			continue
		}
	}

	// The events API needs most of
	definitions := m["definitions"].(map[interface{}]interface{})
	for nameInterface, definition := range definitions {

		name := nameInterface.(string)
		if strings.HasPrefix(name, "DistrictAdmin") {
			delete(definitions, nameInterface)
			continue
		}
		if name == "GradeLevelsResponse" {
			delete(definitions, nameInterface)
			continue
		}

		modifyDefinitions(version, false, name, definition.(map[interface{}]interface{}))
	}

	return yaml.Marshal(m)
}

// generateClientYml generates the yml for the client libraries. It removes things we don't new
// implementations to use.
func generateClientYml(i map[interface{}]interface{}) ([]byte, error) {
	m := deepCopyMap(i)

	delete(m, "x-sample-languages")
	info := m["info"].(map[interface{}]interface{})
	info["title"] = "Clever API"
	info["description"] = "The Clever API"
	m["basePath"] = "/v1.2"

	paths := m["paths"].(map[interface{}]interface{})
	for path, methodOp := range paths {

		// The /districts/{id}/collection endpoints are redundant because you can just use
		// /collection so let's remove them
		if strings.HasPrefix(path.(string), "/districts/{id}/") &&
			path.(string) != "/districts/{id}/status" {
			delete(paths, path)
			continue
		}

		for _, o := range methodOp.(map[interface{}]interface{}) {
			operation := o.(map[interface{}]interface{})

			// Tweak the tags so they show up correctly in the client libraries
			if strings.Contains(path.(string), "/events") {
				operation["tags"] = []string{"Events"}
			} else {
				operation["tags"] = []string{"Data"}
			}

			// Remove the parameters we don't want in the client library
			params, ok := operation["parameters"].([]interface{})
			if !ok {
				continue
			}
			paramsForClient := make([]map[interface{}]interface{}, 0)
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
					paramsForClient = append(paramsForClient, param)
				}
			}
			operation["parameters"] = paramsForClient
		}
	}

	return yaml.Marshal(m)
}

// deepCopyMap recursively copies the map. We use this so we can modify it for each output yml
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
