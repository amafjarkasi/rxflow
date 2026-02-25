// Package script2023011 provides additional NCPDP SCRIPT message types.
package script2023011

// RxRenewal represents a prescription renewal request/response
type RxRenewal struct {
	Prescriber           Prescriber            `xml:"Prescriber"`
	Pharmacy             Pharmacy              `xml:"Pharmacy"`
	Patient              Patient               `xml:"Patient"`
	MedicationPrescribed MedicationPrescribed  `xml:"MedicationPrescribed"`
	Facility             *Facility             `xml:"Facility,omitempty"`
	BenefitsCoordination *BenefitsCoordination `xml:"BenefitsCoordination,omitempty"`
	RenewalRequestReason string                `xml:"RenewalRequestReason,omitempty"`
	ResponseCode         string                `xml:"ResponseCode,omitempty"`
	ResponseNote         string                `xml:"ResponseNote,omitempty"`
}

// RxChange represents a prescription change request/response
type RxChange struct {
	Prescriber           Prescriber            `xml:"Prescriber"`
	Pharmacy             Pharmacy              `xml:"Pharmacy"`
	Patient              Patient               `xml:"Patient"`
	MedicationPrescribed MedicationPrescribed  `xml:"MedicationPrescribed"`
	MedicationRequested  *MedicationPrescribed `xml:"MedicationRequested,omitempty"`
	Facility             *Facility             `xml:"Facility,omitempty"`
	BenefitsCoordination *BenefitsCoordination `xml:"BenefitsCoordination,omitempty"`
	ChangeRequestType    string                `xml:"ChangeRequestType,omitempty"`
	ChangeRequestReason  string                `xml:"ChangeRequestReason,omitempty"`
	ResponseCode         string                `xml:"ResponseCode,omitempty"`
	ResponseNote         string                `xml:"ResponseNote,omitempty"`
}

// Change request type constants
const (
	ChangeTypeGenericSubstitution     = "G"
	ChangeTypeTherapeuticSubstitution = "T"
	ChangeTypeScriptClarification     = "S"
	ChangeTypePriorAuthorization      = "P"
	ChangeTypeOutOfStock              = "OS"
	ChangeTypePatientRequest          = "PR"
)

// CancelRx represents a prescription cancellation request/response
type CancelRx struct {
	Prescriber           Prescriber           `xml:"Prescriber"`
	Pharmacy             Pharmacy             `xml:"Pharmacy"`
	Patient              Patient              `xml:"Patient"`
	MedicationPrescribed MedicationPrescribed `xml:"MedicationPrescribed"`
	Facility             *Facility            `xml:"Facility,omitempty"`
	CancelReason         string               `xml:"CancelReason,omitempty"`
	ResponseCode         string               `xml:"ResponseCode,omitempty"`
	ResponseNote         string               `xml:"ResponseNote,omitempty"`
}

// Cancel reason constants
const (
	CancelReasonPrescriberRequest = "01"
	CancelReasonPatientRequest    = "02"
	CancelReasonDuplicateRx       = "03"
	CancelReasonWrongPatient      = "04"
	CancelReasonWrongDrug         = "05"
	CancelReasonWrongPharmacy     = "06"
	CancelReasonPatientDeceased   = "07"
	CancelReasonOther             = "99"
)

// Status represents a status notification message
type Status struct {
	Prescriber           *Prescriber           `xml:"Prescriber,omitempty"`
	Pharmacy             *Pharmacy             `xml:"Pharmacy,omitempty"`
	Patient              *Patient              `xml:"Patient,omitempty"`
	MedicationPrescribed *MedicationPrescribed `xml:"MedicationPrescribed,omitempty"`
	Code                 string                `xml:"Code"`
	Description          string                `xml:"Description,omitempty"`
	RelatesToMessageID   string                `xml:"RelatesToMessageID,omitempty"`
}

// Error represents an error response message
type Error struct {
	Code               string `xml:"Code"`
	Description        string `xml:"Description,omitempty"`
	DescriptionCode    string `xml:"DescriptionCode,omitempty"`
	RelatesToMessageID string `xml:"RelatesToMessageID,omitempty"`
}

