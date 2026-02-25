// Package mapper provides reverse transformation from NCPDP SCRIPT to FHIR R5.
package mapper

import (
	"strings"

	fhir "github.com/drfirst/go-oec/internal/fhir/r5"
	script "github.com/drfirst/go-oec/internal/ncpdp/script2023011"
)

// ScriptToFHIRMapper transforms NCPDP SCRIPT v2023011 messages back to FHIR R5
type ScriptToFHIRMapper struct{}

// NewScriptToFHIRMapper creates a new reverse mapper
func NewScriptToFHIRMapper() *ScriptToFHIRMapper {
	return &ScriptToFHIRMapper{}
}

// MapStatusToOperationOutcome converts a SCRIPT Status to FHIR OperationOutcome
func (m *ScriptToFHIRMapper) MapStatusToOperationOutcome(status *script.Status) *fhir.OperationOutcome {
	severity := "information"
	code := "informational"

	// Determine severity based on status code
	if status.Code != script.StatusCodeSuccess && status.Code != script.StatusCodeAccepted {
		severity = "error"
		code = "processing"
	}

	return &fhir.OperationOutcome{
		ResourceType: "OperationOutcome",
		Issue: []fhir.OperationOutcomeIssue{
			{
				Severity:    severity,
				Code:        code,
				Diagnostics: status.Description,
				Details: &fhir.CodeableConcept{
					Coding: []fhir.Coding{
						{
							System:  "http://ncpdp.org/SCRIPT/StatusCode",
							Code:    status.Code,
							Display: status.Description,
						},
					},
				},
			},
		},
	}
}

// MapErrorToOperationOutcome converts a SCRIPT Error to FHIR OperationOutcome
func (m *ScriptToFHIRMapper) MapErrorToOperationOutcome(err *script.Error) *fhir.OperationOutcome {
	severity := "error"
	code := mapScriptErrorToFHIRCode(err.Code)

	return &fhir.OperationOutcome{
		ResourceType: "OperationOutcome",
		Issue: []fhir.OperationOutcomeIssue{
			{
				Severity:    severity,
				Code:        code,
				Diagnostics: err.Description,
				Details: &fhir.CodeableConcept{
					Coding: []fhir.Coding{
						{
							System:  "http://ncpdp.org/SCRIPT/ErrorCode",
							Code:    err.Code,
							Display: err.Description,
						},
					},
				},
			},
		},
	}
}

// MapPatientFromScript converts SCRIPT Patient to FHIR Patient
func (m *ScriptToFHIRMapper) MapPatientFromScript(scriptPatient *script.Patient) *fhir.Patient {
	if scriptPatient == nil {
		return nil
	}

	patient := &fhir.Patient{
		ResourceType: "Patient",
		Name: []fhir.HumanName{
			{
				Use:    "official",
				Family: scriptPatient.Name.LastName,
				Given:  buildGivenNames(scriptPatient.Name.FirstName, scriptPatient.Name.MiddleName),
				Prefix: buildStringSlice(scriptPatient.Name.Prefix),
				Suffix: buildStringSlice(scriptPatient.Name.Suffix),
			},
		},
		Gender: mapNCPDPGenderToFHIR(scriptPatient.Gender),
	}

	// Map date of birth
	if scriptPatient.DateOfBirth != nil && scriptPatient.DateOfBirth.Date != "" {
		patient.BirthDate = formatNCPDPDateToFHIR(scriptPatient.DateOfBirth.Date)
	}

	// Map address
	if scriptPatient.Address != nil {
		patient.Address = []fhir.Address{
			{
				Use:        "home",
				Line:       buildAddressLines(scriptPatient.Address.AddressLine1, scriptPatient.Address.AddressLine2),
				City:       scriptPatient.Address.City,
				State:      scriptPatient.Address.State,
				PostalCode: scriptPatient.Address.PostalCode,
				Country:    scriptPatient.Address.CountryCode,
			},
		}
	}

	// Map telecom
	if scriptPatient.CommunicationNumbers != nil {
		patient.Telecom = mapCommunicationNumbersToFHIR(scriptPatient.CommunicationNumbers)
	}

	// Map identifiers
	if scriptPatient.Identification.MutuallyDefined != "" {
		patient.Identifier = append(patient.Identifier, fhir.Identifier{
			Use:    "usual",
			System: fhir.SystemMRN,
			Value:  scriptPatient.Identification.MutuallyDefined,
		})
	}

	return patient
}

