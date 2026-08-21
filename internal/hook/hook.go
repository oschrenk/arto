package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/oschrenk/arto/internal/config"
	"github.com/oschrenk/arto/internal/ntfy"
)

// HookInput represents the JSON input from Claude Code hooks.
type HookInput struct {
	HookEventName         string                 `json:"hook_event_name"`
	ToolName              string                 `json:"tool_name"`
	ToolInput             map[string]interface{}  `json:"tool_input"`
	PermissionSuggestions []string               `json:"permission_suggestions"`
}

// Question represents a single question in an AskUserQuestion request.
type Question struct {
	Question    string   `json:"question"`
	Header      string   `json:"header"`
	MultiSelect bool     `json:"multiSelect"`
	Options     []Option `json:"options"`
}

// Option represents a single option in a question.
type Option struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Decision represents the output JSON for Claude Code hooks.
type Decision struct {
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput"`
}

type HookSpecificOutput struct {
	HookEventName string         `json:"hookEventName"`
	DecisionBody  DecisionBody   `json:"decision"`
}

type DecisionBody struct {
	Behavior           string            `json:"behavior"`
	UpdatedPermissions []string          `json:"updatedPermissions,omitempty"`
	UpdatedInput       *UpdatedInput     `json:"updatedInput,omitempty"`
}

type UpdatedInput struct {
	Questions []Question        `json:"questions"`
	Answers   map[string]string `json:"answers"`
}

func askDecision() Decision {
	return Decision{
		HookSpecificOutput: HookSpecificOutput{
			HookEventName: "PermissionRequest",
			DecisionBody:  DecisionBody{Behavior: "ask"},
		},
	}
}

func denyDecision() Decision {
	return Decision{
		HookSpecificOutput: HookSpecificOutput{
			HookEventName: "PermissionRequest",
			DecisionBody:  DecisionBody{Behavior: "deny"},
		},
	}
}

func allowDecision(updatedPermissions []string, updatedInput *UpdatedInput) Decision {
	d := Decision{
		HookSpecificOutput: HookSpecificOutput{
			HookEventName: "PermissionRequest",
			DecisionBody:  DecisionBody{Behavior: "allow"},
		},
	}
	if len(updatedPermissions) > 0 {
		d.HookSpecificOutput.DecisionBody.UpdatedPermissions = updatedPermissions
	}
	if updatedInput != nil {
		d.HookSpecificOutput.DecisionBody.UpdatedInput = updatedInput
	}
	return d
}

func isAskUserQuestion(input *HookInput) bool {
	if input.ToolName != "AskUserQuestion" {
		return false
	}
	questions, ok := input.ToolInput["questions"]
	if !ok {
		return false
	}
	arr, ok := questions.([]interface{})
	return ok && len(arr) > 0
}

func buildActions(server, topic, requestID string, permissionSuggestions []string, auth *config.Auth) []ntfy.Action {
	url := fmt.Sprintf("%s/%s-response", server, topic)
	authHeaders := ntfy.BuildAuthHeader(auth)

	approveBody, _ := json.Marshal(map[string]interface{}{"requestId": requestID, "approved": true})
	denyBody, _ := json.Marshal(map[string]interface{}{"requestId": requestID, "approved": false})

	actions := []ntfy.Action{
		{ActionType: "http", Label: "Approve", URL: url, Body: string(approveBody), Method: "POST", Headers: authHeaders},
	}

	if len(permissionSuggestions) > 0 {
		alwaysBody, _ := json.Marshal(map[string]interface{}{"requestId": requestID, "approved": true, "alwaysAllow": true})
		actions = append(actions, ntfy.Action{
			ActionType: "http", Label: "Always Approve", URL: url, Body: string(alwaysBody), Method: "POST", Headers: authHeaders,
		})
	}

	actions = append(actions, ntfy.Action{
		ActionType: "http", Label: "Deny", URL: url, Body: string(denyBody), Method: "POST", Headers: authHeaders,
	})

	return actions
}

func buildQuestionActions(server, topic, requestID string, options []Option, auth *config.Auth) []ntfy.Action {
	url := fmt.Sprintf("%s/%s-response", server, topic)
	authHeaders := ntfy.BuildAuthHeader(auth)

	actions := make([]ntfy.Action, len(options))
	for i, opt := range options {
		body, _ := json.Marshal(map[string]interface{}{"requestId": requestID, "answer": opt.Label})
		actions[i] = ntfy.Action{
			ActionType: "http", Label: opt.Label, URL: url, Body: string(body), Method: "POST", Headers: authHeaders,
		}
	}
	return actions
}

func buildQuestionMessage(question string, options []Option, multiSelect bool, batchInfo string) string {
	msg := question
	if batchInfo != "" {
		msg += " " + batchInfo
	}
	if multiSelect {
		msg += "\n(multiple selections allowed)"
	}
	msg += "\n\n"
	for _, opt := range options {
		msg += fmt.Sprintf("• %s: %s\n", opt.Label, opt.Description)
	}
	return msg[:len(msg)-1] // trimEnd
}

