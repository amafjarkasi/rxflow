// Package r5 provides FHIR R5 data structures for the prescription orchestration engine.
package r5

import "time"

// Meta contains metadata about a resource.
type Meta struct {
	VersionID   string    `json:"versionId,omitempty"`
	LastUpdated time.Time `json:"lastUpdated,omitempty"`
	Source      string    `json:"source,omitempty"`
	Profile     []string  `json:"profile,omitempty"`
	Security    []Coding  `json:"security,omitempty"`
	Tag         []Coding  `json:"tag,omitempty"`
}

// Identifier represents a FHIR Identifier.
type Identifier struct {
	Use      string           `json:"use,omitempty"` // usual | official | temp | secondary | old
	Type     *CodeableConcept `json:"type,omitempty"`
	System   string           `json:"system,omitempty"`
	Value    string           `json:"value,omitempty"`
	Period   *Period          `json:"period,omitempty"`
	Assigner *Reference       `json:"assigner,omitempty"`
}

// CodeableConcept represents a concept with text and codings.
type CodeableConcept struct {
	Coding []Coding `json:"coding,omitempty"`
	Text   string   `json:"text,omitempty"`
}

// Coding represents a code from a terminology system.
type Coding struct {
	System       string `json:"system,omitempty"`
	Version      string `json:"version,omitempty"`
	Code         string `json:"code,omitempty"`
	Display      string `json:"display,omitempty"`
	UserSelected bool   `json:"userSelected,omitempty"`
}

// Reference represents a reference to another resource.
type Reference struct {
	Reference  string      `json:"reference,omitempty"`
	Type       string      `json:"type,omitempty"`
	Identifier *Identifier `json:"identifier,omitempty"`
	Display    string      `json:"display,omitempty"`
}

// CodeableReference is new in FHIR R5 - can be either a CodeableConcept or a Reference.
type CodeableReference struct {
	Concept   *CodeableConcept `json:"concept,omitempty"`
	Reference *Reference       `json:"reference,omitempty"`
}

// Period represents a time period.
type Period struct {
	Start time.Time `json:"start,omitempty"`
	End   time.Time `json:"end,omitempty"`
}

// Quantity represents a measured amount.
type Quantity struct {
	Value      float64 `json:"value,omitempty"`
	Comparator string  `json:"comparator,omitempty"`
	Unit       string  `json:"unit,omitempty"`
	System     string  `json:"system,omitempty"`
	Code       string  `json:"code,omitempty"`
}

// Duration is a Quantity with a temporal unit.
type Duration struct {
	Value  float64 `json:"value,omitempty"`
	Unit   string  `json:"unit,omitempty"`
	System string  `json:"system,omitempty"`
	Code   string  `json:"code,omitempty"`
}

// Range represents a range of values.
type Range struct {
	Low  *Quantity `json:"low,omitempty"`
	High *Quantity `json:"high,omitempty"`
}

// Ratio represents a ratio between two quantities.
type Ratio struct {
	Numerator   *Quantity `json:"numerator,omitempty"`
	Denominator *Quantity `json:"denominator,omitempty"`
}

// Annotation represents a note or comment.
type Annotation struct {
	AuthorReference *Reference `json:"authorReference,omitempty"`
	AuthorString    string     `json:"authorString,omitempty"`
	Time            time.Time  `json:"time,omitempty"`
	Text            string     `json:"text"`
}

// HumanName represents a human name.
type HumanName struct {
	Use    string   `json:"use,omitempty"` // usual | official | temp | nickname | anonymous | old | maiden
	Text   string   `json:"text,omitempty"`
	Family string   `json:"family,omitempty"`
	Given  []string `json:"given,omitempty"`
	Prefix []string `json:"prefix,omitempty"`
	Suffix []string `json:"suffix,omitempty"`
	Period *Period  `json:"period,omitempty"`
}

// Address represents a postal address.
type Address struct {
	Use        string   `json:"use,omitempty"`  // home | work | temp | old | billing
	Type       string   `json:"type,omitempty"` // postal | physical | both
	Text       string   `json:"text,omitempty"`
	Line       []string `json:"line,omitempty"`
	City       string   `json:"city,omitempty"`
	District   string   `json:"district,omitempty"`
	State      string   `json:"state,omitempty"`
	PostalCode string   `json:"postalCode,omitempty"`
	Country    string   `json:"country,omitempty"`
	Period     *Period  `json:"period,omitempty"`
}

