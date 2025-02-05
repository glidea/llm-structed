package llmstructed

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCompletions(t *testing.T) {
	tests := []struct {
		scenario     string
		given        string
		when         string
		then         string
		config       llmConfig
		messages     []string
		schema       *schema
		mockResponse string
		mockStatus   int
		mockHTTPErr  error
		expectErr    bool
		validateFunc func(t *testing.T, req *http.Request)
	}{
		{
			scenario: "Successful Completion",
			given:    "valid messages and schema",
			when:     "calling completions",
			then:     "should return valid response",
			config: llmConfig{
				APIKey:      "test-key",
				Temperature: 0.7,
			},
			messages: []string{"Hello"},
			schema: &schema{
				Type: schemaTypeString,
			},
			mockResponse: `{"choices":[{"message":{"content":"Hello back"}}]}`,
			mockStatus:   http.StatusOK,
			expectErr:    false,
			validateFunc: func(t *testing.T, req *http.Request) {
				assert.Equal(t, "Bearer test-key", req.Header.Get("Authorization"))
				body, err := io.ReadAll(req.Body)
				assert.NoError(t, err)
				assert.Contains(t, string(body), "Hello")
			},
		},
		{
			scenario: "API Error Response",
			given:    "valid request",
			when:     "API returns error",
			then:     "should return error",
			config: llmConfig{
				APIKey:      "test-key",
				Temperature: 0.7,
			},
			messages: []string{"Hello"},
			schema: &schema{
				Type: schemaTypeString,
			},
			mockResponse: `{"error": "invalid request"}`,
			mockStatus:   http.StatusBadRequest,
			expectErr:    true,
		},
		{
			scenario: "Schema Validation",
			given:    "strict schema validation enabled",
			when:     "calling completions",
			then:     "should include schema in request",
			config: llmConfig{
				APIKey:                    "test-key",
				Temperature:               0.7,
				StructuredOutputSupported: true,
			},
			messages: []string{"Hello"},
			schema: &schema{
				Type: schemaTypeObject,
				ObjectProperties: map[string]*schema{
					"message": {Type: schemaTypeString},
				},
			},
			mockResponse: `{"choices":[{"message":{"content":"Hello"}}]}`,
			mockStatus:   http.StatusOK,
			expectErr:    false,
			validateFunc: func(t *testing.T, req *http.Request) {
				body, err := io.ReadAll(req.Body)
				assert.NoError(t, err)
				assert.Contains(t, string(body), "json_schema")
				assert.Contains(t, string(body), "strict")
			},
		},
		{
			scenario: "HTTP Request Failure",
			given:    "network error occurs",
			when:     "calling completions",
			then:     "should return error",
			config: llmConfig{
				APIKey:      "test-key",
				Temperature: 0.7,
			},
			messages: []string{"Hello"},
			schema: &schema{
				Type: schemaTypeString,
			},
			mockHTTPErr: errors.New("network error"),
			expectErr:   true,
		},
		{
			scenario: "Complex Schema",
			given:    "complex nested schema",
			when:     "calling completions",
			then:     "should properly format schema in request",
			config: llmConfig{
				APIKey:                    "test-key",
				Temperature:               0.7,
				StructuredOutputSupported: true,
			},
			messages: []string{"Hello"},
			schema: &schema{
				Type: schemaTypeObject,
				ObjectProperties: map[string]*schema{
					"data": {
						Type: schemaTypeArray,
						ArrayItems: &schema{
							Type: schemaTypeObject,
							ObjectProperties: map[string]*schema{
								"id":   {Type: schemaTypeNumber},
								"name": {Type: schemaTypeString},
							},
						},
					},
				},
			},
			mockResponse: `{"choices":[{"message":{"content":"{\"data\":[{\"id\":1,\"name\":\"test\"}]}"}}]}`,
			mockStatus:   http.StatusOK,
			expectErr:    false,
			validateFunc: func(t *testing.T, req *http.Request) {
				body, err := io.ReadAll(req.Body)
				assert.NoError(t, err)
				assert.Contains(t, string(body), `"type":"array"`)
				assert.Contains(t, string(body), `"type":"object"`)
				assert.Contains(t, string(body), `"type":"number"`)
				assert.Contains(t, string(body), `"type":"string"`)
			},
		},
		{
			scenario: "Schema With Description",
			given:    "schema with field descriptions",
			when:     "calling completions",
			then:     "should include descriptions in request",
			config: llmConfig{
				APIKey:                    "test-key",
				Temperature:               0.7,
				StructuredOutputSupported: true,
			},
			messages: []string{"Hello"},
			schema: &schema{
				Type: schemaTypeObject,
				ObjectProperties: map[string]*schema{
					"name": {
						Type:        schemaTypeString,
						Description: "The user's name",
					},
					"age": {
						Type:        schemaTypeNumber,
						Description: "The user's age",
					},
				},
			},
			mockResponse: `{"choices":[{"message":{"content":"{\"name\":\"John\",\"age\":30}"}}]}`,
			mockStatus:   http.StatusOK,
			expectErr:    false,
			validateFunc: func(t *testing.T, req *http.Request) {
				body, err := io.ReadAll(req.Body)
				assert.NoError(t, err)
				assert.Contains(t, string(body), `"description":"The user's name"`)
				assert.Contains(t, string(body), `"description":"The user's age"`)
			},
		},
		{
			scenario: "Schema With Enum",
			given:    "schema with enum values",
			when:     "calling completions",
			then:     "should include enum values in request",
			config: llmConfig{
				APIKey:                    "test-key",
				Temperature:               0.7,
				StructuredOutputSupported: true,
			},
			messages: []string{"Hello"},
			schema: &schema{
				Type: schemaTypeObject,
				ObjectProperties: map[string]*schema{
					"status": {
						Type: schemaTypeString,
						Enum: []string{"pending", "active", "completed"},
					},
				},
			},
			mockResponse: `{"choices":[{"message":{"content":"{\"status\":\"active\"}"}}]}`,
			mockStatus:   http.StatusOK,
			expectErr:    false,
			validateFunc: func(t *testing.T, req *http.Request) {
				body, err := io.ReadAll(req.Body)
				assert.NoError(t, err)
				assert.Contains(t, string(body), `"enum":["pending","active","completed"]`)
			},
		},
		{
			scenario: "Context Cancellation",
			given:    "context is cancelled",
			when:     "calling completions",
			then:     "should return error",
			config: llmConfig{
				APIKey:      "test-key",
				Temperature: 0.7,
			},
			messages: []string{"Hello"},
			schema: &schema{
				Type: schemaTypeString,
			},
			mockHTTPErr: context.Canceled,
			expectErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.scenario, func(t *testing.T) {
			mockClient := &mockHTTPClient{}
			if tc.mockHTTPErr != nil {
				mockClient.On("Do", mock.Anything).Return(nil, tc.mockHTTPErr)
			} else {
				mockClient.On("Do", mock.Anything).Return(&http.Response{
					StatusCode: tc.mockStatus,
					Body:       io.NopCloser(strings.NewReader(tc.mockResponse)),
				}, nil)
			}

			llm := &openai{
				config: tc.config,
				hc:     mockClient,
			}

			resp, err := llm.Completions(context.Background(), tc.messages, tc.schema)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, resp)
			if tc.validateFunc != nil {
				calls := mockClient.Calls
				assert.Len(t, calls, 1)
				req := calls[0].Arguments[0].(*http.Request)
				tc.validateFunc(t, req)
			}
		})
	}
}
