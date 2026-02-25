// Package integration provides integration tests for the prescription engine.
package integration

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	fhir "github.com/drfirst/go-oec/internal/fhir/r5"
	"github.com/drfirst/go-oec/internal/ncpdp/mapper"
	"github.com/drfirst/go-oec/pkg/idempotency"
)

func TestFHIRToSCRIPTMapping(t *testing.T) {
	// Load fixture
	data, err := os.ReadFile("../fixtures/medication_request_lisinopril.json")
	if err != nil {
		t.Skipf("fixture not found: %v", err)
	}

	var medRequest fhir.MedicationRequest
	if err := json.Unmarshal(data, &medRequest); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// Create test patient
	patient := &fhir.Patient{
		ResourceType: "Patient",
		ID:           "test-patient-001",
		Name: []fhir.HumanName{
			{Use: "official", Family: "Doe", Given: []string{"John"}},
		},
		Gender:    "male",
		BirthDate: "1980-01-15",
		Identifier: []fhir.Identifier{
			{System: fhir.SystemMRN, Value: "MRN12345"},
		},
	}

	// Create test practitioner
	practitioner := &fhir.Practitioner{
		ResourceType: "Practitioner",
		ID:           "test-prescriber-001",
		Name: []fhir.HumanName{
			{Use: "official", Family: "Smith", Given: []string{"Jane"}},
		},
		Identifier: []fhir.Identifier{
			{System: fhir.SystemNPI, Value: "1234567890"},
			{System: fhir.SystemDEA, Value: "AS1234567"},
		},
	}

	// Create test pharmacy
	pharmacy := &fhir.Organization{
		ResourceType: "Organization",
		ID:           "test-pharmacy-001",
		Name:         "Test Pharmacy",
		Identifier: []fhir.Identifier{
			{System: fhir.SystemNCPDP, Value: "1234567"},
		},
	}

	// Map to SCRIPT
	m := mapper.NewFHIRToScriptMapper()
	result, err := m.MapToNewRx(&medRequest, patient, practitioner, pharmacy)
	if err != nil {
		t.Fatalf("mapping failed: %v", err)
	}

	// Verify result
	if result.MessageID == "" {
		t.Error("expected message ID")
	}
	if result.IdempotencyKey == "" {
		t.Error("expected idempotency key")
	}
	if result.PatientHash == "" {
		t.Error("expected patient hash")
	}
	if result.IsControlled {
		t.Error("lisinopril should not be controlled")
	}

	// Verify SCRIPT message
	xml, err := result.Message.ToXML()
	if err != nil {
		t.Fatalf("XML marshal failed: %v", err)
	}
	if len(xml) == 0 {
		t.Error("expected XML output")
	}

	t.Logf("Generated SCRIPT message: %d bytes", len(xml))
}

func TestControlledSubstanceMapping(t *testing.T) {
	data, err := os.ReadFile("../fixtures/medication_request_controlled.json")
	if err != nil {
		t.Skipf("fixture not found: %v", err)
	}

	var medRequest fhir.MedicationRequest
	if err := json.Unmarshal(data, &medRequest); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	patient := &fhir.Patient{
		ResourceType: "Patient",
		Name:         []fhir.HumanName{{Family: "Smith", Given: []string{"Jane"}}},
		Identifier:   []fhir.Identifier{{System: fhir.SystemMRN, Value: "MRN67890"}},
	}

	practitioner := &fhir.Practitioner{
		ResourceType: "Practitioner",
		Name:         []fhir.HumanName{{Family: "Smith", Given: []string{"Jane"}}},
		Identifier: []fhir.Identifier{
			{System: fhir.SystemNPI, Value: "1234567890"},
			{System: fhir.SystemDEA, Value: "AS1234567"},
		},
	}

	pharmacy := &fhir.Organization{
		ResourceType: "Organization",
		Name:         "Test Pharmacy",
		Identifier:   []fhir.Identifier{{System: fhir.SystemNCPDP, Value: "1234567"}},
	}

	m := mapper.NewFHIRToScriptMapper()
	result, err := m.MapToNewRx(&medRequest, patient, practitioner, pharmacy)
	if err != nil {
		t.Fatalf("mapping failed: %v", err)
	}

	if !result.IsControlled {
		t.Error("oxycodone should be controlled")
	}
	if result.DEASchedule == "" {
		t.Error("expected DEA schedule for controlled substance")
	}

	t.Logf("DEA Schedule: %s, Controlled: %v", result.DEASchedule, result.IsControlled)
}

func TestIdempotencyKeyGeneration(t *testing.T) {
	ts := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)

	key1 := generateTestKey("1234567890", "MRN001", "314076", ts)
	key2 := generateTestKey("1234567890", "MRN001", "314076", ts)
	key3 := generateTestKey("1234567890", "MRN001", "314076", ts.Add(time.Second*30))
	key4 := generateTestKey("9876543210", "MRN001", "314076", ts)

	if key1 != key2 {
		t.Error("same inputs should produce same key")
	}
	if key1 != key3 {
		t.Error("keys within same minute should match")
	}
	if key1 == key4 {
		t.Error("different prescriber should produce different key")
	}
}

func generateTestKey(npi, mrn, code string, ts time.Time) string {
	return idempotency.GenerateKey(npi, mrn, code, ts)
}
