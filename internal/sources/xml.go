package sources

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"

	"os"
)

type Dictionary struct {
	XMLName       xml.Name       `xml:"dictionary"`
	Notifications []Notification `xml:"notification"`
}

type Notification struct {
	ID       string `xml:"ID,attr"`
	Flag     bool   `xml:"flag,attr"`
	Severity string `xml:"severity,attr"`
	Text     string `xml:"text"`
}

func ReadEventsList(filePath string) ([]Notification, error) {
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

	return dictionary.Notifications, nil
}

func GenerateJSON(filePath string, notificationID string) (string, error) {
	notifications, err := ReadEventsList(filePath)
	if err != nil {
		return "", err
	}

	var selectedNotification *Notification
	for _, notification := range notifications {
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