// ContactPoint represents a contact detail.
type ContactPoint struct {
	System string  `json:"system,omitempty"` // phone | fax | email | pager | url | sms | other
	Value  string  `json:"value,omitempty"`
	Use    string  `json:"use,omitempty"` // home | work | temp | old | mobile
	Rank   int     `json:"rank,omitempty"`
	Period *Period `json:"period,omitempty"`
}

// Extension represents a FHIR extension.
type Extension struct {
	URL                  string           `json:"url"`
	ValueString          string           `json:"valueString,omitempty"`
	ValueBoolean         *bool            `json:"valueBoolean,omitempty"`
	ValueInteger         *int             `json:"valueInteger,omitempty"`
	ValueDecimal         *float64         `json:"valueDecimal,omitempty"`
	ValueCode            string           `json:"valueCode,omitempty"`
	ValueCoding          *Coding          `json:"valueCoding,omitempty"`
	ValueCodeableConcept *CodeableConcept `json:"valueCodeableConcept,omitempty"`
	ValueIdentifier      *Identifier      `json:"valueIdentifier,omitempty"`
	ValueReference       *Reference       `json:"valueReference,omitempty"`
}

// OperationOutcome represents errors and warnings from FHIR operations.
type OperationOutcome struct {
	ResourceType string                  `json:"resourceType"`
	Issue        []OperationOutcomeIssue `json:"issue"`
}

// OperationOutcomeIssue represents a single issue in an OperationOutcome.
type OperationOutcomeIssue struct {
	Severity    string           `json:"severity"` // fatal | error | warning | information
	Code        string           `json:"code"`
	Details     *CodeableConcept `json:"details,omitempty"`
	Diagnostics string           `json:"diagnostics,omitempty"`
	Location    []string         `json:"location,omitempty"`
	Expression  []string         `json:"expression,omitempty"`
}

// NewOperationOutcome creates a new OperationOutcome with the given issues.
func NewOperationOutcome(issues ...OperationOutcomeIssue) *OperationOutcome {
	return &OperationOutcome{
		ResourceType: "OperationOutcome",
		Issue:        issues,
	}
}

// NewErrorOutcome creates an OperationOutcome with a single error issue.
func NewErrorOutcome(code, diagnostics string) *OperationOutcome {
	return NewOperationOutcome(OperationOutcomeIssue{
		Severity:    "error",
		Code:        code,
		Diagnostics: diagnostics,
	})
}

// Common code systems
const (
	SystemRxNorm       = "http://www.nlm.nih.gov/research/umls/rxnorm"
	SystemNDC          = "http://hl7.org/fhir/sid/ndc"
	SystemSNOMED       = "http://snomed.info/sct"
	SystemLOINC        = "http://loinc.org"
	SystemNPI          = "http://hl7.org/fhir/sid/us-npi"
	SystemDEA          = "http://hl7.org/fhir/sid/us-dea"
	SystemUCUM         = "http://unitsofmeasure.org"
	SystemNCIThesaurus = "http://ncithesaurus-stage.nci.nih.gov"
	SystemNCPDP        = "http://hl7.org/fhir/sid/ncpdp"
	SystemMRN          = "http://hospital.example.org/mrn"
	SystemDEASchedule  = "http://terminology.hl7.org/CodeSystem/DEADrugSchedule"
)

// Common medication request statuses
const (
	StatusActive         = "active"
	StatusOnHold         = "on-hold"
	StatusCancelled      = "cancelled"
	StatusCompleted      = "completed"
	StatusEnteredInError = "entered-in-error"
	StatusStopped        = "stopped"
	StatusDraft          = "draft"
	StatusUnknown        = "unknown"
)

// Common medication request intents
const (
	IntentProposal      = "proposal"
	IntentPlan          = "plan"
	IntentOrder         = "order"
	IntentOriginalOrder = "original-order"
	IntentReflexOrder   = "reflex-order"
	IntentFillerOrder   = "filler-order"
	IntentInstanceOrder = "instance-order"
	IntentOption        = "option"
)

// DEA schedules for controlled substances
const (
	DEAScheduleII  = "C-II"
	DEAScheduleIII = "C-III"
	DEAScheduleIV  = "C-IV"
	DEAScheduleV   = "C-V"
)
