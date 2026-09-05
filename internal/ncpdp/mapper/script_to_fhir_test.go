package mapper

import (
	"testing"

	script "github.com/drfirst/go-oec/internal/ncpdp/script2023011"
)

func TestMapStatusToOperationOutcome(t *testing.T) {
	m := NewScriptToFHIRMapper()

	tests := []struct {
		name             string
		status           *script.Status
		expectedSeverity string
		expectedCode     string
	}{
		{
			name: "Success status",
			status: &script.Status{
				Code:        script.StatusCodeSuccess,
				Description: "Success",
			},
			expectedSeverity: "information",
			expectedCode:     "informational",
		},
		{
			name: "Accepted status",
			status: &script.Status{
				Code:        script.StatusCodeAccepted,
				Description: "Accepted",
			},
			expectedSeverity: "information",
			expectedCode:     "informational",
		},
		{
			name: "Validation Error status",
			status: &script.Status{
				Code:        script.StatusCodeValidationError,
				Description: "Validation Failed",
			},
			expectedSeverity: "error",
			expectedCode:     "processing",
		},
		{
			name: "Transmission Error status",
			status: &script.Status{
				Code:        script.StatusCodeTransmissionError,
				Description: "Transmission Failed",
			},
			expectedSeverity: "error",
			expectedCode:     "processing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.MapStatusToOperationOutcome(tt.status)

			if got.ResourceType != "OperationOutcome" {
				t.Errorf("ResourceType = %v, want OperationOutcome", got.ResourceType)
			}

			if len(got.Issue) != 1 {
				t.Fatalf("len(Issue) = %d, want 1", len(got.Issue))
			}

			issue := got.Issue[0]
			if issue.Severity != tt.expectedSeverity {
				t.Errorf("Severity = %v, want %v", issue.Severity, tt.expectedSeverity)
			}

			if issue.Code != tt.expectedCode {
				t.Errorf("Code = %v, want %v", issue.Code, tt.expectedCode)
			}

			if issue.Diagnostics != tt.status.Description {
				t.Errorf("Diagnostics = %v, want %v", issue.Diagnostics, tt.status.Description)
			}

			if issue.Details == nil || len(issue.Details.Coding) != 1 {
				t.Fatal("Expected 1 coding in details")
			}

			coding := issue.Details.Coding[0]
			if coding.System != "http://ncpdp.org/SCRIPT/StatusCode" {
				t.Errorf("Coding.System = %v, want http://ncpdp.org/SCRIPT/StatusCode", coding.System)
			}

			if coding.Code != tt.status.Code {
				t.Errorf("Coding.Code = %v, want %v", coding.Code, tt.status.Code)
			}

			if coding.Display != tt.status.Description {
				t.Errorf("Coding.Display = %v, want %v", coding.Display, tt.status.Description)
			}
		})
	}
}

func TestMapErrorToOperationOutcome(t *testing.T) {
	m := NewScriptToFHIRMapper()

	tests := []struct {
		name             string
		scriptErr        *script.Error
		expectedSeverity string
		expectedCode     string
	}{
		{
			name: "Record Not Found error",
			scriptErr: &script.Error{
				Code:        script.ErrorCodeRecordNotFound,
				Description: "Record not found",
			},
			expectedSeverity: "error",
			expectedCode:     "not-found",
		},
		{
			name: "System Error",
			scriptErr: &script.Error{
				Code:        script.ErrorCodeSystemError,
				Description: "Internal system error",
			},
			expectedSeverity: "error",
			expectedCode:     "exception",
		},
		{
			name: "Invalid Message error",
			scriptErr: &script.Error{
				Code:        script.ErrorCodeInvalidMessage,
				Description: "Invalid XML message",
			},
			expectedSeverity: "error",
			expectedCode:     "invalid",
		},
		{
			name: "Unknown error code",
			scriptErr: &script.Error{
				Code:        "999",
				Description: "Some unknown error",
			},
			expectedSeverity: "error",
			expectedCode:     "processing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.MapErrorToOperationOutcome(tt.scriptErr)

			if got.ResourceType != "OperationOutcome" {
				t.Errorf("ResourceType = %v, want OperationOutcome", got.ResourceType)
			}

			if len(got.Issue) != 1 {
				t.Fatalf("len(Issue) = %d, want 1", len(got.Issue))
			}

			issue := got.Issue[0]
			if issue.Severity != tt.expectedSeverity {
				t.Errorf("Severity = %v, want %v", issue.Severity, tt.expectedSeverity)
			}

			if issue.Code != tt.expectedCode {
				t.Errorf("Code = %v, want %v", issue.Code, tt.expectedCode)
			}

			if issue.Diagnostics != tt.scriptErr.Description {
				t.Errorf("Diagnostics = %v, want %v", issue.Diagnostics, tt.scriptErr.Description)
			}

			if issue.Details == nil || len(issue.Details.Coding) != 1 {
				t.Fatal("Expected 1 coding in details")
			}

			coding := issue.Details.Coding[0]
			if coding.System != "http://ncpdp.org/SCRIPT/ErrorCode" {
				t.Errorf("Coding.System = %v, want http://ncpdp.org/SCRIPT/ErrorCode", coding.System)
			}

			if coding.Code != tt.scriptErr.Code {
				t.Errorf("Coding.Code = %v, want %v", coding.Code, tt.scriptErr.Code)
			}

			if coding.Display != tt.scriptErr.Description {
				t.Errorf("Coding.Display = %v, want %v", coding.Display, tt.scriptErr.Description)
			}
		})
	}
}
