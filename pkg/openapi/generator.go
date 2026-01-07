package openapi

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type RouteDocs struct {
	Summary     string
	Description string
	Tags        []string
	RequestBody interface{} // Struct for request body schema
	Responses   map[int]ResponseDoc
}

type ResponseDoc struct {
	Description string
	Model       interface{} // Struct for response schema
	Example     interface{} // Example value
}

type Generator struct {
	engine    *gin.Engine
	info      Info
	servers   []Server
	tags      []Tag
	routeDocs map[string]RouteDocs
}

func NewGenerator(engine *gin.Engine, info Info, servers []Server, tags []Tag) *Generator {
	return &Generator{
		engine:    engine,
		info:      info,
		servers:   servers,
		tags:      tags,
		routeDocs: make(map[string]RouteDocs),
	}
}

// RegisterDocs registers documentation for a specific route
// method: GET, POST, etc.
// path: /api/v1/repos/:owner/:repo
func (g *Generator) RegisterDocs(method, path string, docs RouteDocs) {
	key := method + " " + path
	g.routeDocs[key] = docs
}

func (g *Generator) Generate() *OpenAPI {
	spec := &OpenAPI{
		OpenAPI: "3.0.3",
		Info:    g.info,
		Servers: g.servers,
		Tags:    g.tags,
		Paths:   make(map[string]*PathItem),
		Components: Components{
			Schemas: make(map[string]*Schema),
		},
	}

	for _, route := range g.engine.Routes() {
		// Convert Gin path to OpenAPI path
		// e.g., /api/repos/:owner/:repo -> /api/repos/{owner}/{repo}
		openAPIPath := convertPath(route.Path)

		if _, exists := spec.Paths[openAPIPath]; !exists {
			spec.Paths[openAPIPath] = &PathItem{}
		}
		pathItem := spec.Paths[openAPIPath]

		// Extract path parameters
		pathParams := extractPathParams(route.Path)

		// Get registered docs
		key := route.Method + " " + route.Path
		docs, hasDocs := g.routeDocs[key]

		operation := &Operation{
			Summary:     route.Handler,
			OperationID: getOperationID(route.Handler),
			Parameters:  pathParams,
			Responses:   make(map[string]Response),
		}

		if hasDocs {
			if docs.Summary != "" {
				operation.Summary = docs.Summary
			}
			if docs.Description != "" {
				operation.Description = docs.Description
			}
			if len(docs.Tags) > 0 {
				operation.Tags = docs.Tags
			}

			// Handle Request Body
			if docs.RequestBody != nil {
				schema := GenerateSchema(docs.RequestBody)
				operation.RequestBody = &RequestBody{
					Content: map[string]MediaType{
						"application/json": {
							Schema: schema,
						},
					},
					Required: true,
				}
			}

			// Handle Responses
			for status, respDoc := range docs.Responses {
				resp := Response{
					Description: respDoc.Description,
				}

				if respDoc.Model != nil || respDoc.Example != nil {
					mediaType := MediaType{}
					if respDoc.Model != nil {
						mediaType.Schema = GenerateSchema(respDoc.Model)
						if respDoc.Example != nil {
							mediaType.Schema.Example = respDoc.Example
						}
					}
					resp.Content = map[string]MediaType{
						"application/json": mediaType,
					}
				}

				operation.Responses[strconv.Itoa(status)] = resp
			}
		}

		// Default response if none provided
		if len(operation.Responses) == 0 {
			operation.Responses["200"] = Response{
				Description: "Successful response",
			}
		}

		// Assign operation to method
		switch route.Method {
		case "GET":
			pathItem.Get = operation
		case "POST":
			pathItem.Post = operation
		case "PUT":
			pathItem.Put = operation
		case "DELETE":
			pathItem.Delete = operation
		case "PATCH":
			pathItem.Patch = operation
		case "HEAD":
			pathItem.Head = operation
		case "OPTIONS":
			pathItem.Options = operation
		}
	}

	return spec
}

func convertPath(ginPath string) string {
	parts := strings.Split(ginPath, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + part[1:] + "}"
		} else if strings.HasPrefix(part, "*") {
			parts[i] = "{" + part[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

func extractPathParams(ginPath string) []Parameter {
	var params []Parameter
	parts := strings.Split(ginPath, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			params = append(params, Parameter{
				Name:     part[1:],
				In:       "path",
				Required: true,
				Schema: &Schema{
					Type: "string",
				},
			})
		} else if strings.HasPrefix(part, "*") {
			params = append(params, Parameter{
				Name:     part[1:],
				In:       "path",
				Required: true,
				Schema: &Schema{
					Type: "string",
				},
			})
		}
	}
	return params
}

func getOperationID(handlerName string) string {
	// handlerName is usually "github.com/bravo68web/stasis/internal/transport/http/handler.(*RepoHandler).GetRepository-fm"
	// We want something cleaner like "RepoHandler_GetRepository"

	parts := strings.Split(handlerName, "/")
	lastPart := parts[len(parts)-1]

	// Remove function pointer suffix if present
	if idx := strings.Index(lastPart, "-fm"); idx != -1 {
		lastPart = lastPart[:idx]
	}

	// Clean up parens and stars
	lastPart = strings.ReplaceAll(lastPart, "(", "")
	lastPart = strings.ReplaceAll(lastPart, ")", "")
	lastPart = strings.ReplaceAll(lastPart, "*", "")
	lastPart = strings.ReplaceAll(lastPart, ".", "_")

	return lastPart
}
