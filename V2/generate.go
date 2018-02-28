package V2

import (
	"io/ioutil"
	"log"
	"strings"

	"gopkg.in/v1/yaml"
)

func Generate() {

	fullBytes, err := ioutil.ReadFile("./full-v2.yml")
	if err != nil {
		log.Fatalf("Error reading full-v2.yml: %s", err)
	}

	var swagger map[interface{}]interface{}
	if err := yaml.Unmarshal(fullBytes, &swagger); err != nil {
		log.Fatalf("Error unmarshaling swagger yml: %s", err)
	}

	clientBytes, err := generateClientYml(swagger)
	if err != nil {
		log.Fatalf("Error generating client yml: %s", err)
	}
	if err := ioutil.WriteFile("v2.0-client.yml", clientBytes, 0644); err != nil {
		log.Fatalf("Error writing client v2.0 API: %s", err)
	}

	dataV20, err := generateDataApiYml(swagger, "v2.0")
	if err != nil {
		log.Fatalf("Error generating data v2.0 API: %s", err)
	}
	if err := ioutil.WriteFile("v2.0.yml", dataV20, 0644); err != nil {
		log.Fatalf("Error writing data v2.0 API: %s", err)
	}

	eventsV20, err := generateEventsApiYml(swagger, "v2.0")
	if err != nil {
		log.Fatalf("Error generating events v2.0 API: %s", err)
	}
	if err := ioutil.WriteFile("v2.0-events.yml", eventsV20, 0644); err != nil {
		log.Fatalf("Error writing events v2.0 API: %s", err)
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
		if !isClient {
			delete(properties, "iep_status")
			delete(properties, "home_language")
			delete(properties, "unweighted_gpa")
			delete(properties, "weighted_gpa")
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
	info["version"] = strings.Replace(version, "v", "", -1) + ".0"

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
	info := m["info"].(map[interface{}]interface{})
	info["title"] = "Events API"
	info["description"] = "The Clever Events API"
	info["version"] = strings.Replace(version, "v", "", -1) + ".0"

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
	m["basePath"] = "/v2.0"

	paths := m["paths"].(map[interface{}]interface{})
	for path, methodOp := range paths {
		for _, o := range methodOp.(map[interface{}]interface{}) {
			operation := o.(map[interface{}]interface{})

			// Tweak the tags so they show up correctly in the client libraries
			if strings.Contains(path.(string), "/events") {
				operation["tags"] = []string{"Events"}
			} else {
				operation["tags"] = []string{"Data"}
			}
		}
	}

	definitions := m["definitions"].(map[interface{}]interface{})
	for name, definition := range definitions {
		modifyDefinitions("v2.0", true, name.(string), definition.(map[interface{}]interface{}))
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
