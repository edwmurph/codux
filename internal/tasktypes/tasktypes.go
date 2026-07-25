package tasktypes

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"strings"
	"unicode"

	"github.com/edwmurph/weft/internal/state"
	"github.com/edwmurph/weft/internal/titles"
)

const (
	DefaultClaudeID = "claude"
	DefaultCodexID  = "codex"
	DefaultShellID  = "shell"
	KindClaude      = "claude"
	KindCodex       = "codex"
	KindTerminal    = "terminal"
)

type InputMode string

const (
	InputModeCodex    InputMode = "codex"
	InputModeTerminal InputMode = "terminal"
)

type StartPolicy struct {
	Status         state.TaskStatus
	Visible        bool
	TrackOperation bool
}

type LoadingContext struct {
	Active        bool
	ScreenVisible bool
}

type Definition interface {
	Kind() string
	ConfiguredTypeID() string
	InputMode() InputMode
	StartPolicy() StartPolicy
	InitializeTask(task state.Task) state.Task
	Command(baseCommand string, task state.Task) string
	ScreenStatus(screenContent string) string
	ApplyPTYTitle(task state.Task, terminalTitle string, screenStatus string) state.Task
	Loading(task state.Task, ctx LoadingContext) bool
	TracksSessions() bool
	TracksTerminalCWD() bool
	TracksForegroundCommands() bool
	ShowsExitFooter() bool
	TopAlignedResize() bool
	RestartableTerminal() bool
}

var registry = map[string]Definition{
	KindClaude:   claudeDefinition{},
	KindCodex:    codexDefinition{},
	KindTerminal: terminalDefinition{},
}

func ForKind(kind string) (Definition, bool) {
	definition, ok := registry[strings.TrimSpace(kind)]
	return definition, ok
}

func StatusShowsLoadingIndicator(task state.Task) bool {
	switch titles.ConsolidatedStatus(task) {
	case string(state.StatusReady), string(state.StatusStopped), string(state.StatusKilled), string(state.StatusError), string(state.StatusSitting):
		return false
	default:
		return true
	}
}

type codexDefinition struct{}

func (codexDefinition) Kind() string {
	return KindCodex
}

func (codexDefinition) ConfiguredTypeID() string {
	return DefaultCodexID
}

func (codexDefinition) InputMode() InputMode {
	return InputModeCodex
}

func (codexDefinition) StartPolicy() StartPolicy {
	return StartPolicy{Status: state.StatusRunning, TrackOperation: true}
}

func (codexDefinition) InitializeTask(task state.Task) state.Task {
	return task
}

func (codexDefinition) Command(baseCommand string, task state.Task) string {
	if strings.TrimSpace(task.ResumeID) == "" {
		return strings.TrimSpace(baseCommand)
	}
	return CodexResumeCommand(baseCommand, task.ResumeID)
}

func (codexDefinition) ScreenStatus(screenContent string) string {
	return codexScreenStatus(screenContent)
}

func (codexDefinition) ApplyPTYTitle(task state.Task, terminalTitle string, screenStatus string) state.Task {
	hadLiveStatus := task.LiveStatus != ""
	if terminalTitle != "" {
		task.LiveTitle = titles.NormalizeLiveTitle(terminalTitle)
		task.Status = state.StatusRunning
	}
	titleStatus := titles.CodexActivityStatus(task.LiveTitle)
	titleIndicatesActivity := titles.LiveStatusIndicatesActivity(titleStatus)
	switch {
	case screenStatus != "":
		if titleIndicatesActivity {
			task.LiveStatus = titleStatus
		} else {
			task.LiveStatus = screenStatus
			task.Status = state.StatusReady
		}
	case titleStatus != "":
		task.LiveStatus = titleStatus
		if task.Status == state.StatusReady {
			task.Status = state.StatusRunning
		}
	case hadLiveStatus:
		task.LiveStatus = ""
		if task.Status == state.StatusReady {
			task.Status = state.StatusRunning
		}
	}
	return task
}

