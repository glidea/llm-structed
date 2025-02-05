package llmstructed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
)

type llm interface {
	Completions(ctx context.Context, messages []string, responseSchema *schema) ([]byte, error)
}

type schemaType string

const (
	schemaTypeString  schemaType = "string"
	schemaTypeNumber  schemaType = "number"
	schemaTypeInteger schemaType = "integer"
	schemaTypeBoolean schemaType = "boolean"
	schemaTypeArray   schemaType = "array"
	schemaTypeObject  schemaType = "object"
)

type schema struct {
	Type             schemaType
	Description      string
	Enum             []string
	ArrayItems       *schema
	ObjectProperties map[string]*schema
}

type llmConfig struct {
	Debug                     bool
	BaseURL                   string
	APIKey                    string
	Model                     string
	Temperature               float32
	StructuredOutputSupported bool
}

type openai struct {
	config llmConfig
	hc     httpClient
}

func (o *openai) Completions(ctx context.Context, messages []string, responseSchema *schema) ([]byte, error) {
	baseURL := strings.TrimRight(o.config.BaseURL, "/")
	url := baseURL + "/chat/completions"

	// Build chat messages
	chatMessages := make([]map[string]string, 0, len(messages)+2)
	chatMessages = append(chatMessages, map[string]string{
		"role":    "system",
		"content": "You are a helpful assistant that provides structured output. Your response must be a valid JSON object.",
	})
	for _, msg := range messages {
		chatMessages = append(chatMessages, map[string]string{
			"role":    "user",
			"content": msg,
		})
	}

	// Build request body
	reqBody := map[string]interface{}{
		"model":       o.config.Model,
		"temperature": o.config.Temperature,
		"provider": map[string]interface{}{
			"require_parameters": true,
		},
	}
	if o.config.StructuredOutputSupported {
		reqBody["response_format"] = map[string]interface{}{
			"type": "json_schema",
			"json_schema": map[string]interface{}{
				"name":   "response",
				"strict": true,
				"schema": convertToOpenAISchema(responseSchema),
			},
		}
		reqBody["messages"] = chatMessages
	} else {
		reqBody["response_format"] = map[string]interface{}{
			"type": "json_object",
		}
		reqBody["messages"] = append(chatMessages, map[string]string{
			"role":    "user",
			"content": fmt.Sprintf("You must format your response as a JSON object following this schema: \n%v\nDo not include any other text in your response.", convertToOpenAISchema(responseSchema)),
		})
	}
	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "marshal request body")
	}

	// Build request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBodyBytes))
	if err != nil {
		return nil, errors.Wrap(err, "create request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.config.APIKey))

	if o.config.Debug {
		var curlCmd strings.Builder
		curlCmd.WriteString(fmt.Sprintf("curl -X POST %s \\\n", url))
		curlCmd.WriteString("  -H 'Content-Type: application/json' \\\n")
		curlCmd.WriteString(fmt.Sprintf("  -H 'Authorization: Bearer %s' \\\n", o.config.APIKey))
		curlCmd.WriteString(fmt.Sprintf("  -d '%s'", string(reqBodyBytes)))
		fmt.Println("Generated curl command:")
		fmt.Println(curlCmd.String())
	}

	// Send request
	resp, err := o.hc.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "send request")
	}
	defer resp.Body.Close()

	// Read response body
	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read response body")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBodyBytes))
	}

	if o.config.Debug {
		fmt.Println("Response:")
		fmt.Println(string(respBodyBytes))
	}

	// Parse response
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBodyBytes, &response); err != nil {
		return nil, errors.Wrap(err, "unmarshal response")
	}
	if len(response.Choices) == 0 {
		return nil, errors.New("no choices in response")
	}
	return []byte(response.Choices[0].Message.Content), nil
}

func convertToOpenAISchema(s *schema) map[string]interface{} {
	result := map[string]interface{}{
		"type": s.Type,
	}

	if s.Description != "" {
		result["description"] = s.Description
	}

	if len(s.Enum) > 0 {
		result["enum"] = s.Enum
	}

	if s.ArrayItems != nil {
		result["items"] = convertToOpenAISchema(s.ArrayItems)
	}

	if len(s.ObjectProperties) > 0 {
		properties := make(map[string]interface{})
		names := make([]string, 0, len(s.ObjectProperties))
		for k, v := range s.ObjectProperties {
			properties[k] = convertToOpenAISchema(v)
			names = append(names, k)
		}
		result["properties"] = properties
		result["required"] = names
		result["additionalProperties"] = false
	}

	return result
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type mockHTTPClient struct {
	mock.Mock
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if resp, ok := args.Get(0).(*http.Response); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

type mockLLM struct {
	responses [][]byte
	errors    []error
	calls     int
}

func (m *mockLLM) Completions(ctx context.Context, messages []string, responseSchema *schema) ([]byte, error) {
	if m.calls < len(m.responses) {
		resp := m.responses[m.calls]
		err := m.errors[m.calls]
		m.calls++
		return resp, err
	}
	return nil, errors.New("no more responses")
}
