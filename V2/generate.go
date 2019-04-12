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
		log.Fatalf("Error reading full-v1.yml: %s", err)
	}

	var swagger map[interface{}]interface{}
	if err := yaml.Unmarshal(fullBytes, &swagger); err != nil {
		log.Fatalf("Error unmarshaling swagger yml: %s", err)
	}

	clientBytes, err := generateClientYml(swagger)
	if err != nil {
		log.Fatalf("Error generating client yml: %s", err)
	}
	if err := ioutil.WriteFile("v2.1-client.yml", clientBytes, 0644); err != nil {
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

	dataV21, err := generateDataApiYml(swagger, "v2.1")
	if err != nil {
		log.Fatalf("Error generating data v2.1 API: %s", err)
	}
	if err := ioutil.WriteFile("v2.1.yml", dataV21, 0644); err != nil {
		log.Fatalf("Error writing data v2.1 API: %s", err)
	}

	eventsV21, err := generateEventsApiYml(swagger, "v2.1")
	if err != nil {
		log.Fatalf("Error generating events v2.1 API: %s", err)
	}
	if err := ioutil.WriteFile("v2.1-events.yml", eventsV21, 0644); err != nil {
		log.Fatalf("Error writing events v2.1 API: %s", err)
	}
}

// removeFieldProperty removes the specified value from a field's properties, if it exists.
func removeFieldProperty(properties map[interface{}]interface{}, fieldName, propertyName string) {
	fieldProperties := properties[fieldName].(map[interface{}]interface{})
	delete(fieldProperties, propertyName)
}

// removeEnum removes the specified enum from a field's properties, if it exists.
func removeEnum(properties map[interface{}]interface{}, fieldName string, enumBlacklist map[string]bool) {
	fieldProperties := properties[fieldName].(map[interface{}]interface{})
	oldEnums := fieldProperties["enum"].([]interface{})
	newEnums := []interface{}{}
	for _, enum := range oldEnums {
		if _, ok := enumBlacklist[enum.(string)]; !ok {
			newEnums = append(newEnums, enum)
		}
	}
	fieldProperties["enum"] = newEnums
}

func removeV21GradeEnums(properties map[interface{}]interface{}, fieldName string) {
	removeEnum(properties, fieldName, map[string]bool{
		"InfantToddler":            true,
		"Preschool":                true,
		"TransitionalKindergarten": true,
		"13":       true,
		"Ungraded": true,
		"":         true,
	})
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
		if version == "v2.0" {
			delete(properties, "enrollments")
			delete(properties, "ext")
			removeEnum(properties, "gender", map[string]bool{"X": true})
			removeEnum(properties, "home_language", map[string]bool{"": true})
			removeV21GradeEnums(properties, "grade")
		}
		if !isClient {
			delete(properties, "iep_status")
			delete(properties, "home_language")
			delete(properties, "unweighted_gpa")
			delete(properties, "weighted_gpa")
		}
	case "Contact":
		if version == "v2.0" {
			removeFieldProperty(properties, "type", "x-nullable")
		}
	case "Teacher":
		if version == "v2.0" {
			delete(properties, "ext")
		}
	case "Section":
		if version == "v2.0" {
			delete(properties, "ext")
			removeEnum(properties, "subject", map[string]bool{"": true})
			removeV21GradeEnums(properties, "grade")
			removeFieldProperty(properties, "subject", "x-nullable")
		}
	case "School":
		if version == "v2.0" {
			delete(properties, "ext")
			removeV21GradeEnums(properties, "high_grade")
			removeV21GradeEnums(properties, "low_grade")
		}
	case "District":
		if version == "v2.0" {
			delete(properties, "portal_url")
			delete(properties, "login_methods")
			delete(properties, "district_contact")
			delete(properties, "goals_enabled")
			removeEnum(properties, "state", map[string]bool{"": true})
			removeFieldProperty(properties, "state", "x-nullable")
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
	for path, methodOp := range paths {
		if strings.Contains(path.(string), "/events") && path.(string) != "/districts/{id}/status" {
			delete(paths, path)
			continue
		}

		for _, o := range methodOp.(map[interface{}]interface{}) {
			operation := o.(map[interface{}]interface{})
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
						if version == "v2.0" && value.(string) == "count" {
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

		if version == "v2.0" {
			if name == "SchoolEnrollment" {
				delete(definitions, nameInterface)
				continue
			}
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

		if version == "v2.0" {
			if name == "SchoolEnrollment" {
				delete(definitions, nameInterface)
				continue
			}
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
	m["basePath"] = "/v2.1"

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
		modifyDefinitions("v2.1", true, name.(string), definition.(map[interface{}]interface{}))
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
