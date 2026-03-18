package prescription

import (
	"errors"
	"fmt"
	"strings"
)

// ValidationError represents a business rule violation
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed on field '%s': %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	var msgs []string
	for _, err := range v {
		msgs = append(msgs, err.Error())
	}
	return "prescription validation failed: " + strings.Join(msgs, "; ")
}

// Validate checks if the prescription meets all business rules for DEA and standard compliance
func (a *Aggregate) Validate() error {
	if a.status != StatusPending {
		return errors.New("prescription must be in pending status to be validated")
	}

	var errs ValidationErrors

	// 1. Basic required fields
	if a.patientHash == "" {
		errs = append(errs, ValidationError{Field: "PatientHash", Message: "is required for audit trail"})
	}
	if a.medicationNDC == "" {
		errs = append(errs, ValidationError{Field: "MedicationNDC", Message: "NDC is required"})
	}
	if a.quantity <= 0 {
		errs = append(errs, ValidationError{Field: "Quantity", Message: "must be greater than zero"})
	}
	if a.daysSupply <= 0 {
		errs = append(errs, ValidationError{Field: "DaysSupply", Message: "must be greater than zero"})
	}
	if a.sigText == "" {
		errs = append(errs, ValidationError{Field: "SigText", Message: "directions are required"})
	}

	// 2. Prescriber Identification Rules
	if a.prescriberNPI == "" && a.prescriberDEA == "" {
		errs = append(errs, ValidationError{Field: "PrescriberID", Message: "either NPI or DEA number must be provided"})
	}
	if a.prescriberNPI != "" && len(a.prescriberNPI) != 10 {
		errs = append(errs, ValidationError{Field: "PrescriberNPI", Message: "must be exactly 10 digits"})
	}
	if a.prescriberDEA != "" && len(a.prescriberDEA) != 9 {
		errs = append(errs, ValidationError{Field: "PrescriberDEA", Message: "must be exactly 9 characters"})
	}

	// 3. Controlled Substance Rules
	if a.isControlled {
		if a.prescriberDEA == "" {
			errs = append(errs, ValidationError{Field: "PrescriberDEA", Message: "is required for controlled substances"})
		}
		if a.deaSchedule == "" {
			errs = append(errs, ValidationError{Field: "DEASchedule", Message: "is required for controlled substances"})
		}

		// Schedule-specific rules
		switch a.deaSchedule {
		case "2", "II":
			if a.refillsAllowed > 0 {
				errs = append(errs, ValidationError{Field: "RefillsAllowed", Message: "Schedule II drugs cannot have refills"})
			}
			if a.daysSupply > 90 {
				errs = append(errs, ValidationError{Field: "DaysSupply", Message: "Schedule II drugs limited to 90 days supply"})
			}
		case "3", "III", "4", "IV", "5", "V":
			if a.refillsAllowed > 5 {
				errs = append(errs, ValidationError{Field: "RefillsAllowed", Message: "Schedule III-V drugs cannot have more than 5 refills"})
			}
			// Usually limited to 6 months total, but simplfying here for example
		}
	}

	if len(errs) > 0 {
		// Log validation failure event? Could be useful for audit
		return errs
	}

	// If valid, apply the transition event
	data := &PrescriptionValidatedData{
		PrescriptionID: a.id,
		ValidatedAt:    a.updatedAt,
	}

	event, err := NewEvent(a.id, EventPrescriptionValidated, data)
	if err != nil {
		return err
	}

	a.apply(event)
	a.changes = append(a.changes, event)

	return nil
}
