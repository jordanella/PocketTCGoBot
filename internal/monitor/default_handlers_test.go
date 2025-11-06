package monitor

import (
	"fmt"
	"testing"
	"time"
)

func TestHandleCommunicationError(t *testing.T) {
	event := &ErrorEvent{
		Type:       ErrorCommunication,
		Severity:   SeverityCritical,
		Message:    "ADB connection lost",
		DetectedAt: time.Now(),
	}

	response := HandleCommunicationError(event)

	if response.Handled {
		t.Error("Communication errors should not be marked as handled")
	}

	if response.Action != ActionStop {
		t.Errorf("Expected ActionStop, got %v", response.Action)
	}

	if response.Error == nil {
		t.Error("Expected error to be set")
	}
}

func TestHandleMaintenanceError(t *testing.T) {
	event := &ErrorEvent{
		Type:       ErrorMaintenance,
		Severity:   SeverityHigh,
		Message:    "Game in maintenance",
		DetectedAt: time.Now(),
	}

	response := HandleMaintenanceError(event)

	if !response.Handled {
		t.Error("Maintenance errors should be marked as handled")
	}

	if response.Action != ActionAbort {
		t.Errorf("Expected ActionAbort, got %v", response.Action)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got %v", response.Error)
	}
}

func TestHandleUpdateRequiredError(t *testing.T) {
	event := &ErrorEvent{
		Type:       ErrorUpdate,
		Severity:   SeverityCritical,
		Message:    "Update required",
		DetectedAt: time.Now(),
	}

	response := HandleUpdateRequiredError(event)

	if response.Handled {
		t.Error("Update required errors should not be marked as handled")
	}

	if response.Action != ActionStop {
		t.Errorf("Expected ActionStop, got %v", response.Action)
	}

	if response.Error == nil {
		t.Error("Expected error to be set")
	}
}

func TestHandleBannedError(t *testing.T) {
	event := &ErrorEvent{
		Type:       ErrorBanned,
		Severity:   SeverityCritical,
		Message:    "Account banned",
		DetectedAt: time.Now(),
	}

	response := HandleBannedError(event)

	if response.Handled {
		t.Error("Banned errors should not be marked as handled")
	}

	if response.Action != ActionStop {
		t.Errorf("Expected ActionStop, got %v", response.Action)
	}

	if response.Error == nil {
		t.Error("Expected error to be set")
	}
}

func TestHandleTitleScreenError(t *testing.T) {
	event := &ErrorEvent{
		Type:       ErrorTitleScreen,
		Severity:   SeverityHigh,
		Message:    "Returned to title screen",
		DetectedAt: time.Now(),
	}

	response := HandleTitleScreenError(event)

	if !response.Handled {
		t.Error("Title screen errors should be marked as handled")
	}

	if response.Action != ActionAbort {
		t.Errorf("Expected ActionAbort, got %v", response.Action)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got %v", response.Error)
	}
}

func TestHandleNoResponseError(t *testing.T) {
	event := &ErrorEvent{
		Type:       ErrorNoResponse,
		Severity:   SeverityHigh,
		Message:    "Game not responding",
		DetectedAt: time.Now(),
	}

	response := HandleNoResponseError(event)

	if response.Handled {
		t.Error("No response errors should not be marked as handled")
	}

	if response.Action != ActionAbort {
		t.Errorf("Expected ActionAbort, got %v", response.Action)
	}

	if response.Error == nil {
		t.Error("Expected error to be set")
	}
}

func TestHandleTimeoutError(t *testing.T) {
	event := &ErrorEvent{
		Type:       ErrorTimeout,
		Severity:   SeverityHigh,
		Message:    "Action exceeded timeout",
		DetectedAt: time.Now(),
	}

	response := HandleTimeoutError(event)

	if !response.Handled {
		t.Error("Timeout errors should be marked as handled")
	}

	if response.Action != ActionAbort {
		t.Errorf("Expected ActionAbort, got %v", response.Action)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got %v", response.Error)
	}
}

func TestDefaultErrorHandler(t *testing.T) {
	tests := []struct {
		name           string
		errorType      ErrorType
		expectedAction ErrorAction
		shouldHandle   bool
	}{
		{
			name:           "Communication error",
			errorType:      ErrorCommunication,
			expectedAction: ActionStop,
			shouldHandle:   false,
		},
		{
			name:           "Maintenance error",
			errorType:      ErrorMaintenance,
			expectedAction: ActionAbort,
			shouldHandle:   true,
		},
		{
			name:           "Update required error",
			errorType:      ErrorUpdate,
			expectedAction: ActionStop,
			shouldHandle:   false,
		},
		{
			name:           "Banned error",
			errorType:      ErrorBanned,
			expectedAction: ActionStop,
			shouldHandle:   false,
		},
		{
			name:           "Title screen error",
			errorType:      ErrorTitleScreen,
			expectedAction: ActionAbort,
			shouldHandle:   true,
		},
		{
			name:           "No response error",
			errorType:      ErrorNoResponse,
			expectedAction: ActionAbort,
			shouldHandle:   false,
		},
		{
			name:           "Timeout error",
			errorType:      ErrorTimeout,
			expectedAction: ActionAbort,
			shouldHandle:   true,
		},
		{
			name:           "Popup error",
			errorType:      ErrorPopup,
			expectedAction: ActionContinue,
			shouldHandle:   false,
		},
		{
			name:           "Stuck error",
			errorType:      ErrorStuck,
			expectedAction: ActionAbort,
			shouldHandle:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &ErrorEvent{
				Type:       tt.errorType,
				Severity:   SeverityHigh,
				Message:    fmt.Sprintf("Test %s", tt.name),
				DetectedAt: time.Now(),
			}

			response := DefaultErrorHandler(event)

			if response.Handled != tt.shouldHandle {
				t.Errorf("Expected handled=%v, got %v", tt.shouldHandle, response.Handled)
			}

			if response.Action != tt.expectedAction {
				t.Errorf("Expected action=%v, got %v", tt.expectedAction, response.Action)
			}
		})
	}
}

func TestGetDefaultHandler(t *testing.T) {
	handler := GetDefaultHandler()
	if handler == nil {
		t.Error("GetDefaultHandler returned nil")
	}

	// Test that it works
	event := &ErrorEvent{
		Type:       ErrorMaintenance,
		Severity:   SeverityHigh,
		Message:    "Test",
		DetectedAt: time.Now(),
	}

	response := handler(event)
	if !response.Handled {
		t.Error("Handler should mark maintenance as handled")
	}
}

func TestGetHandlerForType(t *testing.T) {
	tests := []struct {
		errorType      ErrorType
		expectedAction ErrorAction
	}{
		{ErrorCommunication, ActionStop},
		{ErrorMaintenance, ActionAbort},
		{ErrorUpdate, ActionStop},
		{ErrorBanned, ActionStop},
		{ErrorTitleScreen, ActionAbort},
		{ErrorNoResponse, ActionAbort},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Handler for %v", tt.errorType), func(t *testing.T) {
			handler := GetHandlerForType(tt.errorType)
			if handler == nil {
				t.Error("GetHandlerForType returned nil")
			}

			event := &ErrorEvent{
				Type:       tt.errorType,
				Severity:   SeverityHigh,
				Message:    "Test",
				DetectedAt: time.Now(),
			}

			response := handler(event)
			if response.Action != tt.expectedAction {
				t.Errorf("Expected action=%v, got %v", tt.expectedAction, response.Action)
			}
		})
	}
}