func (codexDefinition) Loading(task state.Task, ctx LoadingContext) bool {
	switch titles.ConsolidatedStatus(task) {
	case string(state.StatusError), string(state.StatusStopped), string(state.StatusKilled), string(state.StatusSitting):
		return false
	}
	if !ctx.Active {
		return StatusShowsLoadingIndicator(task)
	}
	if StatusShowsLoadingIndicator(task) {
		return true
	}
	return !ctx.ScreenVisible
}

func (codexDefinition) TracksSessions() bool {
	return true
}

func (codexDefinition) TracksTerminalCWD() bool {
	return false
}

func (codexDefinition) TracksForegroundCommands() bool {
	return false
}

func (codexDefinition) ShowsExitFooter() bool {
	return false
}

func (codexDefinition) TopAlignedResize() bool {
	return false
}

func (codexDefinition) RestartableTerminal() bool {
	return false
}

type claudeDefinition struct{}

func (claudeDefinition) Kind() string {
	return KindClaude
}

func (claudeDefinition) ConfiguredTypeID() string {
	return DefaultClaudeID
}

func (claudeDefinition) InputMode() InputMode {
	return InputModeCodex
}

func (claudeDefinition) StartPolicy() StartPolicy {
	return StartPolicy{Status: state.StatusRunning, TrackOperation: true}
}

func (claudeDefinition) InitializeTask(task state.Task) state.Task {
	if strings.TrimSpace(task.ResumeID) == "" {
		task.ResumeID = newClaudeSessionID(task)
	}
	return task
}

func (claudeDefinition) Command(baseCommand string, task state.Task) string {
	if strings.TrimSpace(task.ResumeID) == "" {
		return strings.TrimSpace(baseCommand)
	}
	if task.Status == state.StatusStarting || task.Status == state.StatusError {
		return ClaudeNewCommand(baseCommand, task.ResumeID)
	}
	return ClaudeResumeCommand(baseCommand, task.ResumeID)
}

func (claudeDefinition) ScreenStatus(screenContent string) string {
	return claudeScreenStatus(screenContent)
}

func (claudeDefinition) ApplyPTYTitle(task state.Task, terminalTitle string, screenStatus string) state.Task {
	if terminalTitle != "" {
		task.LiveTitle = titles.NormalizeLiveTitle(terminalTitle)
	}
	switch screenStatus {
	case "Ready":
		task.LiveStatus = screenStatus
		task.Status = state.StatusReady
	case "Working":
		task.LiveStatus = screenStatus
		task.Status = state.StatusRunning
	}
	return task
}

func (claudeDefinition) Loading(task state.Task, ctx LoadingContext) bool {
	switch titles.ConsolidatedStatus(task) {
	case string(state.StatusError), string(state.StatusStopped), string(state.StatusKilled), string(state.StatusSitting):
		return false
	}
	if !ctx.Active {
		return StatusShowsLoadingIndicator(task)
	}
	if StatusShowsLoadingIndicator(task) {
		return true
	}
	return !ctx.ScreenVisible
}

func (claudeDefinition) TracksSessions() bool {
	return true
}

func (claudeDefinition) TracksTerminalCWD() bool {
	return false
}

func (claudeDefinition) TracksForegroundCommands() bool {
	return false
}

func (claudeDefinition) ShowsExitFooter() bool {
	return false
}

func (claudeDefinition) TopAlignedResize() bool {
	return false
}

func (claudeDefinition) RestartableTerminal() bool {
	return false
}

type terminalDefinition struct{}

func (terminalDefinition) Kind() string {
	return KindTerminal
}

func (terminalDefinition) ConfiguredTypeID() string {
	return ""
}

func (terminalDefinition) InputMode() InputMode {
	return InputModeTerminal
}

func (terminalDefinition) StartPolicy() StartPolicy {
	return StartPolicy{Status: state.StatusReady, Visible: true}
}

func (terminalDefinition) InitializeTask(task state.Task) state.Task {
	return task
}

func (terminalDefinition) Command(baseCommand string, _ state.Task) string {
	return strings.TrimSpace(baseCommand)
}

func (terminalDefinition) ScreenStatus(string) string {
	return ""
}

func (terminalDefinition) ApplyPTYTitle(task state.Task, _ string, _ string) state.Task {
	return task
}

func (terminalDefinition) Loading(task state.Task, _ LoadingContext) bool {
	return StatusShowsLoadingIndicator(task)
}

