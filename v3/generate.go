package v3

import (
	"errors"
	"io/ioutil"
	"log"
	"strings"

	"github.com/Clever/i18n-go/v2/languages"
	"github.com/Clever/swagger-api/sharedlib"
	"github.com/Clever/yaml"
)

// Major version and minor versions are defined here.
// May reuse this file for other/future API versions, but you may find it simpler
// to copy this file and modify it for new API versions.
const majorVersion = "3"

var minorVersions = []string{"0", "1"}
var versionStrs = []string{}

var lmsConnectModels = []string{
	"Unauthorized",
	"Assignment",
	"Submission",
	"AssigneeMode",
	"AssignmentState",
	"Attachment",
	"GradingScale",
	"GradingScaleEntry",
	"GradingType",
	"SubmissionType",
	"SubmissionFlag",
	"SubmissionState",
	"AssignmentRequest",
	"AttachmentRequest",
	"SubmissionRequest",
	"AssignmentResponse",
	"SubmissionResponse",
	"SubmissionsResponse",
	"SubmissionsLink",
}

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
		swaggerCopy := duplicateMap(swagger)

		if versionStr == "v3.0" {
			deleteV31Definitions(swaggerCopy)
			deleteLMSConnectObjects(swaggerCopy)
		}

		clientBytes, err := generateClientYml(swaggerCopy, versionStr)
		if err != nil {
			log.Fatalf("Error generating client yml: %s", err)
		}
		if err := ioutil.WriteFile(versionStr+"-client.yml", clientBytes, 0644); err != nil {
			log.Fatalf("Error writing client %s API: %s", versionStr, err)
		}

		versionData, err := generateDataAPIYml(swaggerCopy, versionStr)
		if err != nil {
			log.Fatalf("Error generating data %s API: %s", versionStr, err)
		}
		if err := ioutil.WriteFile(versionStr+".yml", versionData, 0644); err != nil {
			log.Fatalf("Error writing data %s API: %s", versionStr, err)
		}

		versionEvents, err := generateEventsAPIYml(swaggerCopy, versionStr)
		if err != nil {
			log.Fatalf("Error generating events %s API: %s", versionStr, err)
		}
		if err := ioutil.WriteFile(versionStr+"-events.yml", versionEvents, 0644); err != nil {
			log.Fatalf("Error writing events %s API: %s", versionStr, err)
		}

		if versionStr != "v3.0" {
			versionLMSConnect, err := generateLMSConnectAPIYml(swaggerCopy, versionStr)
			if err != nil {
				log.Fatalf("Error generating LMS Connect %s API: %s", versionStr, err)
			}
			if err := ioutil.WriteFile(versionStr+"-lms.yml", versionLMSConnect, 0644); err != nil {
				log.Fatalf("Error writing LMS Connect %s API: %s", versionStr, err)
			}
		}
	}
}

func isLMSConnectEndpoint(path interface{}) bool {
	return strings.Contains(path.(string), "/assignments") || strings.Contains(path.(string), "/submissions")
}

// duplicateMap creates a deep copy of the provided map
func duplicateMap(m map[interface{}]interface{}) map[interface{}]interface{} {
	cp := make(map[interface{}]interface{})
	for k, v := range m {
		if v == nil {
			cp[k] = nil
			continue
		}
		vm, ok := v.(map[interface{}]interface{})
		if ok {
			cp[k] = duplicateMap(vm)
		} else {
			cp[k] = v
		}
	}

	return cp
}

// deleteLMSConnectResponses deletes responses and definitions not used in v3.0 of our APIs
func deleteLMSConnectObjects(i map[interface{}]interface{}) error {
	responses, ok := i["responses"].(map[interface{}]interface{})
	if ok {
		delete(responses, "Unauthorized")
	} else {
		return errors.New("no responses found in provided map")
	}

	definitions, ok := i["definitions"].(map[interface{}]interface{})
	if ok {
		for _, modelName := range lmsConnectModels {
			delete(definitions, modelName)
		}
	} else {
		return errors.New("no definitions found in provided map")
	}
	return nil
}

