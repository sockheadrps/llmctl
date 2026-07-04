package runtime

import (
	"encoding/json"
	"os"

	"github.com/sockheadrps/llmctl/internal/models"
)

// loadState reads the list of tracked running instances from path.
// A missing file is treated as an empty list, not an error.
func loadState(path string) ([]models.Running, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var running []models.Running
	if err := json.Unmarshal(data, &running); err != nil {
		return nil, err
	}
	return running, nil
}

// saveState writes the list of tracked running instances to path.
func saveState(path string, running []models.Running) error {
	data, err := json.MarshalIndent(running, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
