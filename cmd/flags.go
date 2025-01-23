package cmd

import (
	"fmt"
	"strings"
)

// Labels is a custom flag type for parsing "key:value" pairs
type Labels map[string]string

// String is a string representation of the Labels map
func (l *Labels) String() string {
	pairs := []string{}
	for key, value := range *l {
		pairs = append(pairs, fmt.Sprintf("%s:%s", key, value))
	}
	return strings.Join(pairs, ",")
}

// Set parses the input string and sets the labels
func (l *Labels) Set(value string) error {
	if *l == nil {
		*l = make(map[string]string)
	}
	pairs := strings.Split(value, ",")
	for _, pair := range pairs {
		kv := strings.Split(pair, ":")
		if len(kv) != 2 {
			return fmt.Errorf("invalid label format: %s", pair)
		}
		(*l)[kv[0]] = kv[1]
	}
	return nil
}

// Type returns the type of the custom flag
func (l *Labels) Type() string {
	return "labels"
}

// Append appends new labels to the existing ones, overriding any existing keys
func (l *Labels) Append(newLabels Labels) {
	for key, value := range newLabels {
		(*l)[key] = value
	}
}