// MapPrescriberFromScript converts SCRIPT Prescriber to FHIR Practitioner
func (m *ScriptToFHIRMapper) MapPrescriberFromScript(scriptPrescriber *script.Prescriber) *fhir.Practitioner {
	if scriptPrescriber == nil {
		return nil
	}

	practitioner := &fhir.Practitioner{
		ResourceType: "Practitioner",
		Name: []fhir.HumanName{
			{
				Use:    "official",
				Family: scriptPrescriber.Name.LastName,
				Given:  buildGivenNames(scriptPrescriber.Name.FirstName, scriptPrescriber.Name.MiddleName),
				Prefix: buildStringSlice(scriptPrescriber.Name.Prefix),
				Suffix: buildStringSlice(scriptPrescriber.Name.Suffix),
			},
		},
	}

	// Map NPI
	if scriptPrescriber.Identification.NPI != "" {
		practitioner.Identifier = append(practitioner.Identifier, fhir.Identifier{
			System: fhir.SystemNPI,
			Value:  scriptPrescriber.Identification.NPI,
		})
	}

	// Map DEA
	if scriptPrescriber.Identification.DEANumber != "" {
		practitioner.Identifier = append(practitioner.Identifier, fhir.Identifier{
			System: fhir.SystemDEA,
			Value:  scriptPrescriber.Identification.DEANumber,
		})
	}

	// Map address
	if scriptPrescriber.Address != nil {
		practitioner.Address = []fhir.Address{
			{
				Use:        "work",
				Line:       buildAddressLines(scriptPrescriber.Address.AddressLine1, scriptPrescriber.Address.AddressLine2),
				City:       scriptPrescriber.Address.City,
				State:      scriptPrescriber.Address.State,
				PostalCode: scriptPrescriber.Address.PostalCode,
				Country:    scriptPrescriber.Address.CountryCode,
			},
		}
	}

	// Map telecom
	if scriptPrescriber.CommunicationNumbers != nil {
		practitioner.Telecom = mapCommunicationNumbersToFHIR(scriptPrescriber.CommunicationNumbers)
	}

	return practitioner
}

// MapPharmacyFromScript converts SCRIPT Pharmacy to FHIR Organization
func (m *ScriptToFHIRMapper) MapPharmacyFromScript(scriptPharmacy *script.Pharmacy) *fhir.Organization {
	if scriptPharmacy == nil {
		return nil
	}

	org := &fhir.Organization{
		ResourceType: "Organization",
		Name:         scriptPharmacy.StoreName,
		Type: []fhir.CodeableConcept{
			{
				Coding: []fhir.Coding{
					{
						System:  "http://terminology.hl7.org/CodeSystem/organization-type",
						Code:    "prov",
						Display: "Healthcare Provider",
					},
				},
				Text: "Pharmacy",
			},
		},
	}

	// Map NCPDP ID
	if scriptPharmacy.Identification.NCPDPID != "" {
		org.Identifier = append(org.Identifier, fhir.Identifier{
			System: fhir.SystemNCPDP,
			Value:  scriptPharmacy.Identification.NCPDPID,
		})
	}

	// Map NPI
	if scriptPharmacy.Identification.NPI != "" {
		org.Identifier = append(org.Identifier, fhir.Identifier{
			System: fhir.SystemNPI,
			Value:  scriptPharmacy.Identification.NPI,
		})
	}

	// Map address
	if scriptPharmacy.Address != nil {
		org.Address = []fhir.Address{
			{
				Use:        "work",
				Line:       buildAddressLines(scriptPharmacy.Address.AddressLine1, scriptPharmacy.Address.AddressLine2),
				City:       scriptPharmacy.Address.City,
				State:      scriptPharmacy.Address.State,
				PostalCode: scriptPharmacy.Address.PostalCode,
				Country:    scriptPharmacy.Address.CountryCode,
			},
		}
	}

	// Map telecom
	if scriptPharmacy.CommunicationNumbers != nil {
		org.Telecom = mapCommunicationNumbersToFHIR(scriptPharmacy.CommunicationNumbers)
	}

	return org
}

