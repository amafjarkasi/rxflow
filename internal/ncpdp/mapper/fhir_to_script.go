// Package mapper provides transformation logic between FHIR R5 and NCPDP SCRIPT v2023011.
package mapper

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	fhir "github.com/drfirst/go-oec/internal/fhir/r5"
	script "github.com/drfirst/go-oec/internal/ncpdp/script2023011"
	"github.com/google/uuid"
)

// FHIRToScriptMapper transforms FHIR R5 MedicationRequest to NCPDP SCRIPT v2023011
type FHIRToScriptMapper struct {
	// PatientResolver resolves patient details from reference
	PatientResolver func(reference string) (*fhir.Patient, error)
	// PractitionerResolver resolves practitioner details from reference
	PractitionerResolver func(reference string) (*fhir.Practitioner, error)
	// OrganizationResolver resolves organization (pharmacy) details from reference
	OrganizationResolver func(reference string) (*fhir.Organization, error)
}

// MapResult contains the mapping result
type MapResult struct {
	Message        *script.Message
	MessageID      string
	IdempotencyKey string
	IsControlled   bool
	DEASchedule    string
	PatientHash    string
}

// MapError represents a mapping error with context
type MapError struct {
	Field   string
	Code    string
	Message string
	Cause   error
}

func (e *MapError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%s)", e.Field, e.Message, e.Cause.Error())
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func (e *MapError) Unwrap() error {
	return e.Cause
}

// NewFHIRToScriptMapper creates a new mapper with default resolvers
func NewFHIRToScriptMapper() *FHIRToScriptMapper {
	return &FHIRToScriptMapper{
		PatientResolver:      defaultPatientResolver,
		PractitionerResolver: defaultPractitionerResolver,
		OrganizationResolver: defaultOrganizationResolver,
	}
}

// Default resolvers return empty structs (to be replaced with actual implementations)
func defaultPatientResolver(reference string) (*fhir.Patient, error) {
	return nil, &MapError{Field: "Patient", Code: "RESOLVER_NOT_CONFIGURED", Message: "patient resolver not configured"}
}

func defaultPractitionerResolver(reference string) (*fhir.Practitioner, error) {
	return nil, &MapError{Field: "Practitioner", Code: "RESOLVER_NOT_CONFIGURED", Message: "practitioner resolver not configured"}
}

func defaultOrganizationResolver(reference string) (*fhir.Organization, error) {
	return nil, &MapError{Field: "Organization", Code: "RESOLVER_NOT_CONFIGURED", Message: "organization resolver not configured"}
}

// MapToNewRx transforms a FHIR MedicationRequest to NCPDP SCRIPT NewRx
func (m *FHIRToScriptMapper) MapToNewRx(
	medRequest *fhir.MedicationRequest,
	patient *fhir.Patient,
	prescriber *fhir.Practitioner,
	pharmacy *fhir.Organization,
) (*MapResult, error) {
	if medRequest == nil {
		return nil, &MapError{Field: "MedicationRequest", Code: "NULL_INPUT", Message: "medication request is required"}
	}

	// Generate message ID
	messageID := uuid.New().String()

	// Generate idempotency key: Hash(PrescriberNPI + PatientMRN + MedicationCode + Timestamp)
	idempotencyKey := generateIdempotencyKey(medRequest, prescriber, patient)

	// Map prescriber
	scriptPrescriber, err := m.mapPrescriber(prescriber)
	if err != nil {
		return nil, err
	}

	// Map pharmacy
	scriptPharmacy, err := m.mapPharmacy(pharmacy)
	if err != nil {
		return nil, err
	}

	// Map patient
	scriptPatient, err := m.mapPatient(patient)
	if err != nil {
		return nil, err
	}

	// Map medication
	medicationPrescribed, err := m.mapMedication(medRequest)
	if err != nil {
		return nil, err
	}

	// Build NewRx
	newRx := &script.NewRx{
		Prescriber:           *scriptPrescriber,
		Pharmacy:             *scriptPharmacy,
		Patient:              *scriptPatient,
		MedicationPrescribed: *medicationPrescribed,
	}

	// Validate the NewRx
	if err := newRx.Validate(); err != nil {
		return nil, &MapError{Field: "NewRx", Code: "VALIDATION_FAILED", Message: "validation failed", Cause: err}
	}

	// Create the message
	message := script.NewNewRxMessage(messageID, newRx)

	// Set routing information in header
	message.Header.To = script.To{Pharmacy: scriptPharmacy}
	message.Header.From = script.From{Prescriber: scriptPrescriber}

	// Calculate patient hash for audit trail
	patientHash := calculatePatientHash(patient)

	return &MapResult{
		Message:        message,
		MessageID:      messageID,
		IdempotencyKey: idempotencyKey,
		IsControlled:   newRx.IsControlledSubstance(),
		DEASchedule:    newRx.GetDEASchedule(),
		PatientHash:    patientHash,
	}, nil
}

