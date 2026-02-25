// Package prescription implements the prescription aggregate and domain events.
package prescription

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of domain event
type EventType string

const (
	EventPrescriptionCreated     EventType = "PrescriptionCreated"
	EventPrescriptionValidated   EventType = "PrescriptionValidated"
	EventPrescriptionRouted      EventType = "PrescriptionRouted"
	EventPrescriptionTransmitted EventType = "PrescriptionTransmitted"
	EventPrescriptionAccepted    EventType = "PrescriptionAccepted"
	EventPrescriptionRejected    EventType = "PrescriptionRejected"
	EventPrescriptionCancelled   EventType = "PrescriptionCancelled"
)

// Event represents a domain event
type Event struct {
	ID            string          `json:"id"`
	AggregateID   string          `json:"aggregate_id"`
	AggregateType string          `json:"aggregate_type"`
	EventType     EventType       `json:"event_type"`
	EventData     json.RawMessage `json:"event_data"`
	Version       int             `json:"version"`
	Timestamp     time.Time       `json:"timestamp"`
	PrescriberNPI string          `json:"prescriber_npi,omitempty"`
	PrescriberDEA string          `json:"prescriber_dea,omitempty"`
	PatientHash   string          `json:"patient_hash,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
}

// NewEvent creates a new event
func NewEvent(aggregateID string, eventType EventType, data interface{}) (*Event, error) {
	eventData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return &Event{
		ID:            uuid.New().String(),
		AggregateID:   aggregateID,
		AggregateType: "Prescription",
		EventType:     eventType,
		EventData:     eventData,
		Timestamp:     time.Now().UTC(),
	}, nil
}

// PrescriptionCreatedData contains prescription creation details
type PrescriptionCreatedData struct {
	PrescriptionID      string          `json:"prescription_id"`
	PatientHash         string          `json:"patient_hash"`
	PrescriberNPI       string          `json:"prescriber_npi"`
	PrescriberDEA       string          `json:"prescriber_dea,omitempty"`
	MedicationNDC       string          `json:"medication_ndc"`
	MedicationRxNorm    string          `json:"medication_rxnorm,omitempty"`
	MedicationName      string          `json:"medication_name"`
	IsControlled        bool            `json:"is_controlled"`
	DEASchedule         string          `json:"dea_schedule,omitempty"`
	Quantity            float64         `json:"quantity"`
	DaysSupply          int             `json:"days_supply"`
	RefillsAllowed      int             `json:"refills_allowed"`
	SigText             string          `json:"sig_text"`
	SubstitutionAllowed bool            `json:"substitution_allowed"`
	FHIRPayload         json.RawMessage `json:"fhir_payload,omitempty"`
	WrittenDate         time.Time       `json:"written_date"`
}

// PrescriptionRoutedData contains routing details
type PrescriptionRoutedData struct {
	PrescriptionID  string    `json:"prescription_id"`
	PharmacyNCPDPID string    `json:"pharmacy_ncpdp_id"`
	PharmacyName    string    `json:"pharmacy_name"`
	RoutedAt        time.Time `json:"routed_at"`
}

// PrescriptionTransmittedData contains transmission details
type PrescriptionTransmittedData struct {
	PrescriptionID  string    `json:"prescription_id"`
	PharmacyNCPDPID string    `json:"pharmacy_ncpdp_id"`
	MessageID       string    `json:"message_id"`
	TransmittedAt   time.Time `json:"transmitted_at"`
}

// WithAuditInfo sets audit fields
func (e *Event) WithAuditInfo(npi, dea, hash string) *Event {
	e.PrescriberNPI = npi
	e.PrescriberDEA = dea
	e.PatientHash = hash
	return e
}
