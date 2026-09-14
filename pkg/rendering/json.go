package rendering

import (
	"encoding/json"
	"fmt"
)

func FormatAsJSON(output []map[string]any) error {

	outputBytes, err := json.Marshal(output)
	if err != nil {
		return err
	}

	fmt.Println(string(outputBytes))
	return nil
}