// mapScriptErrorToFHIRCode maps SCRIPT error codes to FHIR issue codes
func mapScriptErrorToFHIRCode(scriptCode string) string {
	switch scriptCode {
	case script.ErrorCodeRecordNotFound:
		return "not-found"
	case script.ErrorCodeInvalidMessage:
		return "invalid"
	case script.ErrorCodeMessageTimeout:
		return "timeout"
	case script.ErrorCodeSystemError:
		return "exception"
	case script.ErrorCodeInvalidPrescriber, script.ErrorCodeInvalidNPI, script.ErrorCodeInvalidDEANumber:
		return "security"
	case script.ErrorCodeInvalidPharmacy, script.ErrorCodeInvalidNCPDPID:
		return "not-found"
	case script.ErrorCodeInvalidPatient:
		return "not-found"
	case script.ErrorCodeInvalidMedication:
		return "code-invalid"
	case script.ErrorCodeControlledSubstanceError, script.ErrorCodeEPCSValidationFailed:
		return "business-rule"
	case script.ErrorCodeDuplicatePrescription:
		return "duplicate"
	default:
		return "processing"
	}
}

// mapNCPDPGenderToFHIR converts NCPDP gender codes to FHIR gender
func mapNCPDPGenderToFHIR(ncpdpGender string) string {
	switch ncpdpGender {
	case script.GenderMale:
		return "male"
	case script.GenderFemale:
		return "female"
	default:
		return "unknown"
	}
}

// formatNCPDPDateToFHIR converts NCPDP date (CCYYMMDD) to FHIR date (YYYY-MM-DD)
func formatNCPDPDateToFHIR(ncpdpDate string) string {
	if len(ncpdpDate) != 8 {
		return ncpdpDate
	}
	return ncpdpDate[:4] + "-" + ncpdpDate[4:6] + "-" + ncpdpDate[6:8]
}

// mapCommunicationNumbersToFHIR converts SCRIPT CommunicationNumbers to FHIR ContactPoint array
func mapCommunicationNumbersToFHIR(comm *script.CommunicationNumbers) []fhir.ContactPoint {
	var telecoms []fhir.ContactPoint

	if comm.PrimaryTelephone != nil {
		telecoms = append(telecoms, fhir.ContactPoint{
			System: "phone",
			Value:  comm.PrimaryTelephone.Number,
			Use:    "work",
		})
	}

	if comm.OtherTelephone != nil {
		telecoms = append(telecoms, fhir.ContactPoint{
			System: "phone",
			Value:  comm.OtherTelephone.Number,
			Use:    "mobile",
		})
	}

	if comm.Fax != nil {
		telecoms = append(telecoms, fhir.ContactPoint{
			System: "fax",
			Value:  comm.Fax.Number,
			Use:    "work",
		})
	}

	if comm.ElectronicMail != "" {
		telecoms = append(telecoms, fhir.ContactPoint{
			System: "email",
			Value:  comm.ElectronicMail,
			Use:    "work",
		})
	}

	return telecoms
}

// Helper functions
func buildGivenNames(first, middle string) []string {
	var names []string
	if first != "" {
		names = append(names, first)
	}
	if middle != "" {
		names = append(names, middle)
	}
	return names
}

func buildStringSlice(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

func buildAddressLines(line1, line2 string) []string {
	var lines []string
	if line1 != "" {
		lines = append(lines, line1)
	}
	if line2 != "" {
		lines = append(lines, line2)
	}
	return lines
}

// MapMessageTypeToFHIRTask maps SCRIPT message types to FHIR Task status/intent
func MapMessageTypeToFHIRTask(messageType string) (status, intent string) {
	switch strings.ToUpper(messageType) {
	case script.MessageTypeNewRx:
		return "requested", "order"
	case script.MessageTypeRxRenewal:
		return "requested", "reflex-order"
	case script.MessageTypeRxChange:
		return "requested", "proposal"
	case script.MessageTypeCancel:
		return "cancelled", "order"
	case script.MessageTypeStatus:
		return "in-progress", "order"
	case script.MessageTypeError:
		return "failed", "order"
	default:
		return "draft", "order"
	}
}
