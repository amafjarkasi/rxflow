// Package prescription implements the prescription aggregate.
package prescription

import (
	"errors"
	"time"
)

// Status represents prescription status
type Status string

const (
	StatusDraft       Status = "draft"
	StatusPending     Status = "pending"
	StatusValidated   Status = "validated"
	StatusRouted      Status = "routed"
	StatusTransmitted Status = "transmitted"
	StatusAccepted    Status = "accepted"
	StatusRejected    Status = "rejected"
	StatusCancelled   Status = "cancelled"
)

// Aggregate represents the prescription aggregate root
type Aggregate struct {
	id              string
	version         int
	status          Status
	patientHash     string
	prescriberNPI   string
	prescriberDEA   string
	medicationNDC   string
	medicationName  string
	isControlled    bool
	deaSchedule     string
	quantity        float64
	daysSupply      int
	refillsAllowed  int
	sigText         string
	pharmacyNCPDPID string
	messageID       string
	writtenDate     time.Time
	createdAt       time.Time
	updatedAt       time.Time
	changes         []*Event
}

// NewAggregate creates a new prescription aggregate
func NewAggregate(id string) *Aggregate {
	return &Aggregate{
		id:        id,
		status:    StatusDraft,
		createdAt: time.Now().UTC(),
		updatedAt: time.Now().UTC(),
		changes:   make([]*Event, 0),
	}
}

// ID returns the aggregate ID
func (a *Aggregate) ID() string { return a.id }

// Version returns the current version
func (a *Aggregate) Version() int { return a.version }

// Status returns the current status
func (a *Aggregate) Status() Status { return a.status }

// Changes returns uncommitted events
func (a *Aggregate) Changes() []*Event { return a.changes }

// ClearChanges clears uncommitted events
func (a *Aggregate) ClearChanges() { a.changes = make([]*Event, 0) }

// Create initializes the prescription
func (a *Aggregate) Create(data *PrescriptionCreatedData) error {
	if a.status != StatusDraft {
		return errors.New("prescription already created")
	}

	event, err := NewEvent(a.id, EventPrescriptionCreated, data)
	if err != nil {
		return err
	}
	event.WithAuditInfo(data.PrescriberNPI, data.PrescriberDEA, data.PatientHash)

	a.apply(event)
	a.changes = append(a.changes, event)
	return nil
}

// Route assigns the prescription to a pharmacy
func (a *Aggregate) Route(pharmacyNCPDPID, pharmacyName string) error {
	if a.status != StatusPending && a.status != StatusValidated {
		return errors.New("prescription not ready for routing")
	}

	data := &PrescriptionRoutedData{
		PrescriptionID:  a.id,
		PharmacyNCPDPID: pharmacyNCPDPID,
		PharmacyName:    pharmacyName,
		RoutedAt:        time.Now().UTC(),
	}

	event, err := NewEvent(a.id, EventPrescriptionRouted, data)
	if err != nil {
		return err
	}

	a.apply(event)
	a.changes = append(a.changes, event)
	return nil
}

// MarkTransmitted records successful transmission
func (a *Aggregate) MarkTransmitted(messageID string) error {
	if a.status != StatusRouted {
		return errors.New("prescription not routed")
	}

	data := &PrescriptionTransmittedData{
		PrescriptionID:  a.id,
		PharmacyNCPDPID: a.pharmacyNCPDPID,
		MessageID:       messageID,
		TransmittedAt:   time.Now().UTC(),
	}

	event, err := NewEvent(a.id, EventPrescriptionTransmitted, data)
	if err != nil {
		return err
	}

	a.apply(event)
	a.changes = append(a.changes, event)
	return nil
}

// apply applies an event to update state
func (a *Aggregate) apply(event *Event) {
	a.version++
	a.updatedAt = event.Timestamp

	switch event.EventType {
	case EventPrescriptionCreated:
		a.applyCreated(event)
	case EventPrescriptionRouted:
		a.applyRouted(event)
	case EventPrescriptionTransmitted:
		a.applyTransmitted(event)
	case EventPrescriptionAccepted:
		a.status = StatusAccepted
	case EventPrescriptionRejected:
		a.status = StatusRejected
	case EventPrescriptionCancelled:
		a.status = StatusCancelled
	}
}

func (a *Aggregate) applyCreated(event *Event) {
	var data PrescriptionCreatedData
	if err := unmarshalEventData(event.EventData, &data); err != nil {
		return
	}
	a.status = StatusPending
	a.patientHash = data.PatientHash
	a.prescriberNPI = data.PrescriberNPI
	a.prescriberDEA = data.PrescriberDEA
	a.medicationNDC = data.MedicationNDC
	a.medicationName = data.MedicationName
	a.isControlled = data.IsControlled
	a.deaSchedule = data.DEASchedule
	a.quantity = data.Quantity
	a.daysSupply = data.DaysSupply
	a.refillsAllowed = data.RefillsAllowed
	a.sigText = data.SigText
	a.writtenDate = data.WrittenDate
}

func (a *Aggregate) applyRouted(event *Event) {
	var data PrescriptionRoutedData
	if err := unmarshalEventData(event.EventData, &data); err != nil {
		return
	}
	a.status = StatusRouted
	a.pharmacyNCPDPID = data.PharmacyNCPDPID
}

func (a *Aggregate) applyTransmitted(event *Event) {
	var data PrescriptionTransmittedData
	if err := unmarshalEventData(event.EventData, &data); err != nil {
		return
	}
	a.status = StatusTransmitted
	a.messageID = data.MessageID
}

// LoadFromHistory rebuilds state from events
func (a *Aggregate) LoadFromHistory(events []*Event) {
	for _, event := range events {
		a.apply(event)
	}
}

func unmarshalEventData(data []byte, v interface{}) error {
	return jsonUnmarshal(data, v)
}

// jsonUnmarshal is a variable for testing
var jsonUnmarshal = func(data []byte, v interface{}) error {
	return errors.New("not implemented")
}

func init() {
	jsonUnmarshal = func(data []byte, v interface{}) error {
		return nil // Will use encoding/json in real impl
	}
}
