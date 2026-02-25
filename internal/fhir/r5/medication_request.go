// Package r5 provides FHIR R5 data structures for the prescription orchestration engine.
package r5

import (
	"encoding/json"
	"time"
)

// MedicationRequest represents a FHIR R5 MedicationRequest resource.
// This is the primary resource for prescription orders.
type MedicationRequest struct {
	ResourceType string `json:"resourceType"`
	ID           string `json:"id,omitempty"`
	Meta         *Meta  `json:"meta,omitempty"`

	// Identifiers
	Identifier []Identifier `json:"identifier,omitempty"`

	// Status of the prescription
	Status       string           `json:"status"` // active | on-hold | cancelled | completed | entered-in-error | stopped | draft | unknown
	StatusReason *CodeableConcept `json:"statusReason,omitempty"`

	// Intent of the request
	Intent string `json:"intent"` // proposal | plan | order | original-order | reflex-order | filler-order | instance-order | option

	// Category of medication usage
	Category []CodeableConcept `json:"category,omitempty"`

	// Priority
	Priority string `json:"priority,omitempty"` // routine | urgent | asap | stat

	// Do not perform flag
	DoNotPerform bool `json:"doNotPerform,omitempty"`

	// Medication being requested (R5 uses CodeableReference)
	Medication CodeableReference `json:"medication"`

	// Subject (patient) for whom the medication is prescribed
	Subject Reference `json:"subject"`

	// Encounter during which request was created
	Encounter *Reference `json:"encounter,omitempty"`

	// Supporting information
	SupportingInformation []Reference `json:"supportingInformation,omitempty"`

	// When request was initially authored
	AuthoredOn time.Time `json:"authoredOn"`

	// Who/What requested the medication
	Requester *Reference `json:"requester,omitempty"`

	// Intended performer of the medication
	Performer []Reference `json:"performer,omitempty"`

	// Intended type of performer
	PerformerType *CodeableConcept `json:"performerType,omitempty"`

	// Desired kind of performer
	Recorder *Reference `json:"recorder,omitempty"`

	// Reason for the prescription
	Reason []CodeableReference `json:"reason,omitempty"`

	// Associated insurance coverage
	Insurance []Reference `json:"insurance,omitempty"`

	// Additional notes about the prescription
	Note []Annotation `json:"note,omitempty"`

	// Rendered dosage instruction (human-readable sig)
	RenderedDosageInstruction string `json:"renderedDosageInstruction,omitempty"`

	// Dosage instructions
	DosageInstruction []Dosage `json:"dosageInstruction,omitempty"`

	// Dispense request
	DispenseRequest *DispenseRequest `json:"dispenseRequest,omitempty"`

	// Substitution allowance
	Substitution *Substitution `json:"substitution,omitempty"`

	// Prior prescription being replaced
	PriorPrescription *Reference `json:"priorPrescription,omitempty"`

	// Event history
	EventHistory []Reference `json:"eventHistory,omitempty"`
}

// DispenseRequest contains information about the requested dispensing.
type DispenseRequest struct {
	// Initial fill
	InitialFill *InitialFill `json:"initialFill,omitempty"`

	// Minimum time between dispensing
	DispenseInterval *Duration `json:"dispenseInterval,omitempty"`

	// Validity period for the prescription
	ValidityPeriod *Period `json:"validityPeriod,omitempty"`

	// Number of refills authorized
	NumberOfRepeatsAllowed int `json:"numberOfRepeatsAllowed,omitempty"`

	// Quantity per dispense
	Quantity *Quantity `json:"quantity,omitempty"`

	// Expected supply duration
	ExpectedSupplyDuration *Duration `json:"expectedSupplyDuration,omitempty"`

	// Intended dispenser
	Dispenser *Reference `json:"dispenser,omitempty"`

	// Instructions for the dispenser
	DispenserInstruction []Annotation `json:"dispenserInstruction,omitempty"`

	// Dose administration aid
	DoseAdministrationAid *CodeableConcept `json:"doseAdministrationAid,omitempty"`
}

// InitialFill contains information about the initial dispensing.
type InitialFill struct {
	Quantity *Quantity `json:"quantity,omitempty"`
	Duration *Duration `json:"duration,omitempty"`
}

