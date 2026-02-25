// Package script2023011 provides NCPDP SCRIPT Standard v2023011 XML structures.
// This version implements significant structural changes from v2017071, including
// the new MedicationPrescribed/Product/DrugCoded hierarchy.
package script2023011

import (
	"encoding/xml"
	"time"
)

// XML Namespace constants for NCPDP SCRIPT v2023011
const (
	NamespaceScript = "http://www.ncpdp.org/schema/SCRIPT"
	NamespaceXSI    = "http://www.w3.org/2001/XMLSchema-instance"
	SchemaLocation  = "http://www.ncpdp.org/schema/SCRIPT SCRIPT_v2023011.xsd"
	Version         = "2023011"
)

// Message type constants
const (
	MessageTypeNewRx     = "NEWRX"
	MessageTypeRxRenewal = "RXRENEWAL"
	MessageTypeRxChange  = "RXCHG"
	MessageTypeCancel    = "CANRX"
	MessageTypeStatus    = "STATUS"
	MessageTypeError     = "ERROR"
	MessageTypeVerify    = "VERIFY"
)

// Status code constants
const (
	StatusCodeSuccess           = "000"
	StatusCodeAccepted          = "010"
	StatusCodeValidationError   = "600"
	StatusCodeTransmissionError = "900"
)

// DEA Schedule constants aligned with NCPDP terminology
const (
	DEAScheduleII   = "C48675" // Schedule II
	DEAScheduleIII  = "C48676" // Schedule III
	DEAScheduleIV   = "C48677" // Schedule IV
	DEAScheduleV    = "C48679" // Schedule V
	DEAScheduleNone = "C48680" // Non-controlled
)

// Gender code constants (NCPDP)
const (
	GenderMale    = "1"
	GenderFemale  = "2"
	GenderUnknown = "0"
)

// Message represents the root NCPDP SCRIPT message envelope
type Message struct {
	XMLName        xml.Name    `xml:"Message"`
	Xmlns          string      `xml:"xmlns,attr"`
	XmlnsXsi       string      `xml:"xmlns:xsi,attr,omitempty"`
	SchemaLocation string      `xml:"xsi:schemaLocation,attr,omitempty"`
	Version        string      `xml:"version,attr"`
	Release        string      `xml:"release,attr,omitempty"`
	Header         Header      `xml:"Header"`
	Body           MessageBody `xml:"Body"`
}

// MessageBody contains the actual message content (NewRx, RxChange, etc.)
type MessageBody struct {
	NewRx     *NewRx     `xml:"NewRx,omitempty"`
	RxRenewal *RxRenewal `xml:"RxRenewal,omitempty"`
	RxChange  *RxChange  `xml:"RxChange,omitempty"`
	CancelRx  *CancelRx  `xml:"CancelRx,omitempty"`
	Status    *Status    `xml:"Status,omitempty"`
	Error     *Error     `xml:"Error,omitempty"`
}

// Header contains message routing and identification information
type Header struct {
	To                 To                `xml:"To"`
	From               From              `xml:"From"`
	MessageID          string            `xml:"MessageID"`
	RelatesToMessageID string            `xml:"RelatesToMessageID,omitempty"`
	SentTime           string            `xml:"SentTime"`
	Security           *Security         `xml:"Security,omitempty"`
	SenderSoftware     *SenderSoftware   `xml:"SenderSoftware,omitempty"`
	Mailbox            *Mailbox          `xml:"Mailbox,omitempty"`
	TestMessage        string            `xml:"TestMessage,omitempty"`
	DigitalSignature   *DigitalSignature `xml:"DigitalSignature,omitempty"`
}

// To represents the message recipient (pharmacy)
type To struct {
	Pharmacy *Pharmacy `xml:"Pharmacy,omitempty"`
}

// From represents the message sender (prescriber/facility)
type From struct {
	Prescriber *Prescriber `xml:"Prescriber,omitempty"`
	Facility   *Facility   `xml:"Facility,omitempty"`
}

