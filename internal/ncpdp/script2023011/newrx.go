// Package script2023011 provides NCPDP SCRIPT Standard v2023011 NewRx message structure.
package script2023011

import (
	"encoding/xml"
	"fmt"
	"time"
)

// NewRx represents a new prescription message (NEWRX transaction)
// This is the primary message type for transmitting new prescriptions to pharmacies
type NewRx struct {
	Prescriber           Prescriber            `xml:"Prescriber"`
	Pharmacy             Pharmacy              `xml:"Pharmacy"`
	Patient              Patient               `xml:"Patient"`
	MedicationPrescribed MedicationPrescribed  `xml:"MedicationPrescribed"`
	Facility             *Facility             `xml:"Facility,omitempty"`
	Supervisor           *Prescriber           `xml:"Supervisor,omitempty"`
	BenefitsCoordination *BenefitsCoordination `xml:"BenefitsCoordination,omitempty"`
}

// MedicationPrescribed represents the prescribed medication details
// This is the new v2023011 hierarchy replacing the older DrugDescription approach
type MedicationPrescribed struct {
	Product                     Product              `xml:"Product"`
	Quantity                    Quantity             `xml:"Quantity"`
	DaysSupply                  DaysSupply           `xml:"DaysSupply"`
	WrittenDate                 WrittenDate          `xml:"WrittenDate"`
	LastFillDate                *LastFillDate        `xml:"LastFillDate,omitempty"`
	Sig                         Sig                  `xml:"Sig"`
	Refills                     Refills              `xml:"Refills"`
	Substitutions               *Substitutions       `xml:"Substitutions,omitempty"`
	Note                        string               `xml:"Note,omitempty"`
	Diagnosis                   *Diagnosis           `xml:"Diagnosis,omitempty"`
	PriorAuthorization          *PriorAuthorization  `xml:"PriorAuthorization,omitempty"`
	DrugUseEvaluation           []DrugUseEvaluation  `xml:"DrugUseEvaluation,omitempty"`
	DrugCoverageStatusCode      string               `xml:"DrugCoverageStatusCode,omitempty"`
	CompoundInformation         *CompoundInformation `xml:"CompoundInformation,omitempty"`
	ControlledSubstanceSchedule string               `xml:"ControlledSubstanceSchedule,omitempty"`
	PrescriptionNumber          string               `xml:"PrescriptionNumber,omitempty"`
	PharmacyRequestedRefills    string               `xml:"PharmacyRequestedRefills,omitempty"`
}

// Product contains the drug product information
// This follows the new Product/DrugCoded hierarchy in v2023011
type Product struct {
	DrugCoded       DrugCoded   `xml:"DrugCoded"`
	DrugDescription string      `xml:"DrugDescription,omitempty"`
	Strength        *Strength   `xml:"Strength,omitempty"`
	DrugDBCode      *DrugDBCode `xml:"DrugDBCode,omitempty"`
}

// DrugCoded contains coded drug identifiers
type DrugCoded struct {
	ProductCode          ProductCode  `xml:"ProductCode"`
	ProductCodeQualifier string       `xml:"ProductCodeQualifier"`
	DEASchedule          *DEASchedule `xml:"DEASchedule,omitempty"`
	FormSourceCode       string       `xml:"FormSourceCode,omitempty"`
	FormCode             string       `xml:"FormCode,omitempty"`
}

// ProductCode contains the actual drug code (NDC, etc.)
type ProductCode struct {
	Code      string `xml:"Code"`
	Qualifier string `xml:"Qualifier,omitempty"`
}

// DEASchedule represents the DEA schedule for controlled substances
type DEASchedule struct {
	Code string `xml:"Code"`
}

// Strength contains drug strength information
type Strength struct {
	StrengthValue         string `xml:"StrengthValue"`
	StrengthForm          string `xml:"StrengthForm,omitempty"`
	StrengthUnitOfMeasure string `xml:"StrengthUnitOfMeasure,omitempty"`
}

// DrugDBCode contains database-specific drug codes (RxNorm, etc.)
type DrugDBCode struct {
	Code      string `xml:"Code"`
	Qualifier string `xml:"Qualifier"`
}

// WrittenDate contains the date the prescription was written
type WrittenDate struct {
	Date string `xml:"Date"`
}

// LastFillDate contains the date of last fill
type LastFillDate struct {
	Date string `xml:"Date"`
}