func parseQuestions(input *HookInput) []Question {
	raw, ok := input.ToolInput["questions"]
	if !ok {
		return nil
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}

	var questions []Question
	if err := json.Unmarshal(data, &questions); err != nil {
		return nil
	}
	return questions
}

func processAskUserQuestion(input *HookInput, cfg *config.Config, auth *config.Auth) Decision {
	questions := parseQuestions(input)
	if len(questions) == 0 {
		return askDecision()
	}

	server := cfg.NtfyServer
	topic := cfg.Topic
	answers := make(map[string]string)

	for _, q := range questions {
		requestID := config.GenerateTopic() // reuse random generation for unique IDs

		const maxButtons = 3
		var batches [][]Option
		for j := 0; j < len(q.Options); j += maxButtons {
			end := j + maxButtons
			if end > len(q.Options) {
				end = len(q.Options)
			}
			batches = append(batches, q.Options[j:end])
		}

		for i, batch := range batches {
			var batchInfo string
			if len(batches) > 1 {
				batchInfo = fmt.Sprintf("(%d/%d)", i+1, len(batches))
			}
			actions := buildQuestionActions(server, topic, requestID, batch, auth)
			message := buildQuestionMessage(q.Question, batch, q.MultiSelect, batchInfo)

			header := q.Header
			if header == "" {
				header = "Question"
			}

			params := ntfy.NotificationParams{
				Server:    server,
				Topic:     topic,
				Title:     "Claude Code: " + header,
				Message:   message,
				Actions:   actions,
				RequestID: requestID,
				Auth:      auth,
			}

			if err := ntfy.SendWithRetry(params); err != nil {
				return askDecision()
			}
		}

		timeout := time.Duration(cfg.Timeout) * time.Second
		params := ntfy.NotificationParams{
			Server: server,
			Topic:  topic,
			Auth:   auth,
		}
		response := ntfy.WaitForResponse(context.Background(), params, requestID, timeout)

		if response.Answer != "" {
			answers[q.Question] = response.Answer
		} else {
			fmt.Fprintln(os.Stderr, "[arto] No answer received. Falling back to CLI.")
			return askDecision()
		}
	}

	return allowDecision(nil, &UpdatedInput{
		Questions: questions,
		Answers:   answers,
	})
}

// ProcessHook processes a Claude Code hook request and returns the decision JSON.
func ProcessHook(inputJSON []byte) ([]byte, error) {
	var input HookInput
	if err := json.Unmarshal(inputJSON, &input); err != nil {
		result, _ := json.Marshal(askDecision())
		return result, nil
	}

	cfg, err := config.Load()
	if err != nil || cfg.Topic == "" {
		result, _ := json.Marshal(askDecision())
		return result, nil
	}

	auth := config.ResolveAuth(cfg)

	if isAskUserQuestion(&input) {
		decision := processAskUserQuestion(&input, cfg, auth)
		result, _ := json.Marshal(decision)
		return result, nil
	}

	requestID := config.GenerateTopic()
	info := ntfy.FormatToolInfo(input.ToolName, input.ToolInput)
	actions := buildActions(cfg.NtfyServer, cfg.Topic, requestID, input.PermissionSuggestions, auth)

	params := ntfy.NotificationParams{
		Server:    cfg.NtfyServer,
		Topic:     cfg.Topic,
		Title:     info.Title,
		Message:   info.Message,
		Actions:   actions,
		RequestID: requestID,
		Auth:      auth,
	}

	if err := ntfy.SendWithRetry(params); err != nil {
		result, _ := json.Marshal(askDecision())
		return result, nil
	}

	isPlanReview := input.ToolName == "ExitPlanMode"
	timeout := time.Duration(cfg.Timeout) * time.Second
	if isPlanReview {
		timeout = time.Duration(cfg.PlanTimeout) * time.Second
	}

	response := ntfy.WaitForResponse(context.Background(), params, requestID, timeout)

	if response.Timeout {
		fmt.Fprintln(os.Stderr, "[arto] Timed out waiting for response. Falling back to CLI.")
		result, _ := json.Marshal(askDecision())
		return result, nil
	}
	if response.Err != nil {
		fmt.Fprintf(os.Stderr, "[arto] Response error: %v — Falling back to CLI.\n", response.Err)
		result, _ := json.Marshal(askDecision())
		return result, nil
	}
	if !response.Approved {
		result, _ := json.Marshal(denyDecision())
		return result, nil
	}

	var updatedPerms []string
	if response.AlwaysAllow && len(input.PermissionSuggestions) > 0 {
		updatedPerms = input.PermissionSuggestions
	}

	decision := allowDecision(updatedPerms, nil)
	result, _ := json.Marshal(decision)
	return result, nil
}