// Security contains authentication information for EPCS
type Security struct {
	Sender        *SecuritySender   `xml:"Sender,omitempty"`
	Receiver      *SecurityReceiver `xml:"Receiver,omitempty"`
	UsernameToken *UsernameToken    `xml:"UsernameToken,omitempty"`
}

// SecuritySender contains sender security credentials
type SecuritySender struct {
	TertiaryIdentification string `xml:"TertiaryIdentification,omitempty"`
}

// SecurityReceiver contains receiver security credentials
type SecurityReceiver struct {
	TertiaryIdentification string `xml:"TertiaryIdentification,omitempty"`
}

// UsernameToken contains username/password authentication
type UsernameToken struct {
	Username string `xml:"Username"`
	Password string `xml:"Password,omitempty"`
}

// SenderSoftware identifies the sending application
type SenderSoftware struct {
	SenderSoftwareDeveloper      string `xml:"SenderSoftwareDeveloper"`
	SenderSoftwareProduct        string `xml:"SenderSoftwareProduct"`
	SenderSoftwareVersionRelease string `xml:"SenderSoftwareVersionRelease"`
}

// Mailbox contains store-and-forward mailbox information
type Mailbox struct {
	MailboxID string `xml:"MailboxID"`
}

// DigitalSignature contains EPCS digital signature data
type DigitalSignature struct {
	SignatureValue     string `xml:"SignatureValue"`
	SignatureAlgorithm string `xml:"SignatureAlgorithm,omitempty"`
	CertificateID      string `xml:"CertificateID,omitempty"`
	HashValue          string `xml:"HashValue,omitempty"`
}

// Identification provides identifier type and value
type Identification struct {
	NPI                string `xml:"NPI,omitempty"`
	DEANumber          string `xml:"DEANumber,omitempty"`
	NCPDPID            string `xml:"NCPDPID,omitempty"`
	StateLicenseNumber string `xml:"StateLicenseNumber,omitempty"`
	FileID             string `xml:"FileID,omitempty"`
	MutuallyDefined    string `xml:"MutuallyDefined,omitempty"`
	SocialSecurity     string `xml:"SocialSecurity,omitempty"`
	MedicareNumber     string `xml:"MedicareNumber,omitempty"`
	MedicaidNumber     string `xml:"MedicaidNumber,omitempty"`
	PayerID            string `xml:"PayerID,omitempty"`
	BINLocationNumber  string `xml:"BINLocationNumber,omitempty"`
}

// Name represents a person's name
type Name struct {
	LastName   string `xml:"LastName"`
	FirstName  string `xml:"FirstName"`
	MiddleName string `xml:"MiddleName,omitempty"`
	Suffix     string `xml:"Suffix,omitempty"`
	Prefix     string `xml:"Prefix,omitempty"`
}

// Address represents a physical address
type Address struct {
	AddressLine1 string `xml:"AddressLine1"`
	AddressLine2 string `xml:"AddressLine2,omitempty"`
	City         string `xml:"City"`
	State        string `xml:"State,omitempty"`
	PostalCode   string `xml:"PostalCode,omitempty"`
	CountryCode  string `xml:"CountryCode,omitempty"`
}

// CommunicationNumbers contains phone/fax numbers
type CommunicationNumbers struct {
	PrimaryTelephone *Telephone `xml:"PrimaryTelephone,omitempty"`
	OtherTelephone   *Telephone `xml:"OtherTelephone,omitempty"`
	Fax              *Telephone `xml:"Fax,omitempty"`
	ElectronicMail   string     `xml:"ElectronicMail,omitempty"`
}

// Telephone represents a phone number
type Telephone struct {
	Number    string `xml:"Number"`
	Extension string `xml:"Extension,omitempty"`
	Qualifier string `xml:"Qualifier,omitempty"`
}

// Pharmacy represents a pharmacy entity
type Pharmacy struct {
	Identification       Identification        `xml:"Identification"`
	StoreName            string                `xml:"StoreName,omitempty"`
	Pharmacist           *Pharmacist           `xml:"Pharmacist,omitempty"`
	Address              *Address              `xml:"Address,omitempty"`
	CommunicationNumbers *CommunicationNumbers `xml:"CommunicationNumbers,omitempty"`
}

