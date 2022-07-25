package v3

import (
	"io/ioutil"
	"log"
	"strings"

	"github.com/Clever/i18n-go/languages"
	"github.com/Clever/swagger-api/sharedlib"
	"github.com/Clever/yaml"
)

// Major version and minor versions are defined here.
// May reuse this file for other/future API versions, but you may find it simpler
// to copy this file and modify it for new API versions.
const majorVersion = "3"

var minorVersions = []string{"0", "1"}
var versionStrs = []string{}

// Generate generates API source ymls for the major/minor versions.
func Generate() {
	for _, minorVersion := range minorVersions {
		versionStrs = append(versionStrs, "v"+majorVersion+"."+minorVersion)
	}

	definitionFileName := "full-v" + majorVersion + ".yml"
	fullBytes, err := ioutil.ReadFile("./" + definitionFileName)
	if err != nil {
		log.Fatalf("Error reading %s: %s", definitionFileName, err)
	}

	var swagger map[interface{}]interface{}
	if err := yaml.Unmarshal(fullBytes, &swagger); err != nil {
		log.Fatalf("Error unmarshaling swagger yml: %s", err)
	}

	for _, versionStr := range versionStrs {
		clientBytes, err := generateClientYml(swagger, versionStr)
		if err != nil {
			log.Fatalf("Error generating client yml: %s", err)
		}
		if err := ioutil.WriteFile(versionStr+"-client.yml", clientBytes, 0644); err != nil {
			log.Fatalf("Error writing client %s API: %s", versionStr, err)
		}

		versionData, err := generateDataAPIYml(swagger, versionStr)
		if err != nil {
			log.Fatalf("Error generating data %s API: %s", versionStr, err)
		}
		if err := ioutil.WriteFile(versionStr+".yml", versionData, 0644); err != nil {
			log.Fatalf("Error writing data %s API: %s", versionStr, err)
		}

		versionEvents, err := generateEventsAPIYml(swagger, versionStr)
		if err != nil {
			log.Fatalf("Error generating events %s API: %s", versionStr, err)
		}
		if err := ioutil.WriteFile(versionStr+"-events.yml", versionEvents, 0644); err != nil {
			log.Fatalf("Error writing events %s API: %s", versionStr, err)
		}
	}
}

// modifyDefinitions removes fields that don't apply to the particular version / client
// combination. For example, it removes students.schools from v1.1.
func modifyDefinitions(version string, isClient bool, name string, def map[interface{}]interface{}) {
	properties, ok := def["properties"].(map[interface{}]interface{})
	if !ok {
		// Polymorphic sub-types, like students.updated, don't have their own properties
		return
	}

	switch name {
	case "Student":
		if version == "v3.0" {
			delete(properties, "home_language_name")
			delete(properties, "home_language_code")
		}
		if !isClient {
			delete(properties, "iep_status")
			delete(properties, "home_language")
			delete(properties, "unweighted_gpa")
			delete(properties, "weighted_gpa")
		} else {
			if version > "v3.0" {
				delete(properties, "home_language")
				home_language_name := properties["home_language_name"].(map[interface{}]interface{})
				home_language_code := properties["home_language_code"].(map[interface{}]interface{})
				home_language_name["enum"] = languages.ISO6392Names
				home_language_code["enum"] = languages.ISO6392Names
			}
		}
	default:
	}
}

// generateDataAPIYml generates the data API from the base yml for a specific version. It does
// this by removing things from the yml, for example the /events endpoints.
func generateDataAPIYml(i map[interface{}]interface{}, version string) ([]byte, error) {
	m := sharedlib.DeepCopyMap(i)

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
				paramsForClient = append(paramsForClient, param)
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

		modifyDefinitions(version, false, name, definition.(map[interface{}]interface{}))
	}

	return yaml.Marshal(m)
}

// generateEventsAPIYml generates the events API from the base yml for a specific version. It does
// this by removing things from the yml, for example the non /events endpoints.
func generateEventsAPIYml(i map[interface{}]interface{}, version string) ([]byte, error) {
	m := sharedlib.DeepCopyMap(i)

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

	definitions := m["definitions"].(map[interface{}]interface{})
	for nameInterface, definition := range definitions {
		name := nameInterface.(string)
		modifyDefinitions(version, false, name, definition.(map[interface{}]interface{}))
	}

	return yaml.Marshal(m)
}

// generateClientYml generates the yml for the client libraries. It removes things we don't need
// implementations to use.
func generateClientYml(i map[interface{}]interface{}, versionStr string) ([]byte, error) {
	m := sharedlib.DeepCopyMap(i)

	delete(m, "x-sample-languages")
	info := m["info"].(map[interface{}]interface{})
	info["title"] = "Clever API"
	info["description"] = "The Clever API"
	m["basePath"] = "/" + versionStr

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
		modifyDefinitions(versionStr, true, name.(string), definition.(map[interface{}]interface{}))
	}

	return yaml.Marshal(m)
}
