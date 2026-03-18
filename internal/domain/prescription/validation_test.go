package prescription

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAggregate_Validate(t *testing.T) {
	// Setup the JSON hook
	originalHook := jsonUnmarshal
	defer func() { jsonUnmarshal = originalHook }()
	jsonUnmarshal = func(data []byte, v interface{}) error {
		return json.Unmarshal(data, v)
	}

	validData := PrescriptionCreatedData{
		PrescriptionID:  "rx-123",
		PatientHash:     "hash123",
		PrescriberNPI:   "1234567890",
		MedicationNDC:   "12345-678-90",
		Quantity:        30,
		DaysSupply:      30,
		SigText:         "Take 1 daily",
		IsControlled:    false,
	}

	t.Run("successful validation", func(t *testing.T) {
		a := NewAggregate("rx-123")
		err := a.Create(&validData)
		if err != nil {
			t.Fatalf("failed to create aggregate: %v", err)
		}

		err = a.Validate()
		if err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}

		if a.Status() != StatusValidated {
			t.Errorf("expected status %v, got %v", StatusValidated, a.Status())
		}
	})

	t.Run("fails if not pending", func(t *testing.T) {
		a := NewAggregate("rx-123")
		err := a.Validate()
		if err == nil {
			t.Error("expected error for non-pending aggregate")
		}
		if !strings.Contains(err.Error(), "must be in pending status") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		a := NewAggregate("rx-123")

		invalidData := validData
		invalidData.PatientHash = ""
		invalidData.MedicationNDC = ""
		invalidData.Quantity = 0
		invalidData.DaysSupply = 0
		invalidData.SigText = ""

		a.Create(&invalidData)

		err := a.Validate()
		if err == nil {
			t.Fatal("expected validation error")
		}

		errs, ok := err.(ValidationErrors)
		if !ok {
			t.Fatalf("expected ValidationErrors type, got %T", err)
		}

		if len(errs) != 5 {
			t.Errorf("expected 5 errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("prescriber ID rules", func(t *testing.T) {
		a := NewAggregate("rx-123")

		invalidData := validData
		invalidData.PrescriberNPI = ""
		invalidData.PrescriberDEA = ""

		a.Create(&invalidData)

		err := a.Validate()
		if err == nil {
			t.Fatal("expected validation error")
		}

		errs, ok := err.(ValidationErrors)
		if !ok {
			t.Fatalf("expected ValidationErrors type, got %T", err)
		}

		found := false
		for _, e := range errs {
			if e.Field == "PrescriberID" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected PrescriberID error, got: %v", errs)
		}
	})

	t.Run("controlled substance rules", func(t *testing.T) {
		a := NewAggregate("rx-123")

		invalidData := validData
		invalidData.IsControlled = true
		invalidData.DEASchedule = "2"
		invalidData.RefillsAllowed = 1 // Not allowed for Sch 2
		invalidData.DaysSupply = 100   // Max 90 for Sch 2
		invalidData.PrescriberNPI = "1234567890"
		invalidData.PrescriberDEA = "AB1234567"

		a.Create(&invalidData)

		err := a.Validate()
		if err == nil {
			t.Fatal("expected validation error")
		}

		errs, ok := err.(ValidationErrors)
		if !ok {
			t.Fatalf("expected ValidationErrors type, got %T", err)
		}

		if len(errs) != 2 {
			t.Errorf("expected 2 errors, got %d: %v", len(errs), errs)
		}
	})
}
