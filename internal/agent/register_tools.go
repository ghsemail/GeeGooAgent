package agent

import "github.com/ghsemail/GeeGooAgent/internal/tools"

func init() {
	tools.AddRegistrar(registerAgentTools)
}

func registerAgentTools(r *tools.Registry, deps tools.Deps) {
	if deps.Delegate != nil {
		RegisterDelegateTask(r, deps.Delegate)
		RegisterDelegateTasks(r, deps.Delegate)
	}
}