// Sig contains the prescription directions (signa)
type Sig struct {
	SigText               string              `xml:"SigText"`
	CodeSystem            *SigCodeSystem      `xml:"CodeSystem,omitempty"`
	DoseDeliveryMethod    *DoseDeliveryMethod `xml:"DoseDeliveryMethod,omitempty"`
	DoseForm              string              `xml:"DoseForm,omitempty"`
	RouteOfAdministration string              `xml:"RouteOfAdministration,omitempty"`
	Dosage                *SigDosage          `xml:"Dosage,omitempty"`
	Timing                *SigTiming          `xml:"Timing,omitempty"`
	Duration              *SigDuration        `xml:"Duration,omitempty"`
	Instructions          []SigInstruction    `xml:"Instruction,omitempty"`
}

// SigCodeSystem identifies the code system for structured sig
type SigCodeSystem struct {
	SNOMEDVersion string `xml:"SNOMEDVersion,omitempty"`
	FMTVersion    string `xml:"FMTVersion,omitempty"`
}

// DoseDeliveryMethod contains dose delivery information
type DoseDeliveryMethod struct {
	Code string `xml:"Code,omitempty"`
	Text string `xml:"Text,omitempty"`
}

// SigDosage contains dose amount information
type SigDosage struct {
	DoseQuantity      string `xml:"DoseQuantity,omitempty"`
	DoseUnitOfMeasure string `xml:"DoseUnitOfMeasure,omitempty"`
	DoseLow           string `xml:"DoseLow,omitempty"`
	DoseHigh          string `xml:"DoseHigh,omitempty"`
}

// SigTiming contains timing/frequency information
type SigTiming struct {
	FrequencyNumericValue string `xml:"FrequencyNumericValue,omitempty"`
	FrequencyUnits        string `xml:"FrequencyUnits,omitempty"`
	FrequencyCode         string `xml:"FrequencyCode,omitempty"`
	Interval              string `xml:"Interval,omitempty"`
	IntervalUnits         string `xml:"IntervalUnits,omitempty"`
}

// SigDuration contains duration information
type SigDuration struct {
	Value string `xml:"Value,omitempty"`
	Units string `xml:"Units,omitempty"`
}

// SigInstruction contains additional sig instructions
type SigInstruction struct {
	Text string `xml:"Text"`
	Code string `xml:"Code,omitempty"`
}

// Substitutions contains substitution allowance information
type Substitutions struct {
	NoSubstitution       string `xml:"NoSubstitution,omitempty"`
	NoSubstitutionReason string `xml:"NoSubstitutionReason,omitempty"`
}

// SubstitutionCode constants
const (
	SubstitutionAllowed    = "0"
	SubstitutionNotAllowed = "1"
)

// PriorAuthorization contains prior authorization information
type PriorAuthorization struct {
	PriorAuthorizationNumber string `xml:"PriorAuthorizationNumber,omitempty"`
	PriorAuthorizationStatus string `xml:"PriorAuthorizationStatus,omitempty"`
}

// DrugUseEvaluation contains DUR (Drug Utilization Review) information
type DrugUseEvaluation struct {
	ServiceReasonCode        string `xml:"ServiceReasonCode,omitempty"`
	ProfessionalServiceCode  string `xml:"ProfessionalServiceCode,omitempty"`
	ResultOfServiceCode      string `xml:"ResultOfServiceCode,omitempty"`
	CoAgentID                string `xml:"CoAgentID,omitempty"`
	CoAgentIDQualifier       string `xml:"CoAgentIDQualifier,omitempty"`
	ClinicalSignificanceCode string `xml:"ClinicalSignificanceCode,omitempty"`
	AcknowledgementReason    string `xml:"AcknowledgementReason,omitempty"`
}

// CompoundInformation contains compound medication details
type CompoundInformation struct {
	CompoundType string               `xml:"CompoundType,omitempty"`
	Ingredients  []CompoundIngredient `xml:"Ingredient,omitempty"`
}

// CompoundIngredient represents a single ingredient in a compound
type CompoundIngredient struct {
	ProductCode           string `xml:"ProductCode"`
	ProductCodeQualifier  string `xml:"ProductCodeQualifier,omitempty"`
	DrugDescription       string `xml:"DrugDescription,omitempty"`
	Quantity              string `xml:"Quantity,omitempty"`
	QuantityUnitOfMeasure string `xml:"QuantityUnitOfMeasure,omitempty"`
}

