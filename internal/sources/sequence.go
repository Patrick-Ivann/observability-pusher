package sources

import (
	"encoding/json"
	"os"
)

type SequenceNotification struct {
	ID         string   `json:"ID"`
	Values     []string `json:"values"`
	Repetition int      `json:"repetition"`
	Interval   int      `json:"interval"`
	Name       string   `json:"name,omitempty"`
	Labels     []string `json:"labels,omitempty"`
}

func ParseSequence(jsonFile string) ([]SequenceNotification, error) {
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, err
	}
	var sequences []SequenceNotification
	err = json.Unmarshal(data, &sequences)
	if err != nil {
		return nil, err
	}
	return sequences, nil
}