func (terminalDefinition) TracksSessions() bool {
	return false
}

func (terminalDefinition) TracksTerminalCWD() bool {
	return true
}

func (terminalDefinition) TracksForegroundCommands() bool {
	return true
}

func (terminalDefinition) ShowsExitFooter() bool {
	return true
}

func (terminalDefinition) TopAlignedResize() bool {
	return true
}

func (terminalDefinition) RestartableTerminal() bool {
	return true
}

func CodexResumeCommand(codexCommand string, sessionID string) string {
	codexCommand = strings.TrimSpace(codexCommand)
	if codexCommand == "" {
		codexCommand = DefaultCodexID
	}
	return codexCommand + " resume " + shellQuote(sessionID)
}

func ClaudeNewCommand(claudeCommand string, sessionID string) string {
	claudeCommand = strings.TrimSpace(claudeCommand)
	if claudeCommand == "" {
		claudeCommand = DefaultClaudeID
	}
	return claudeCommand + " --session-id " + shellQuote(sessionID)
}

func ClaudeResumeCommand(claudeCommand string, sessionID string) string {
	claudeCommand = strings.TrimSpace(claudeCommand)
	if claudeCommand == "" {
		claudeCommand = DefaultClaudeID
	}
	return claudeCommand + " --resume " + shellQuote(sessionID)
}

func newClaudeSessionID(task state.Task) string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		sum := sha256.Sum256([]byte(strings.Join([]string{
			task.ID,
			task.WorkspaceID,
			task.GroupID,
			task.CreatedAt,
		}, "\x00")))
		copy(raw[:], sum[:16])
	}
	raw[6] = raw[6]&0x0f | 0x40
	raw[8] = raw[8]&0x3f | 0x80
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		raw[0:4],
		raw[4:6],
		raw[6:8],
		raw[8:10],
		raw[10:16],
	)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func claudeScreenStatus(content string) string {
	content = strings.ToLower(content)
	contentKey := screenStatusKey(content)
	for _, activeMarker := range []string{
		"esctointerrupt",
		"ctrl+ctointerrupt",
		"pressesctointerrupt",
	} {
		if strings.Contains(contentKey, activeMarker) {
			return "Working"
		}
	}
	for _, promptMarker := range []string{
		"?forshortcuts",
		"doyouwanttoproceed?",
		"wouldyouliketoproceed?",
		"doyoutrustthefilesinthisfolder?",
		"entertoconfirm",
		"esctocancel",
	} {
		if strings.Contains(contentKey, promptMarker) {
			return "Ready"
		}
	}
	lines := strings.Split(content, "\n")
	firstTailLine := max(0, len(lines)-6)
	for _, line := range lines[firstTailLine:] {
		if strings.HasPrefix(strings.TrimSpace(line), "❯") {
			return "Ready"
		}
	}
	if strings.TrimSpace(content) != "" {
		return "Working"
	}
	return ""
}

func codexScreenStatus(content string) string {
	content = strings.ToLower(content)
	contentKey := screenStatusKey(content)
	hasSubmitAction := strings.Contains(content, "to submit answer") ||
		strings.Contains(content, "to submit all")
	hasQuestionPrompt := strings.Contains(content, "question ") ||
		strings.Contains(content, "unanswered") ||
		strings.Contains(content, "user_note:")
	if hasSubmitAction && hasQuestionPrompt {
		return "Ready"
	}
	hasPermissionPrompt := strings.Contains(content, "allow codex to ") &&
		strings.Contains(content, "allow this request") &&
		strings.Contains(content, "deny") &&
		strings.Contains(content, "enter to submit")
	if hasPermissionPrompt {
		return "Ready"
	}
	hasCommandApprovalPrompt := strings.Contains(contentKey, "wouldyouliketorunthefollowingcommand?") &&
		strings.Contains(contentKey, "yes,proceed") &&
		strings.Contains(contentKey, "no,andtellcodex")
	if hasCommandApprovalPrompt {
		return "Ready"
	}
	return ""
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func screenStatusKey(content string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || strings.ContainsRune("╭╮╰╯─│", r) {
			return -1
		}
		return r
	}, content)
}