// BenefitsCoordination contains insurance/benefits information
type BenefitsCoordination struct {
	PayerIdentification *PayerIdentification `xml:"PayerIdentification,omitempty"`
	CardholderID        string               `xml:"CardholderID,omitempty"`
	CardholderName      *Name                `xml:"CardholderName,omitempty"`
	GroupID             string               `xml:"GroupID,omitempty"`
	GroupName           string               `xml:"GroupName,omitempty"`
	PBMMemberID         string               `xml:"PBMMemberID,omitempty"`
}

// PayerIdentification contains payer/PBM identification
type PayerIdentification struct {
	BINLocationNumber             string `xml:"BINLocationNumber,omitempty"`
	PayerID                       string `xml:"PayerID,omitempty"`
	PayerName                     string `xml:"PayerName,omitempty"`
	ProcessorIdentificationNumber string `xml:"ProcessorIdentificationNumber,omitempty"`
}

// NewNewRxMessage creates a new SCRIPT message with a NewRx body
func NewNewRxMessage(messageID string, newRx *NewRx) *Message {
	return &Message{
		Xmlns:          NamespaceScript,
		XmlnsXsi:       NamespaceXSI,
		SchemaLocation: SchemaLocation,
		Version:        Version,
		Header: Header{
			MessageID: messageID,
			SentTime:  FormatDateTime(now()),
			SenderSoftware: &SenderSoftware{
				SenderSoftwareDeveloper:      "Prescription Orchestration Engine",
				SenderSoftwareProduct:        "go-oec",
				SenderSoftwareVersionRelease: "1.0.0",
			},
		},
		Body: MessageBody{
			NewRx: newRx,
		},
	}
}

// ToXML marshals the message to XML with proper indentation
func (m *Message) ToXML() ([]byte, error) {
	return xml.MarshalIndent(m, "", "  ")
}

// ToXMLCompact marshals the message to compact XML (no indentation)
func (m *Message) ToXMLCompact() ([]byte, error) {
	return xml.Marshal(m)
}

// FromXML unmarshals a message from XML
func FromXML(data []byte) (*Message, error) {
	var msg Message
	if err := xml.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SCRIPT message: %w", err)
	}
	return &msg, nil
}

// Validate performs basic validation on the NewRx message
func (n *NewRx) Validate() error {
	if n.Prescriber.Name.LastName == "" {
		return fmt.Errorf("prescriber last name is required")
	}
	if n.Prescriber.Identification.NPI == "" && n.Prescriber.Identification.DEANumber == "" {
		return fmt.Errorf("prescriber NPI or DEA number is required")
	}
	if n.Pharmacy.Identification.NCPDPID == "" && n.Pharmacy.Identification.NPI == "" {
		return fmt.Errorf("pharmacy NCPDP ID or NPI is required")
	}
	if n.Patient.Name.LastName == "" {
		return fmt.Errorf("patient last name is required")
	}
	if n.MedicationPrescribed.Product.DrugCoded.ProductCode.Code == "" {
		return fmt.Errorf("medication product code (NDC) is required")
	}
	if n.MedicationPrescribed.Sig.SigText == "" {
		return fmt.Errorf("prescription directions (sig) are required")
	}
	if n.MedicationPrescribed.Quantity.Value == "" {
		return fmt.Errorf("quantity is required")
	}
	return nil
}

// IsControlledSubstance returns true if this prescription is for a controlled substance
func (n *NewRx) IsControlledSubstance() bool {
	schedule := n.MedicationPrescribed.ControlledSubstanceSchedule
	if schedule == "" && n.MedicationPrescribed.Product.DrugCoded.DEASchedule != nil {
		schedule = n.MedicationPrescribed.Product.DrugCoded.DEASchedule.Code
	}
	return schedule != "" && schedule != DEAScheduleNone
}

// GetDEASchedule returns the DEA schedule code for this prescription
func (n *NewRx) GetDEASchedule() string {
	if n.MedicationPrescribed.ControlledSubstanceSchedule != "" {
		return n.MedicationPrescribed.ControlledSubstanceSchedule
	}
	if n.MedicationPrescribed.Product.DrugCoded.DEASchedule != nil {
		return n.MedicationPrescribed.Product.DrugCoded.DEASchedule.Code
	}
	return ""
}

// timeNow is a variable to allow mocking in tests
var timeNow = time.Now

// now returns current time formatted
func now() time.Time {
	return timeNow()
}