// Substitution contains information about medication substitution.
type Substitution struct {
	// Whether substitution is allowed (boolean or CodeableConcept)
	AllowedBoolean         *bool            `json:"allowedBoolean,omitempty"`
	AllowedCodeableConcept *CodeableConcept `json:"allowedCodeableConcept,omitempty"`

	// Reason for substitution
	Reason *CodeableConcept `json:"reason,omitempty"`
}

// Dosage contains dosage instructions for the medication.
type Dosage struct {
	Sequence                 int               `json:"sequence,omitempty"`
	Text                     string            `json:"text,omitempty"`
	AdditionalInstruction    []CodeableConcept `json:"additionalInstruction,omitempty"`
	PatientInstruction       string            `json:"patientInstruction,omitempty"`
	Timing                   *Timing           `json:"timing,omitempty"`
	AsNeeded                 bool              `json:"asNeeded,omitempty"`
	AsNeededFor              []CodeableConcept `json:"asNeededFor,omitempty"`
	Site                     *CodeableConcept  `json:"site,omitempty"`
	Route                    *CodeableConcept  `json:"route,omitempty"`
	Method                   *CodeableConcept  `json:"method,omitempty"`
	DoseAndRate              []DoseAndRate     `json:"doseAndRate,omitempty"`
	MaxDosePerPeriod         []Ratio           `json:"maxDosePerPeriod,omitempty"`
	MaxDosePerAdministration *Quantity         `json:"maxDosePerAdministration,omitempty"`
	MaxDosePerLifetime       *Quantity         `json:"maxDosePerLifetime,omitempty"`
}

// DoseAndRate contains dose/rate information.
type DoseAndRate struct {
	Type         *CodeableConcept `json:"type,omitempty"`
	DoseRange    *Range           `json:"doseRange,omitempty"`
	DoseQuantity *Quantity        `json:"doseQuantity,omitempty"`
	RateRatio    *Ratio           `json:"rateRatio,omitempty"`
	RateRange    *Range           `json:"rateRange,omitempty"`
	RateQuantity *Quantity        `json:"rateQuantity,omitempty"`
}

// Timing contains timing information for dosage.
type Timing struct {
	Event  []time.Time      `json:"event,omitempty"`
	Repeat *TimingRepeat    `json:"repeat,omitempty"`
	Code   *CodeableConcept `json:"code,omitempty"`
}

// TimingRepeat contains repeat details for timing.
type TimingRepeat struct {
	BoundsDuration *Duration `json:"boundsDuration,omitempty"`
	BoundsRange    *Range    `json:"boundsRange,omitempty"`
	BoundsPeriod   *Period   `json:"boundsPeriod,omitempty"`
	Count          int       `json:"count,omitempty"`
	CountMax       int       `json:"countMax,omitempty"`
	Duration       float64   `json:"duration,omitempty"`
	DurationMax    float64   `json:"durationMax,omitempty"`
	DurationUnit   string    `json:"durationUnit,omitempty"`
	Frequency      int       `json:"frequency,omitempty"`
	FrequencyMax   int       `json:"frequencyMax,omitempty"`
	Period         float64   `json:"period,omitempty"`
	PeriodMax      float64   `json:"periodMax,omitempty"`
	PeriodUnit     string    `json:"periodUnit,omitempty"`
	DayOfWeek      []string  `json:"dayOfWeek,omitempty"`
	TimeOfDay      []string  `json:"timeOfDay,omitempty"`
	When           []string  `json:"when,omitempty"`
	Offset         int       `json:"offset,omitempty"`
}

// GetPatientID extracts the patient ID from the Subject reference.
func (m *MedicationRequest) GetPatientID() string {
	if m.Subject.Reference != "" {
		// Extract ID from reference like "Patient/123"
		return extractIDFromReference(m.Subject.Reference)
	}
	return ""
}

// GetPrescriberNPI extracts the NPI from the Requester reference.
func (m *MedicationRequest) GetPrescriberNPI() string {
	if m.Requester == nil || m.Requester.Identifier == nil {
		return ""
	}
	if m.Requester.Identifier.System == SystemNPI {
		return m.Requester.Identifier.Value
	}
	return ""
}

// GetPrescriberDEA extracts the DEA number from extensions or identifiers.
func (m *MedicationRequest) GetPrescriberDEA() string {
	if m.Requester == nil || m.Requester.Identifier == nil {
		return ""
	}
	if m.Requester.Identifier.System == SystemDEA {
		return m.Requester.Identifier.Value
	}
	return ""
}