// mapPrescriber transforms FHIR Practitioner to SCRIPT Prescriber
func (m *FHIRToScriptMapper) mapPrescriber(practitioner *fhir.Practitioner) (*script.Prescriber, error) {
	if practitioner == nil {
		return nil, &MapError{Field: "Practitioner", Code: "NULL_INPUT", Message: "practitioner is required"}
	}

	name := practitioner.GetOfficialName()
	if name == nil {
		return nil, &MapError{Field: "Practitioner.Name", Code: "MISSING_FIELD", Message: "practitioner name is required"}
	}

	scriptPrescriber := &script.Prescriber{
		Identification: script.Identification{
			NPI:       practitioner.GetNPI(),
			DEANumber: practitioner.GetDEA(),
		},
		Name: script.Name{
			LastName:   name.Family,
			FirstName:  getFirstGiven(name.Given),
			MiddleName: getSecondGiven(name.Given),
			Prefix:     getFirstPrefix(name.Prefix),
			Suffix:     getFirstSuffix(name.Suffix),
		},
	}

	// Map address if available
	if len(practitioner.Address) > 0 {
		addr := practitioner.Address[0]
		scriptPrescriber.Address = &script.Address{
			AddressLine1: getFirstLine(addr.Line),
			AddressLine2: getSecondLine(addr.Line),
			City:         addr.City,
			State:        addr.State,
			PostalCode:   addr.PostalCode,
			CountryCode:  addr.Country,
		}
	}

	// Map telecom if available
	if len(practitioner.Telecom) > 0 {
		scriptPrescriber.CommunicationNumbers = mapTelecom(practitioner.Telecom)
	}

	return scriptPrescriber, nil
}

// mapPharmacy transforms FHIR Organization to SCRIPT Pharmacy
func (m *FHIRToScriptMapper) mapPharmacy(org *fhir.Organization) (*script.Pharmacy, error) {
	if org == nil {
		return nil, &MapError{Field: "Organization", Code: "NULL_INPUT", Message: "pharmacy organization is required"}
	}

	ncpdpID := org.GetNCPDPID()
	npi := org.GetNPI()
	if ncpdpID == "" && npi == "" {
		return nil, &MapError{Field: "Organization.Identifier", Code: "MISSING_FIELD", Message: "pharmacy NCPDP ID or NPI is required"}
	}

	scriptPharmacy := &script.Pharmacy{
		Identification: script.Identification{
			NCPDPID: ncpdpID,
			NPI:     npi,
		},
		StoreName: org.Name,
	}

	// Map address if available
	if len(org.Address) > 0 {
		addr := org.Address[0]
		scriptPharmacy.Address = &script.Address{
			AddressLine1: getFirstLine(addr.Line),
			AddressLine2: getSecondLine(addr.Line),
			City:         addr.City,
			State:        addr.State,
			PostalCode:   addr.PostalCode,
			CountryCode:  addr.Country,
		}
	}

	// Map telecom if available
	if len(org.Telecom) > 0 {
		scriptPharmacy.CommunicationNumbers = mapTelecom(org.Telecom)
	}

	return scriptPharmacy, nil
}

// mapPatient transforms FHIR Patient to SCRIPT Patient
func (m *FHIRToScriptMapper) mapPatient(patient *fhir.Patient) (*script.Patient, error) {
	if patient == nil {
		return nil, &MapError{Field: "Patient", Code: "NULL_INPUT", Message: "patient is required"}
	}

	name := patient.GetOfficialName()
	if name == nil {
		return nil, &MapError{Field: "Patient.Name", Code: "MISSING_FIELD", Message: "patient name is required"}
	}

	scriptPatient := &script.Patient{
		Name: script.Name{
			LastName:   name.Family,
			FirstName:  getFirstGiven(name.Given),
			MiddleName: getSecondGiven(name.Given),
			Prefix:     getFirstPrefix(name.Prefix),
			Suffix:     getFirstSuffix(name.Suffix),
		},
		Gender: mapGender(patient.Gender),
	}

	// Map identifiers
	mrn := patient.GetMRN()
	if mrn != "" {
		scriptPatient.Identification.MutuallyDefined = mrn
	}

	// Map date of birth
	if patient.BirthDate != "" {
		scriptPatient.DateOfBirth = &script.DateOfBirth{
			Date: formatFHIRDate(patient.BirthDate),
		}
	}

	// Map address if available
	addr := patient.GetHomeAddress()
	if addr != nil {
		scriptPatient.Address = &script.Address{
			AddressLine1: getFirstLine(addr.Line),
			AddressLine2: getSecondLine(addr.Line),
			City:         addr.City,
			State:        addr.State,
			PostalCode:   addr.PostalCode,
			CountryCode:  addr.Country,
		}
	}

	// Map telecom if available
	if len(patient.Telecom) > 0 {
		scriptPatient.CommunicationNumbers = mapTelecom(patient.Telecom)
	}

	return scriptPatient, nil
}

