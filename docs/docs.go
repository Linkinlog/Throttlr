// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/delete/{throttlrPath}": {
            "post": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Users will hit this endpoint to delete an existing endpoint",
                "consumes": [
                    "application/x-www-form-urlencoded"
                ],
                "produces": [
                    "text/plain",
                    "text/html"
                ],
                "tags": [
                    "Delete"
                ],
                "summary": "Delete endpoint",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Throttlr path",
                        "name": "throttlrPath",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Deleted",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/endpoints/{throttlrPath}": {
            "get": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Users will hit this endpoint to access the throttled endpoint",
                "consumes": [
                    "application/x-www-form-urlencoded",
                    "application/json"
                ],
                "produces": [
                    "text/plain",
                    "application/json",
                    "text/html"
                ],
                "tags": [
                    "Throttlr"
                ],
                "summary": "Throttle endpoint",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Throttlr path",
                        "name": "throttlrPath",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "429": {
                        "description": "Too many requests",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            },
            "post": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Users will hit this endpoint to access the throttled endpoint",
                "consumes": [
                    "application/x-www-form-urlencoded",
                    "application/json"
                ],
                "produces": [
                    "text/plain",
                    "application/json",
                    "text/html"
                ],
                "tags": [
                    "Throttlr"
                ],
                "summary": "Throttle endpoint",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Throttlr path",
                        "name": "throttlrPath",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "429": {
                        "description": "Too many requests",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/proxy/{throttlrPath}": {
            "get": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Users will hit this endpoint to access the proxied endpoint",
                "consumes": [
                    "application/x-www-form-urlencoded",
                    "application/json"
                ],
                "produces": [
                    "text/plain",
                    "application/json",
                    "text/html"
                ],
                "tags": [
                    "Proxy"
                ],
                "summary": "Proxy endpoint",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Throttlr path",
                        "name": "throttlrPath",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {}
            },
            "post": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Users will hit this endpoint to access the proxied endpoint",
                "consumes": [
                    "application/x-www-form-urlencoded",
                    "application/json"
                ],
                "produces": [
                    "text/plain",
                    "application/json",
                    "text/html"
                ],
                "tags": [
                    "Proxy"
                ],
                "summary": "Proxy endpoint",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Throttlr path",
                        "name": "throttlrPath",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {}
            }
        },
        "/register": {
            "post": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Users will hit this endpoint to register a new endpoint",
                "consumes": [
                    "application/x-www-form-urlencoded"
                ],
                "produces": [
                    "text/plain",
                    "text/html"
                ],
                "tags": [
                    "Register"
                ],
                "summary": "Register endpoint",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Endpoint to register",
                        "name": "endpoint",
                        "in": "formData",
                        "required": true
                    },
                    {
                        "enum": [
                            1,
                            2,
                            3,
                            4,
                            5
                        ],
                        "type": "integer",
                        "description": "Interval, 1 = minute, 2 = hour, 3 = day, 4 = week, 5 = month",
                        "name": "interval",
                        "in": "formData",
                        "required": true
                    },
                    {
                        "type": "integer",
                        "description": "Max requests per interval",
                        "name": "max",
                        "in": "formData",
                        "required": true
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/update/{throttlrPath}": {
            "post": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Users will hit this endpoint to update an existing endpoint",
                "consumes": [
                    "application/x-www-form-urlencoded"
                ],
                "produces": [
                    "text/plain",
                    "text/html"
                ],
                "tags": [
                    "Update"
                ],
                "summary": "Update endpoint",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Updated endpoint",
                        "name": "endpoint",
                        "in": "formData",
                        "required": true
                    },
                    {
                        "enum": [
                            1,
                            2,
                            3,
                            4,
                            5
                        ],
                        "type": "integer",
                        "description": "Interval, 1 = minute, 2 = hour, 3 = day, 4 = week, 5 = month",
                        "name": "interval",
                        "in": "formData",
                        "required": true
                    },
                    {
                        "type": "integer",
                        "description": "Max requests per interval",
                        "name": "max",
                        "in": "formData",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "Throttlr path",
                        "name": "throttlrPath",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        }
    },
    "securityDefinitions": {
        "ApiKeyAuth": {
            "type": "apiKey",
            "name": "key",
            "in": "query"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "0.0.1",
	Host:             "",
	BasePath:         "/v1",
	Schemes:          []string{},
	Title:            "Throttlr API",
	Description:      "This is the API for Throttlr, a rate limiting service.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
