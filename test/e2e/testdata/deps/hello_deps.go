package main

import "github.com/gosimple/slug"

func Handler(req map[string]any) map[string]any {
	name, _ := req["name"].(string)
	if name == "" {
		name = "world"
	}
	return map[string]any{
		"message": "Hello, " + name + "!",
		"slug":    slug.Make(name),
	}
}