// mapMedication transforms FHIR MedicationRequest to SCRIPT MedicationPrescribed
func (m *FHIRToScriptMapper) mapMedication(medRequest *fhir.MedicationRequest) (*script.MedicationPrescribed, error) {
	// Get NDC code (required)
	ndc := medRequest.GetNDC()
	if ndc == "" {
		return nil, &MapError{Field: "MedicationRequest.Medication", Code: "MISSING_NDC", Message: "NDC code is required for NCPDP SCRIPT"}
	}

	// Build DrugCoded with v2023011 hierarchy
	drugCoded := script.DrugCoded{
		ProductCode: script.ProductCode{
			Code:      ndc,
			Qualifier: "ND", // NDC qualifier
		},
		ProductCodeQualifier: "ND",
	}

	// Add DEA schedule if controlled
	deaSchedule := getDEAScheduleFromFHIR(medRequest)
	if deaSchedule != "" {
		drugCoded.DEASchedule = &script.DEASchedule{Code: deaSchedule}
	}

	// Build Product
	product := script.Product{
		DrugCoded:       drugCoded,
		DrugDescription: medRequest.GetMedicationDisplay(),
	}

	// Add RxNorm as DrugDBCode
	rxnorm := medRequest.GetRxNorm()
	if rxnorm != "" {
		product.DrugDBCode = &script.DrugDBCode{
			Code:      rxnorm,
			Qualifier: "SBD", // RxNorm Semantic Branded Drug
		}
	}

	// Build MedicationPrescribed
	qty, _ := medRequest.GetQuantity()
	medicationPrescribed := &script.MedicationPrescribed{
		Product: product,
		Quantity: script.Quantity{
			Value:             fmt.Sprintf("%.2f", qty),
			CodeListQualifier: "38", // Original prescription
		},
		DaysSupply: script.DaysSupply{
			Value: fmt.Sprintf("%d", medRequest.GetDaysSupply()),
		},
		WrittenDate: script.WrittenDate{
			Date: script.FormatDate(time.Now()), // Will be overridden with actual date
		},
		Sig: script.Sig{
			SigText: medRequest.GetSigText(),
		},
		Refills: script.Refills{
			Value:     fmt.Sprintf("%d", medRequest.GetRefillsAllowed()),
			Qualifier: "R", // Refills
		},
	}

	// Handle written date from authoredOn
	if !medRequest.AuthoredOn.IsZero() {
		medicationPrescribed.WrittenDate.Date = script.FormatDate(medRequest.AuthoredOn)
	}

	// Handle substitution
	if !medRequest.IsSubstitutionAllowed() {
		medicationPrescribed.Substitutions = &script.Substitutions{
			NoSubstitution: script.SubstitutionNotAllowed,
		}
	}

	// Add controlled substance schedule if applicable
	if deaSchedule != "" {
		medicationPrescribed.ControlledSubstanceSchedule = deaSchedule
	}

	return medicationPrescribed, nil
}