// Error code constants
const (
	ErrorCodeSuccess                  = "000"
	ErrorCodeRecordNotFound           = "001"
	ErrorCodeInvalidMessage           = "002"
	ErrorCodeMessageTimeout           = "003"
	ErrorCodeSystemError              = "004"
	ErrorCodeInvalidPrescriber        = "010"
	ErrorCodeInvalidPharmacy          = "011"
	ErrorCodeInvalidPatient           = "012"
	ErrorCodeInvalidMedication        = "013"
	ErrorCodeInvalidDEANumber         = "020"
	ErrorCodeInvalidNPI               = "021"
	ErrorCodeInvalidNCPDPID           = "022"
	ErrorCodeControlledSubstanceError = "030"
	ErrorCodeEPCSValidationFailed     = "031"
	ErrorCodeDuplicatePrescription    = "040"
	ErrorCodePharmacyClosed           = "050"
	ErrorCodePharmacyNotAccepting     = "051"
)

// NewStatusMessage creates a new SCRIPT status message
func NewStatusMessage(messageID, relatesToID, code, description string) *Message {
	return &Message{
		Xmlns:          NamespaceScript,
		XmlnsXsi:       NamespaceXSI,
		SchemaLocation: SchemaLocation,
		Version:        Version,
		Header: Header{
			MessageID:          messageID,
			RelatesToMessageID: relatesToID,
			SentTime:           FormatDateTime(now()),
			SenderSoftware: &SenderSoftware{
				SenderSoftwareDeveloper:      "Prescription Orchestration Engine",
				SenderSoftwareProduct:        "go-oec",
				SenderSoftwareVersionRelease: "1.0.0",
			},
		},
		Body: MessageBody{
			Status: &Status{
				Code:               code,
				Description:        description,
				RelatesToMessageID: relatesToID,
			},
		},
	}
}

// NewErrorMessage creates a new SCRIPT error message
func NewErrorMessage(messageID, relatesToID, code, description string) *Message {
	return &Message{
		Xmlns:          NamespaceScript,
		XmlnsXsi:       NamespaceXSI,
		SchemaLocation: SchemaLocation,
		Version:        Version,
		Header: Header{
			MessageID:          messageID,
			RelatesToMessageID: relatesToID,
			SentTime:           FormatDateTime(now()),
			SenderSoftware: &SenderSoftware{
				SenderSoftwareDeveloper:      "Prescription Orchestration Engine",
				SenderSoftwareProduct:        "go-oec",
				SenderSoftwareVersionRelease: "1.0.0",
			},
		},
		Body: MessageBody{
			Error: &Error{
				Code:               code,
				Description:        description,
				RelatesToMessageID: relatesToID,
			},
		},
	}
}

// Validate performs basic validation on the RxRenewal message
func (r *RxRenewal) Validate() error {
	if r.Prescriber.Name.LastName == "" {
		return &ValidationError{Field: "Prescriber.Name.LastName", Message: "prescriber last name is required"}
	}
	if r.Pharmacy.Identification.NCPDPID == "" && r.Pharmacy.Identification.NPI == "" {
		return &ValidationError{Field: "Pharmacy.Identification", Message: "pharmacy NCPDP ID or NPI is required"}
	}
	if r.Patient.Name.LastName == "" {
		return &ValidationError{Field: "Patient.Name.LastName", Message: "patient last name is required"}
	}
	return nil
}

// Validate performs basic validation on the RxChange message
func (r *RxChange) Validate() error {
	if r.Prescriber.Name.LastName == "" {
		return &ValidationError{Field: "Prescriber.Name.LastName", Message: "prescriber last name is required"}
	}
	if r.Pharmacy.Identification.NCPDPID == "" && r.Pharmacy.Identification.NPI == "" {
		return &ValidationError{Field: "Pharmacy.Identification", Message: "pharmacy NCPDP ID or NPI is required"}
	}
	if r.ChangeRequestType == "" {
		return &ValidationError{Field: "ChangeRequestType", Message: "change request type is required"}
	}
	return nil
}

// Validate performs basic validation on the CancelRx message
func (c *CancelRx) Validate() error {
	if c.Prescriber.Name.LastName == "" {
		return &ValidationError{Field: "Prescriber.Name.LastName", Message: "prescriber last name is required"}
	}
	if c.Pharmacy.Identification.NCPDPID == "" && c.Pharmacy.Identification.NPI == "" {
		return &ValidationError{Field: "Pharmacy.Identification", Message: "pharmacy NCPDP ID or NPI is required"}
	}
	return nil
}

// ValidationError represents a validation error with field context
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
