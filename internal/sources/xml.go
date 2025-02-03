package sources

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"os"
)

type Dictionary struct {
	XMLName xml.Name `xml:"dictionary"`
	Logs    []Log    `xml:"notification"`
	Metrics []Metric `xml:"metric"`
}

type Log struct {
	ID       string `xml:"ID,attr"`
	Flag     bool   `xml:"flag,attr"`
	Severity string `xml:"severity,attr"`
	Text     string `xml:"text"`
}

type Metric struct {
	Name               string `xml:"name,attr"`
	FullyQualifiedName string `xml:"fullyQualifiedName,attr"`
	Type               string `xml:"type,attr"`
	Description        string `xml:"description,attr"`
	Tags               string `xml:"tags,attr"`
}

func ReadDictionary(filePath string) (*Dictionary, error) {
	xmlFile, err := os.Open(filePath)

	if err != nil {
		return nil, fmt.Errorf("error opening XML file: %w", err)
	}
	defer xmlFile.Close()

	byteValue, err := io.ReadAll(xmlFile)
	if err != nil {
		return nil, fmt.Errorf("error reading XML file: %w", err)
	}

	var dictionary Dictionary
	err = xml.Unmarshal(byteValue, &dictionary)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling XML: %w", err)
	}

	return &dictionary, nil
}

func (m *Metric) GenerateMetricTemplate(values map[string]string, metricValue int) string {
	tags := strings.Split(m.Tags, ",")
	var tagStringBuilder strings.Builder
	for _, tag := range tags {
		keyValue := strings.Split(tag, "=")
		if len(keyValue) == 2 {
			tagKey := keyValue[0]
			tagStringBuilder.WriteString(fmt.Sprintf("%s=\"%s\",", tagKey, values[tagKey]))
		}
	}
	tagString := strings.TrimRight(tagStringBuilder.String(), ",")
	return fmt.Sprintf("echo \"%s{%s} %d\" >> /usr/share/nginx/html/metrics;", m.FullyQualifiedName, tagString, metricValue)
}

func GenerateJSON(filePath string, notificationID string) (string, error) {
	notifications, err := ReadDictionary(filePath)
	if err != nil {
		return "", err
	}

	var selectedNotification *Log
	for _, notification := range notifications.Logs {
		if notification.ID == notificationID {
			selectedNotification = &notification
			break
		}
	}

	if selectedNotification == nil {
		return "", fmt.Errorf("notification with ID %s not found", notificationID)
	}

	jsonData, err := json.MarshalIndent(selectedNotification, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error generating JSON: %w", err)
	}

	return string(jsonData), nil
}
