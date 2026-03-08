package main

// Handler receives the parsed JSON request body as a map and returns a map
// that will be serialized as the JSON response.
func Handler(req map[string]any) map[string]any {
	name, _ := req["name"].(string)
	if name == "" {
		name = "world"
	}
	return map[string]any{"message": "Hello, " + name + "!"}
}