// deleteV31Definitions deletes object definitions not used in v3.0 of our APIs
func deleteV31Definitions(i map[interface{}]interface{}) error {
	definitions, ok := i["definitions"].(map[interface{}]interface{})
	if ok {
		delete(definitions, "PreferredName")
		delete(definitions, "Disability")
	} else {
		return errors.New("no definitions found in provided map")
	}
	return nil
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
			delete(properties, "disability")
			delete(properties, "gifted_status")
			delete(properties, "home_language_code")
			delete(properties, "section_504_status")
			delete(properties, "preferred_name")

			// Frl_status should be included in v3.0 but filtering
			// it out here to separate out changes to v3.1 from
			// v3.0 as part of SHAPI-861
			delete(properties, "frl_status")
		}
		if !isClient {
			delete(properties, "unweighted_gpa")
			delete(properties, "weighted_gpa")

			// Home_language and iep_status are being filtered out of the
			// v3.0 events schema. Including it in versions 3.1 and above
			if version == "v3.0" {
				delete(properties, "iep_status")
				delete(properties, "home_language")
			}
		}
		if version > "v3.0" {
			// change home_language enum to v3.1 ISO-639-3 languages list and add enums for code
			home_language := properties["home_language"].(map[interface{}]interface{})
			home_language["enum"] = languages.ISO6393Names
			home_language_code := properties["home_language_code"].(map[interface{}]interface{})
			home_language_code["enum"] = languages.ISO6393Codes
		}
	case "District":
		if version == "v3.0" {
			delete(properties, "lms_state")
			delete(properties, "lms_type")
		}
	default:
	}
}

// generateDataAPIYml generates the data API from the base yml for a specific version. It does
// this by removing things from the yml, for example the /events and LMS Connect endpoints.
func generateDataAPIYml(i map[interface{}]interface{}, version string) ([]byte, error) {
	m := sharedlib.DeepCopyMap(i)

	m["basePath"] = "/" + version
	info := m["info"].(map[interface{}]interface{})
	info["version"] = strings.Replace(version, "v", "", -1) + ".0"

	paths := m["paths"].(map[interface{}]interface{})
	for path, methodOp := range paths {
		if strings.Contains(path.(string), "/events") && path.(string) != "/districts/{id}/status" ||
			isLMSConnectEndpoint(path) {
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

	deleteLMSConnectObjects(m)
	definitions := m["definitions"].(map[interface{}]interface{})
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

	deleteLMSConnectObjects(m)
	definitions := m["definitions"].(map[interface{}]interface{})
	for nameInterface, definition := range definitions {
		name := nameInterface.(string)
		modifyDefinitions(version, false, name, definition.(map[interface{}]interface{}))
	}

	return yaml.Marshal(m)
}

// generateLMSConnectAPIYml generates the LMS Connect API from the base yml for a specific version. It does
// this by removing things from the yml, for example the non-LMS Connect endpoints.
func generateLMSConnectAPIYml(i map[interface{}]interface{}, version string) ([]byte, error) {
	m := sharedlib.DeepCopyMap(i)

	m["basePath"] = "/" + version
	info := m["info"].(map[interface{}]interface{})
	info["title"] = "LMS Connect API"
	info["description"] = "The Clever LMS Connect API"
	info["version"] = strings.Replace(version, "v", "", -1) + ".0"

	paths := m["paths"].(map[interface{}]interface{})
	for path, methodOp := range paths {
		if !isLMSConnectEndpoint(path) {
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
	// Remove Data + Events API models from LMS file
	lmsConnectModelsPlusSharedModels := append(lmsConnectModels, []string{
		"BadRequest",
		"InternalError",
		"NotFound",
	}...)
	for nameInterface, _ := range definitions {
		name := nameInterface.(string)
		isLMSConnectModel := false
		for _, lmsConnectModelName := range lmsConnectModelsPlusSharedModels {
			if lmsConnectModelName == name {
				isLMSConnectModel = true
				continue
			}
		}

		if !isLMSConnectModel {
			delete(definitions, nameInterface)
		}
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
		// LMS Connect API is v3.1 and above
		if versionStr == "v3.0" && isLMSConnectEndpoint(path) {
			delete(paths, path)
			continue
		}

		for _, o := range methodOp.(map[interface{}]interface{}) {
			operation := o.(map[interface{}]interface{})

			// Tweak the tags so they show up correctly in the client libraries
			if strings.Contains(path.(string), "/events") {
				operation["tags"] = []string{"Events"}
			} else if isLMSConnectEndpoint(path) {
				operation["tags"] = []string{"LMS Connect"}
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
