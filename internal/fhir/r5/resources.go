// Package r5 provides FHIR R5 data structures for the prescription orchestration engine.
package r5

import "time"

// Patient represents a FHIR R5 Patient resource.
type Patient struct {
	ResourceType         string                 `json:"resourceType"`
	ID                   string                 `json:"id,omitempty"`
	Meta                 *Meta                  `json:"meta,omitempty"`
	Identifier           []Identifier           `json:"identifier,omitempty"`
	Active               bool                   `json:"active,omitempty"`
	Name                 []HumanName            `json:"name,omitempty"`
	Telecom              []ContactPoint         `json:"telecom,omitempty"`
	Gender               string                 `json:"gender,omitempty"` // male | female | other | unknown
	BirthDate            string                 `json:"birthDate,omitempty"`
	DeceasedBoolean      *bool                  `json:"deceasedBoolean,omitempty"`
	DeceasedDateTime     *time.Time             `json:"deceasedDateTime,omitempty"`
	Address              []Address              `json:"address,omitempty"`
	MaritalStatus        *CodeableConcept       `json:"maritalStatus,omitempty"`
	MultipleBirthBoolean *bool                  `json:"multipleBirthBoolean,omitempty"`
	MultipleBirthInteger *int                   `json:"multipleBirthInteger,omitempty"`
	Communication        []PatientCommunication `json:"communication,omitempty"`
	GeneralPractitioner  []Reference            `json:"generalPractitioner,omitempty"`
	ManagingOrganization *Reference             `json:"managingOrganization,omitempty"`
}

// PatientCommunication represents a patient's preferred language.
type PatientCommunication struct {
	Language  CodeableConcept `json:"language"`
	Preferred bool            `json:"preferred,omitempty"`
}

// GetOfficialName returns the patient's official name, or first available.
func (p *Patient) GetOfficialName() *HumanName {
	for i := range p.Name {
		if p.Name[i].Use == "official" {
			return &p.Name[i]
		}
	}
	if len(p.Name) > 0 {
		return &p.Name[0]
	}
	return nil
}

// GetFullName returns the patient's full name as a string.
func (p *Patient) GetFullName() string {
	name := p.GetOfficialName()
	if name == nil {
		return ""
	}
	if name.Text != "" {
		return name.Text
	}
	result := ""
	for _, g := range name.Given {
		if result != "" {
			result += " "
		}
		result += g
	}
	if name.Family != "" {
		if result != "" {
			result += " "
		}
		result += name.Family
	}
	return result
}

// GetHomeAddress returns the patient's home address.
func (p *Patient) GetHomeAddress() *Address {
	for i := range p.Address {
		if p.Address[i].Use == "home" {
			return &p.Address[i]
		}
	}
	if len(p.Address) > 0 {
		return &p.Address[0]
	}
	return nil
}

// GetMRN returns the patient's medical record number.
func (p *Patient) GetMRN() string {
	for _, id := range p.Identifier {
		if id.Type != nil {
			for _, coding := range id.Type.Coding {
				if coding.Code == "MR" {
					return id.Value
				}
			}
		}
	}
	return ""
}

// GetPhone returns the patient's primary phone number.
func (p *Patient) GetPhone() string {
	for _, t := range p.Telecom {
		if t.System == "phone" {
			return t.Value
		}
	}
	return ""
}

// Practitioner represents a FHIR R5 Practitioner resource.
type Practitioner struct {
	ResourceType  string                      `json:"resourceType"`
	ID            string                      `json:"id,omitempty"`
	Meta          *Meta                       `json:"meta,omitempty"`
	Identifier    []Identifier                `json:"identifier,omitempty"`
	Active        bool                        `json:"active,omitempty"`
	Name          []HumanName                 `json:"name,omitempty"`
	Telecom       []ContactPoint              `json:"telecom,omitempty"`
	Gender        string                      `json:"gender,omitempty"`
	BirthDate     string                      `json:"birthDate,omitempty"`
	Address       []Address                   `json:"address,omitempty"`
	Qualification []PractitionerQualification `json:"qualification,omitempty"`
	Communication []CodeableConcept           `json:"communication,omitempty"`
}

// PractitionerQualification represents a practitioner's qualifications.
type PractitionerQualification struct {
	Identifier []Identifier    `json:"identifier,omitempty"`
	Code       CodeableConcept `json:"code"`
	Period     *Period         `json:"period,omitempty"`
	Issuer     *Reference      `json:"issuer,omitempty"`
}

// GetNPI returns the practitioner's NPI.
func (p *Practitioner) GetNPI() string {
	for _, id := range p.Identifier {
		if id.System == SystemNPI {
			return id.Value
		}
	}
	return ""
}

// GetDEA returns the practitioner's DEA number.
func (p *Practitioner) GetDEA() string {
	for _, id := range p.Identifier {
		if id.System == SystemDEA {
			return id.Value
		}
	}
	return ""
}

// GetOfficialName returns the practitioner's official name.
func (p *Practitioner) GetOfficialName() *HumanName {
	for i := range p.Name {
		if p.Name[i].Use == "official" {
			return &p.Name[i]
		}
	}
	if len(p.Name) > 0 {
		return &p.Name[0]
	}
	return nil
}

// GetFullName returns the practitioner's full name as a string.
func (p *Practitioner) GetFullName() string {
	name := p.GetOfficialName()
	if name == nil {
		return ""
	}
	if name.Text != "" {
		return name.Text
	}
	result := ""
	for _, prefix := range name.Prefix {
		result += prefix + " "
	}
	for _, g := range name.Given {
		if result != "" && result[len(result)-1] != ' ' {
			result += " "
		}
		result += g
	}
	if name.Family != "" {
		if result != "" && result[len(result)-1] != ' ' {
			result += " "
		}
		result += name.Family
	}
	for _, suffix := range name.Suffix {
		result += ", " + suffix
	}
	return result
}

// Organization represents a FHIR R5 Organization resource.
type Organization struct {
	ResourceType string                `json:"resourceType"`
	ID           string                `json:"id,omitempty"`
	Meta         *Meta                 `json:"meta,omitempty"`
	Identifier   []Identifier          `json:"identifier,omitempty"`
	Active       bool                  `json:"active,omitempty"`
	Type         []CodeableConcept     `json:"type,omitempty"`
	Name         string                `json:"name,omitempty"`
	Alias        []string              `json:"alias,omitempty"`
	Telecom      []ContactPoint        `json:"telecom,omitempty"`
	Address      []Address             `json:"address,omitempty"`
	PartOf       *Reference            `json:"partOf,omitempty"`
	Contact      []OrganizationContact `json:"contact,omitempty"`
	Endpoint     []Reference           `json:"endpoint,omitempty"`
}

// OrganizationContact represents a contact for an organization.
type OrganizationContact struct {
	Purpose *CodeableConcept `json:"purpose,omitempty"`
	Name    *HumanName       `json:"name,omitempty"`
	Telecom []ContactPoint   `json:"telecom,omitempty"`
	Address *Address         `json:"address,omitempty"`
}

// GetNCPDPID returns the organization's NCPDP ID (for pharmacies).
func (o *Organization) GetNCPDPID() string {
	for _, id := range o.Identifier {
		if id.System == SystemNCPDP ||
			id.System == "http://terminology.hl7.org/CodeSystem/NCPDPProviderIdentificationNumber" {
			return id.Value
		}
	}
	return ""
}

// GetNPI returns the organization's NPI.
func (o *Organization) GetNPI() string {
	for _, id := range o.Identifier {
		if id.System == SystemNPI {
			return id.Value
		}
	}
	return ""
}
