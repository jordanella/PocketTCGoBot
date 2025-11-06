package accounts

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
)

// XMLStringEntry represents a <string> entry in Android SharedPreferences format
type XMLStringEntry struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

// XMLMap represents an Android SharedPreferences XML map
type XMLMap struct {
	Strings []XMLStringEntry `xml:"string"`
}

// XMLAccount represents an account stored in XML format (legacy format)
type XMLAccount struct {
	DeviceAccount  string `xml:"deviceAccount"`
	DevicePassword string `xml:"devicePassword"`
}

// AccountFile represents an XML account with its filename
type AccountFile struct {
	Filename       string
	DeviceAccount  string
	DevicePassword string
	FilePath       string
}

// LoadAccountsFromXML loads all XML account files from a directory
func LoadAccountsFromXML(directory string) ([]*AccountFile, error) {
	// Check if directory exists
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		// Create the directory if it doesn't exist
		if err := os.MkdirAll(directory, 0755); err != nil {
			return nil, fmt.Errorf("failed to create accounts directory: %w", err)
		}
		return []*AccountFile{}, nil
	}

	// Read all files in directory
	files, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read accounts directory: %w", err)
	}

	accounts := make([]*AccountFile, 0)

	// Parse each XML file
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Only process .xml files
		if filepath.Ext(file.Name()) != ".xml" {
			continue
		}

		filePath := filepath.Join(directory, file.Name())

		// Read XML file
		data, err := os.ReadFile(filePath)
		if err != nil {
			// Log error but continue processing other files
			fmt.Printf("Warning: Failed to read %s: %v\n", file.Name(), err)
			continue
		}

		// Try to parse as Android SharedPreferences format first (with <map>)
		var xmlMap XMLMap
		var deviceAccount, devicePassword string

		if err := xml.Unmarshal(data, &xmlMap); err == nil && len(xmlMap.Strings) > 0 {
			// Extract deviceAccount and devicePassword from map
			for _, entry := range xmlMap.Strings {
				switch entry.Name {
				case "deviceAccount":
					deviceAccount = entry.Value
				case "devicePassword":
					devicePassword = entry.Value
				}
			}
		} else {
			fmt.Printf("Warning: Missing required fields in %s\n", file.Name())
			continue
		}

		// Create AccountFile
		accountFile := &AccountFile{
			Filename:       file.Name(),
			DeviceAccount:  deviceAccount,
			DevicePassword: devicePassword,
			FilePath:       filePath,
		}

		accounts = append(accounts, accountFile)
	}

	return accounts, nil
}

// SaveAccountToXML saves an account to an XML file in Android SharedPreferences format
func SaveAccountToXML(directory, filename, deviceAccount, devicePassword string) error {
	// Ensure directory exists
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create XML map in Android SharedPreferences format
	xmlMap := XMLMap{
		Strings: []XMLStringEntry{
			{Name: "deviceAccount", Value: deviceAccount},
			{Name: "devicePassword", Value: devicePassword},
		},
	}

	// Marshal to XML
	data, err := xml.MarshalIndent(xmlMap, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %w", err)
	}

	// Add XML header with Android SharedPreferences style
	xmlData := []byte("<?xml version='1.0' encoding='utf-8' standalone='yes' ?>\n" + string(data))

	// Write to file
	filePath := filepath.Join(directory, filename)
	if err := os.WriteFile(filePath, xmlData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// DeleteAccountXML deletes an XML account file
func DeleteAccountXML(filePath string) error {
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}
