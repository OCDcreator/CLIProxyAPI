package executor

import (
	"context"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
	"github.com/tidwall/gjson"
)

// foreignEncryptedContentPayload mimics a client replaying an encrypted
// reasoning item from a non-xAI Responses upstream (e.g. an OpenAI
// Responses-compatible gateway behind CPA): opaque encrypted_content plus the
// item id, with store disabled.
const foreignEncryptedContentPayload = `{
	"model":"grok-4.5",
	"store":false,
	"input":[
		{"type":"reasoning","id":"rs_resp1:rs_item1","encrypted_content":"Q-PaDgGVPec3rnsjj1ECEPOpsuqSx3iQqCPwb_HE"},
		{"type":"message","role":"user","content":[{"type":"input_text","text":"Reply with exactly: PONG"}]}
	]
}`

// With xai.keep-foreign-encrypted-content enabled, encrypted reasoning content
// from non-xAI Responses dialects must pass through untouched so clients can
// replay it statelessly.
func TestXAIExecutorPrepareKeepsForeignEncryptedContentWhenConfigured(t *testing.T) {
	t.Parallel()

	exec := NewXAIExecutor(&config.Config{XAI: config.XAIConfig{KeepForeignEncryptedContent: true}})
	prepared, errPrepare := exec.prepareResponsesRequest(context.Background(), cliproxyexecutor.Request{
		Model:   "grok-4.5",
		Payload: []byte(foreignEncryptedContentPayload),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAIResponse,
	}, false)
	if errPrepare != nil {
		t.Fatalf("prepareResponsesRequest() error = %v", errPrepare)
	}

	item := gjson.GetBytes(prepared.body, "input.0")
	if item.Get("type").String() != "reasoning" {
		t.Fatalf("input.0 type = %q; body=%s", item.Get("type").String(), prepared.body)
	}
	if !item.Get("encrypted_content").Exists() {
		t.Fatalf("encrypted_content was stripped despite keep-foreign-encrypted-content; body=%s", prepared.body)
	}
	if got := item.Get("encrypted_content").String(); got != "Q-PaDgGVPec3rnsjj1ECEPOpsuqSx3iQqCPwb_HE" {
		t.Fatalf("encrypted_content = %q; body=%s", got, prepared.body)
	}
	if item.Get("id").String() != "rs_resp1:rs_item1" {
		t.Fatalf("reasoning id changed; body=%s", prepared.body)
	}
}

// By default the Grok-format sanitizer keeps running: encrypted_content that
// does not parse as a Grok payload is removed instead of forwarded.
func TestXAIExecutorPrepareSanitizesForeignEncryptedContentByDefault(t *testing.T) {
	t.Parallel()

	exec := NewXAIExecutor(&config.Config{})
	prepared, errPrepare := exec.prepareResponsesRequest(context.Background(), cliproxyexecutor.Request{
		Model:   "grok-4.5",
		Payload: []byte(foreignEncryptedContentPayload),
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FormatOpenAIResponse,
	}, false)
	if errPrepare != nil {
		t.Fatalf("prepareResponsesRequest() error = %v", errPrepare)
	}

	item := gjson.GetBytes(prepared.body, "input.0")
	if item.Get("encrypted_content").Exists() {
		t.Fatalf("encrypted_content should be stripped by default; body=%s", prepared.body)
	}
	if item.Get("type").String() != "reasoning" {
		t.Fatalf("input.0 type = %q; body=%s", item.Get("type").String(), prepared.body)
	}
}