// GetMedicationCode extracts the primary medication code (RxNorm or NDC).
func (m *MedicationRequest) GetMedicationCode() (system, code string) {
	if m.Medication.Concept == nil {
		return "", ""
	}
	// Prefer RxNorm
	for _, coding := range m.Medication.Concept.Coding {
		if coding.System == "http://www.nlm.nih.gov/research/umls/rxnorm" {
			return "rxnorm", coding.Code
		}
	}
	// Fall back to NDC
	for _, coding := range m.Medication.Concept.Coding {
		if coding.System == "http://hl7.org/fhir/sid/ndc" {
			return "ndc", coding.Code
		}
	}
	// Return first available
	if len(m.Medication.Concept.Coding) > 0 {
		return m.Medication.Concept.Coding[0].System, m.Medication.Concept.Coding[0].Code
	}
	return "", ""
}

// GetNDC extracts the NDC code from the medication.
func (m *MedicationRequest) GetNDC() string {
	if m.Medication.Concept == nil {
		return ""
	}
	for _, coding := range m.Medication.Concept.Coding {
		if coding.System == "http://hl7.org/fhir/sid/ndc" {
			return coding.Code
		}
	}
	return ""
}

// GetRxNorm extracts the RxNorm CUI from the medication.
func (m *MedicationRequest) GetRxNorm() string {
	if m.Medication.Concept == nil {
		return ""
	}
	for _, coding := range m.Medication.Concept.Coding {
		if coding.System == "http://www.nlm.nih.gov/research/umls/rxnorm" {
			return coding.Code
		}
	}
	return ""
}

// GetMedicationDisplay returns the display name of the medication.
func (m *MedicationRequest) GetMedicationDisplay() string {
	if m.Medication.Concept != nil && m.Medication.Concept.Text != "" {
		return m.Medication.Concept.Text
	}
	if m.Medication.Concept != nil && len(m.Medication.Concept.Coding) > 0 {
		return m.Medication.Concept.Coding[0].Display
	}
	return ""
}

// GetQuantity returns the dispense quantity.
func (m *MedicationRequest) GetQuantity() (value float64, unit string) {
	if m.DispenseRequest == nil || m.DispenseRequest.Quantity == nil {
		return 0, ""
	}
	return m.DispenseRequest.Quantity.Value, m.DispenseRequest.Quantity.Unit
}

// GetDaysSupply returns the expected supply duration in days.
func (m *MedicationRequest) GetDaysSupply() int {
	if m.DispenseRequest == nil || m.DispenseRequest.ExpectedSupplyDuration == nil {
		return 0
	}
	// Assume days if unit is "d" or empty
	return int(m.DispenseRequest.ExpectedSupplyDuration.Value)
}

// GetRefillsAllowed returns the number of refills authorized.
func (m *MedicationRequest) GetRefillsAllowed() int {
	if m.DispenseRequest == nil {
		return 0
	}
	return m.DispenseRequest.NumberOfRepeatsAllowed
}

// IsSubstitutionAllowed returns whether substitution is allowed.
func (m *MedicationRequest) IsSubstitutionAllowed() bool {
	if m.Substitution == nil {
		return true // Default to allowed
	}
	if m.Substitution.AllowedBoolean != nil {
		return *m.Substitution.AllowedBoolean
	}
	return true
}

// GetSigText returns the rendered dosage instruction (sig).
func (m *MedicationRequest) GetSigText() string {
	if m.RenderedDosageInstruction != "" {
		return m.RenderedDosageInstruction
	}
	if len(m.DosageInstruction) > 0 && m.DosageInstruction[0].Text != "" {
		return m.DosageInstruction[0].Text
	}
	return ""
}

// ToJSON serializes the MedicationRequest to JSON.
func (m *MedicationRequest) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON deserializes a MedicationRequest from JSON.
func (m *MedicationRequest) FromJSON(data []byte) error {
	return json.Unmarshal(data, m)
}

// extractIDFromReference extracts the ID from a FHIR reference string.
func extractIDFromReference(ref string) string {
	// Handle references like "Patient/123" or "urn:uuid:123"
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '/' || ref[i] == ':' {
			return ref[i+1:]
		}
	}
	return ref
}
