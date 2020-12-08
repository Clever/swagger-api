package sharedlib

// RemoveFieldProperty removes the specified value from a field's properties, if it exists.
func RemoveFieldProperty(properties map[interface{}]interface{}, fieldName, propertyName string) {
	fieldProperties := properties[fieldName].(map[interface{}]interface{})
	delete(fieldProperties, propertyName)
}

// RemoveEnum removes the specified enum from a field's properties, if it exists.
func RemoveEnum(properties map[interface{}]interface{}, fieldName string, enumBlacklist map[string]bool) {
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

// DeepCopyMap recursively copies the map. We use this so we can modify it for each output yml
func DeepCopyMap(m map[interface{}]interface{}) map[interface{}]interface{} {
	ret := make(map[interface{}]interface{})

	for k, v := range m {
		subMap, isMap := v.(map[interface{}]interface{})
		if isMap {
			ret[k] = DeepCopyMap(subMap)
		} else {
			ret[k] = v
		}
	}
	return ret
}
