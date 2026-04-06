package policy

import "log/slog"

// PromptAction represents the user's decision on a content keyword prompt.
type PromptAction string

const (
	PromptAllowOnce   PromptAction = "allow_once"
	PromptAllowAlways PromptAction = "allow_always"
	PromptBlockOnce   PromptAction = "block_once"
	PromptBlockAlways PromptAction = "block_always"
)

// ContentPrompt is sent to the UI when a content keyword match requires user input.
type ContentPrompt struct {
	ID             string   `json:"id"`
	SessionID      string   `json:"session_id"`
	URL            string   `json:"url"`
	MatchedKeyword string   `json:"matched_keyword"`
	FilePaths      []string `json:"file_paths"`
}

// ContentPromptResponse is the user's decision from the UI.
type ContentPromptResponse struct {
	Action PromptAction `json:"action"`
}

// PromptResolver pauses a request and waits for user input.
type PromptResolver interface {
	PromptUser(prompt ContentPrompt) ContentPromptResponse
}

// HeadlessResolver blocks all content keyword matches without prompting.
type HeadlessResolver struct{}

func (h HeadlessResolver) PromptUser(prompt ContentPrompt) ContentPromptResponse {
	slog.Warn("content keyword match in headless mode, blocking",
		"keyword", prompt.MatchedKeyword,
		"files", prompt.FilePaths,
	)
	return ContentPromptResponse{Action: PromptBlockOnce}
}