// Pharmacist represents an individual pharmacist
type Pharmacist struct {
	Name           Name           `xml:"Name"`
	Identification Identification `xml:"Identification,omitempty"`
}

// Prescriber represents a prescribing provider
type Prescriber struct {
	Identification       Identification        `xml:"Identification"`
	Name                 Name                  `xml:"Name"`
	Specialty            *Specialty            `xml:"Specialty,omitempty"`
	Address              *Address              `xml:"Address,omitempty"`
	CommunicationNumbers *CommunicationNumbers `xml:"CommunicationNumbers,omitempty"`
	PrescriberAgent      *PrescriberAgent      `xml:"PrescriberAgent,omitempty"`
}

// PrescriberAgent represents an agent acting on behalf of prescriber
type PrescriberAgent struct {
	Name           Name           `xml:"Name"`
	Identification Identification `xml:"Identification,omitempty"`
}

// Specialty represents provider specialty
type Specialty struct {
	Code        string `xml:"Code,omitempty"`
	Description string `xml:"Description,omitempty"`
	Qualifier   string `xml:"Qualifier,omitempty"`
}

// Facility represents a healthcare facility
type Facility struct {
	Identification       Identification        `xml:"Identification"`
	FacilityName         string                `xml:"FacilityName"`
	Address              *Address              `xml:"Address,omitempty"`
	CommunicationNumbers *CommunicationNumbers `xml:"CommunicationNumbers,omitempty"`
}

// Patient represents a patient
type Patient struct {
	Name                 Name                  `xml:"Name"`
	Identification       Identification        `xml:"Identification,omitempty"`
	Gender               string                `xml:"Gender,omitempty"`
	DateOfBirth          *DateOfBirth          `xml:"DateOfBirth,omitempty"`
	Address              *Address              `xml:"Address,omitempty"`
	CommunicationNumbers *CommunicationNumbers `xml:"CommunicationNumbers,omitempty"`
	PatientRelationship  string                `xml:"PatientRelationship,omitempty"`
}

// DateOfBirth contains date of birth information
type DateOfBirth struct {
	Date string `xml:"Date"`
}

// Quantity represents a quantity with units
type Quantity struct {
	Value             string `xml:"Value"`
	CodeListQualifier string `xml:"CodeListQualifier,omitempty"`
	UnitSourceCode    string `xml:"UnitSourceCode,omitempty"`
	PotencyUnitCode   string `xml:"PotencyUnitCode,omitempty"`
}

// DaysSupply represents days supply information
type DaysSupply struct {
	Value string `xml:"Value"`
}

// Refills represents refill information
type Refills struct {
	Value     string `xml:"Value"`
	Qualifier string `xml:"Qualifier,omitempty"`
}

// Diagnosis represents a diagnosis code
type Diagnosis struct {
	Primary   *DiagnosisCode `xml:"Primary,omitempty"`
	Secondary *DiagnosisCode `xml:"Secondary,omitempty"`
}

// DiagnosisCode represents a coded diagnosis
type DiagnosisCode struct {
	Code        string `xml:"Code"`
	Qualifier   string `xml:"Qualifier,omitempty"`
	Description string `xml:"Description,omitempty"`
}

// FormatDateTime formats a time.Time to NCPDP SCRIPT datetime format (CCYYMMDDHHMMSS)
func FormatDateTime(t time.Time) string {
	return t.Format("20060102150405")
}

// FormatDate formats a time.Time to NCPDP SCRIPT date format (CCYYMMDD)
func FormatDate(t time.Time) string {
	return t.Format("20060102")
}

// ParseDateTime parses an NCPDP SCRIPT datetime string
func ParseDateTime(s string) (time.Time, error) {
	return time.Parse("20060102150405", s)
}

// ParseDate parses an NCPDP SCRIPT date string
func ParseDate(s string) (time.Time, error) {
	return time.Parse("20060102", s)
}
