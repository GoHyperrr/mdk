package mdk

import "context"

// MCPResource represents a discoverable data resource exposed to AI agents.
type MCPResource struct {
	URI         string
	Name        string
	Description string
	MimeType    string
}

// ResourceProvider is implemented by modules that want to expose resources to MCP.
type ResourceProvider interface {
	ListResources(ctx context.Context) ([]MCPResource, error)
	ReadResource(ctx context.Context, uri string) (string, error)
}

// MCPPromptArgument represents an argument for a prompt template.
type MCPPromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// MCPPrompt represents a reusable prompt template.
type MCPPrompt struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Arguments   []MCPPromptArgument `json:"arguments,omitempty"`
}

// MCPPromptMessageContent represents the content inside a prompt message.
type MCPPromptMessageContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// MCPPromptMessage represents a prompt message history node.
type MCPPromptMessage struct {
	Role    string                  `json:"role"`
	Content MCPPromptMessageContent `json:"content"`
}

// GetPromptResult is returned by GetPrompt.
type GetPromptResult struct {
	Description string             `json:"description,omitempty"`
	Messages    []MCPPromptMessage `json:"messages"`
}

// PromptProvider is implemented by modules that want to expose custom prompt templates to LLMs.
type PromptProvider interface {
	ListPrompts(ctx context.Context) ([]MCPPrompt, error)
	GetPrompt(ctx context.Context, name string, arguments map[string]string) (*GetPromptResult, error)
}
