package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func checkSchedulerProcess() CheckResult {
	name := "agent scheduler"
	home := strings.TrimSpace(os.Getenv("GEEGOO_HOME"))
	if home == "" {
		home = filepath.Join(os.Getenv("HOME"), ".geegoo")
	}
	pidFile := filepath.Join(home, "geegoo-agent", "scheduler.pid")
	if data, err := os.ReadFile(pidFile); err == nil {
		pid := strings.TrimSpace(string(data))
		if pid != "" {
			if err := exec.Command("kill", "-0", pid).Run(); err == nil {
				return CheckResult{Name: name, OK: true, Detail: "running PID=" + pid}
			}
			return CheckResult{
				Name: name, OK: false,
				Detail: fmt.Sprintf("stale pid file %s (PID %s not running) — run: bash start.sh restart-scheduler", pidFile, pid),
			}
		}
	}
	out, err := exec.Command("pgrep", "-f", "geegoo.bin scheduler run").Output()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		pid := strings.Fields(string(out))[0]
		return CheckResult{Name: name, OK: true, Detail: "running PID=" + pid + " (no pid file)"}
	}
	return CheckResult{
		Name: name, OK: false,
		Detail: "not running — run: cd geegoo-agent && bash start.sh restart-scheduler",
	}
}
