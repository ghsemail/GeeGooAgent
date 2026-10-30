package procedural

import "os"

func readSkillFileOS(path string) ([]byte, error) {
	return os.ReadFile(path)
}