// generateIdempotencyKey creates a deterministic key for deduplication
func generateIdempotencyKey(medRequest *fhir.MedicationRequest, prescriber *fhir.Practitioner, patient *fhir.Patient) string {
	var parts []string

	// Add prescriber NPI
	if prescriber != nil {
		parts = append(parts, prescriber.GetNPI())
	}

	// Add patient MRN or identifier
	if patient != nil {
		parts = append(parts, patient.GetMRN())
	}

	// Add medication code
	_, code := medRequest.GetMedicationCode()
	parts = append(parts, code)

	// Add authored date (truncated to minute for slight clock drift tolerance)
	if !medRequest.AuthoredOn.IsZero() {
		// Truncate to minute: YYYY-MM-DDTHH:MM format
		parts = append(parts, medRequest.AuthoredOn.Format("2006-01-02T15:04"))
	}

	// Hash the combined parts
	data := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// calculatePatientHash creates a SHA-256 hash of patient identifiers for audit
func calculatePatientHash(patient *fhir.Patient) string {
	if patient == nil {
		return ""
	}

	var parts []string

	// Add MRN
	mrn := patient.GetMRN()
	if mrn != "" {
		parts = append(parts, mrn)
	}

	// Add name components
	name := patient.GetOfficialName()
	if name != nil {
		parts = append(parts, name.Family)
		parts = append(parts, strings.Join(name.Given, " "))
	}

	// Add DOB
	if patient.BirthDate != "" {
		parts = append(parts, patient.BirthDate)
	}

	data := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// mapGender converts FHIR gender to NCPDP gender code
func mapGender(fhirGender string) string {
	switch strings.ToLower(fhirGender) {
	case "male":
		return script.GenderMale
	case "female":
		return script.GenderFemale
	default:
		return script.GenderUnknown
	}
}

// getDEAScheduleFromFHIR extracts DEA schedule from FHIR MedicationRequest
func getDEAScheduleFromFHIR(medRequest *fhir.MedicationRequest) string {
	// Check medication coding for DEA schedule
	if medRequest.Medication.Concept != nil {
		for _, coding := range medRequest.Medication.Concept.Coding {
			if coding.System == fhir.SystemDEASchedule {
				return mapFHIRDEAToNCPDP(coding.Code)
			}
		}
	}
	return ""
}

// mapFHIRDEAToNCPDP converts FHIR DEA schedule codes to NCPDP codes
func mapFHIRDEAToNCPDP(fhirCode string) string {
	switch fhirCode {
	case fhir.DEAScheduleII, "2", "II":
		return script.DEAScheduleII
	case fhir.DEAScheduleIII, "3", "III":
		return script.DEAScheduleIII
	case fhir.DEAScheduleIV, "4", "IV":
		return script.DEAScheduleIV
	case fhir.DEAScheduleV, "5", "V":
		return script.DEAScheduleV
	default:
		return script.DEAScheduleNone
	}
}

// formatFHIRDate converts FHIR date (YYYY-MM-DD) to NCPDP date (CCYYMMDD)
func formatFHIRDate(fhirDate string) string {
	// Remove dashes from FHIR date format
	return strings.ReplaceAll(fhirDate, "-", "")
}

// formatFHIRDateTime converts FHIR datetime to NCPDP date
func formatFHIRDateTime(fhirDateTime string) string {
	// Take just the date portion and format
	if len(fhirDateTime) >= 10 {
		return formatFHIRDate(fhirDateTime[:10])
	}
	return formatFHIRDate(fhirDateTime)
}

// mapTelecom converts FHIR ContactPoint array to SCRIPT CommunicationNumbers
func mapTelecom(telecoms []fhir.ContactPoint) *script.CommunicationNumbers {
	comm := &script.CommunicationNumbers{}

	for _, t := range telecoms {
		switch t.System {
		case "phone":
			if t.Use == "work" || t.Use == "mobile" || comm.PrimaryTelephone == nil {
				if comm.PrimaryTelephone == nil {
					comm.PrimaryTelephone = &script.Telephone{Number: t.Value}
				} else {
					comm.OtherTelephone = &script.Telephone{Number: t.Value}
				}
			}
		case "fax":
			if comm.Fax == nil {
				comm.Fax = &script.Telephone{Number: t.Value}
			}
		case "email":
			if comm.ElectronicMail == "" {
				comm.ElectronicMail = t.Value
			}
		}
	}

	return comm
}

// Helper functions for extracting name/address components
func getFirstGiven(given []string) string {
	if len(given) > 0 {
		return given[0]
	}
	return ""
}

func getSecondGiven(given []string) string {
	if len(given) > 1 {
		return given[1]
	}
	return ""
}

func getFirstPrefix(prefixes []string) string {
	if len(prefixes) > 0 {
		return prefixes[0]
	}
	return ""
}

func getFirstSuffix(suffixes []string) string {
	if len(suffixes) > 0 {
		return suffixes[0]
	}
	return ""
}

func getFirstLine(lines []string) string {
	if len(lines) > 0 {
		return lines[0]
	}
	return ""
}

func getSecondLine(lines []string) string {
	if len(lines) > 1 {
		return lines[1]
	}
	return ""
}
