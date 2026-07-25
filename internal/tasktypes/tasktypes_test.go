package tasktypes

import (
	"regexp"
	"testing"

	"github.com/edwmurph/weft/internal/state"
)

func TestRegistryExposesBuiltInDefinitions(t *testing.T) {
	claude, ok := ForKind(KindClaude)
	if !ok {
		t.Fatal("claude definition missing")
	}
	if claude.ConfiguredTypeID() != DefaultClaudeID || claude.InputMode() != InputModeCodex {
		t.Fatalf("claude definition = %#v/%s", claude, claude.InputMode())
	}
	if policy := claude.StartPolicy(); policy.Status != state.StatusRunning || !policy.TrackOperation || policy.Visible {
		t.Fatalf("claude start policy = %#v", policy)
	}
	initialized := claude.InitializeTask(state.Task{ID: "claude-task", WorkspaceID: "workspace", CreatedAt: "2026-07-25T12:00:00Z", Status: state.StatusStarting})
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(initialized.ResumeID) {
		t.Fatalf("claude session id = %q", initialized.ResumeID)
	}
	if got := claude.Command("claude --model sonnet", initialized); got != "claude --model sonnet --session-id '"+initialized.ResumeID+"'" {
		t.Fatalf("claude new command = %q", got)
	}
	initialized.Status = state.StatusReady
	if got := claude.Command("claude --model sonnet", initialized); got != "claude --model sonnet --resume '"+initialized.ResumeID+"'" {
		t.Fatalf("claude resume command = %q", got)
	}
	if got := claude.ScreenStatus("Claude Code\n❯ \n  ? for shortcuts"); got != "Ready" {
		t.Fatalf("claude ready screen status = %q", got)
	}
	if got := claude.ScreenStatus("· Working…\n  Esc to interrupt"); got != "Working" {
		t.Fatalf("claude working screen status = %q", got)
	}

	codex, ok := ForKind(KindCodex)
	if !ok {
		t.Fatal("codex definition missing")
	}
	if codex.ConfiguredTypeID() != DefaultCodexID || codex.InputMode() != InputModeCodex {
		t.Fatalf("codex definition = %#v/%s", codex, codex.InputMode())
	}
	if policy := codex.StartPolicy(); policy.Status != state.StatusRunning || !policy.TrackOperation || policy.Visible {
		t.Fatalf("codex start policy = %#v", policy)
	}
	if got := codex.Command("", state.Task{ResumeID: "session-1"}); got != "codex resume 'session-1'" {
		t.Fatalf("codex resume command = %q", got)
	}
	if got := codex.ScreenStatus("Allow Codex to edit files?\nAllow this request\nDeny\nEnter to submit"); got != "Ready" {
		t.Fatalf("codex screen status = %q", got)
	}

	terminal, ok := ForKind(KindTerminal)
	if !ok {
		t.Fatal("terminal definition missing")
	}
	if terminal.ConfiguredTypeID() != "" || terminal.InputMode() != InputModeTerminal {
		t.Fatalf("terminal definition = %#v/%s", terminal, terminal.InputMode())
	}
	if policy := terminal.StartPolicy(); policy.Status != state.StatusReady || !policy.Visible || policy.TrackOperation {
		t.Fatalf("terminal start policy = %#v", policy)
	}
	if !terminal.TracksTerminalCWD() || !terminal.TracksForegroundCommands() || !terminal.RestartableTerminal() {
		t.Fatalf("terminal capabilities missing")
	}
}

func TestClaudeDefinitionTranslatesScreenIntoLiveStatus(t *testing.T) {
	claude, ok := ForKind(KindClaude)
	if !ok {
		t.Fatal("claude definition missing")
	}
	task := claude.ApplyPTYTitle(state.Task{ID: "a", Status: state.StatusRunning}, "Claude Code", "Ready")
	if task.LiveTitle != "Claude Code" || task.LiveStatus != "Ready" || task.Status != state.StatusReady {
		t.Fatalf("ready claude task = %#v", task)
	}
	task = claude.ApplyPTYTitle(task, "", "Working")
	if task.LiveStatus != "Working" || task.Status != state.StatusRunning {
		t.Fatalf("working claude task = %#v", task)
	}
}

func TestCodexLoadingUsesActiveScreenVisibility(t *testing.T) {
	codex, ok := ForKind(KindCodex)
	if !ok {
		t.Fatal("codex definition missing")
	}
	ready := codex.ApplyPTYTitle(state.Task{ID: "a", Status: state.StatusRunning}, "Fake Codex Ready", "")
	if codex.Loading(ready, LoadingContext{Active: false}) {
		t.Fatal("inactive ready codex task should not load")
	}
	if !codex.Loading(ready, LoadingContext{Active: true, ScreenVisible: false}) {
		t.Fatal("active ready codex task should load until visible content arrives")
	}
	if codex.Loading(ready, LoadingContext{Active: true, ScreenVisible: true}) {
		t.Fatal("active ready codex task with visible screen should not load")
	}
}

func TestCodexDefinitionTranslatesLiveTitleIntoLiveStatus(t *testing.T) {
	codex, ok := ForKind(KindCodex)
	if !ok {
		t.Fatal("codex definition missing")
	}
	task := codex.ApplyPTYTitle(state.Task{ID: "a", Status: state.StatusRunning}, "Fake Codex Crafting", "Ready")
	if task.LiveStatus != "Crafting" || task.Status != state.StatusRunning {
		t.Fatalf("busy codex status = %s/%q", task.Status, task.LiveStatus)
	}
	task = codex.ApplyPTYTitle(state.Task{ID: "a", Status: state.StatusRunning}, "Fake Codex Running", "Ready")
	if task.LiveStatus != "Ready" || task.Status != state.StatusReady {
		t.Fatalf("ready codex status = %s/%q", task.Status, task.LiveStatus)
	}
}
