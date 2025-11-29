// Package prompts contains embedded prompt templates for Air agents.
package prompts

import _ "embed"

// AgentContext is the system prompt for agents in single-repo mode.
//
//go:embed agent-context.md
var AgentContext string

// AgentContextWorkspace is the system prompt for agents in workspace (multi-repo) mode.
//
//go:embed agent-context-workspace.md
var AgentContextWorkspace string

// Orchestration is the system prompt for the planning/orchestration session in single-repo mode.
//
//go:embed orchestration.md
var Orchestration string

// OrchestrationWorkspace is the system prompt for planning in workspace (multi-repo) mode.
//
//go:embed orchestration-workspace.md
var OrchestrationWorkspace string

// Integration is the system prompt for the integration session.
//
//go:embed integration.md
var Integration string
