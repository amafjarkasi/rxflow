package r5

import (
	"testing"
)

func TestMedicationRequest_GetQuantity(t *testing.T) {
	tests := []struct {
		name      string
		m         *MedicationRequest
		wantVal   float64
		wantUnit  string
	}{
		{
			name: "quantity with value and unit",
			m: &MedicationRequest{
				DispenseRequest: &DispenseRequest{
					Quantity: &Quantity{
						Value: 30.0,
						Unit:  "Tablet",
					},
				},
			},
			wantVal:  30.0,
			wantUnit: "Tablet",
		},
		{
			name: "nil dispense request",
			m: &MedicationRequest{
				DispenseRequest: nil,
			},
			wantVal:  0,
			wantUnit: "",
		},
		{
			name: "nil quantity in dispense request",
			m: &MedicationRequest{
				DispenseRequest: &DispenseRequest{
					Quantity: nil,
				},
			},
			wantVal:  0,
			wantUnit: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVal, gotUnit := tt.m.GetQuantity()
			if gotVal != tt.wantVal {
				t.Errorf("MedicationRequest.GetQuantity() value = %v, want %v", gotVal, tt.wantVal)
			}
			if gotUnit != tt.wantUnit {
				t.Errorf("MedicationRequest.GetQuantity() unit = %v, want %v", gotUnit, tt.wantUnit)
			}
		})
	}
}

func TestMedicationRequest_JSON(t *testing.T) {
	m := &MedicationRequest{
		ResourceType: "MedicationRequest",
		ID:           "test-id",
		Status:       StatusActive,
		Intent:       IntentOrder,
	}

	data, err := m.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() failed: %v", err)
	}

	var m2 MedicationRequest
	if err := m2.FromJSON(data); err != nil {
		t.Fatalf("FromJSON() failed: %v", err)
	}

	if m2.ID != m.ID || m2.Status != m.Status || m2.Intent != m.Intent {
		t.Errorf("JSON roundtrip failed: got %+v, want %+v", m2, m)
	}
}

func TestMedicationRequest_GetPatientID(t *testing.T) {
	tests := []struct {
		name string
		m    *MedicationRequest
		want string
	}{
		{
			name: "patient reference",
			m: &MedicationRequest{
				Subject: Reference{Reference: "Patient/123"},
			},
			want: "123",
		},
		{
			name: "uuid reference",
			m: &MedicationRequest{
				Subject: Reference{Reference: "urn:uuid:456"},
			},
			want: "456",
		},
		{
			name: "empty reference",
			m:    &MedicationRequest{},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.GetPatientID(); got != tt.want {
				t.Errorf("MedicationRequest.GetPatientID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMedicationRequest_GetPrescriberNPI(t *testing.T) {
	tests := []struct {
		name string
		m    *MedicationRequest
		want string
	}{
		{
			name: "npi present",
			m: &MedicationRequest{
				Requester: &Reference{
					Identifier: &Identifier{
						System: SystemNPI,
						Value:  "1234567890",
					},
				},
			},
			want: "1234567890",
		},
		{
			name: "wrong system",
			m: &MedicationRequest{
				Requester: &Reference{
					Identifier: &Identifier{
						System: "other",
						Value:  "1234567890",
					},
				},
			},
			want: "",
		},
		{
			name: "nil requester",
			m:    &MedicationRequest{},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.GetPrescriberNPI(); got != tt.want {
				t.Errorf("MedicationRequest.GetPrescriberNPI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMedicationRequest_GetMedicationCode(t *testing.T) {
	tests := []struct {
		name       string
		m          *MedicationRequest
		wantSystem string
		wantCode   string
	}{
		{
			name: "prefers rxnorm",
			m: &MedicationRequest{
				Medication: CodeableReference{
					Concept: &CodeableConcept{
						Coding: []Coding{
							{System: "http://hl7.org/fhir/sid/ndc", Code: "ndc123"},
							{System: "http://www.nlm.nih.gov/research/umls/rxnorm", Code: "rxnorm456"},
						},
					},
				},
			},
			wantSystem: "rxnorm",
			wantCode:   "rxnorm456",
		},
		{
			name: "falls back to ndc",
			m: &MedicationRequest{
				Medication: CodeableReference{
					Concept: &CodeableConcept{
						Coding: []Coding{
							{System: "http://hl7.org/fhir/sid/ndc", Code: "ndc123"},
						},
					},
				},
			},
			wantSystem: "ndc",
			wantCode:   "ndc123",
		},
		{
			name: "returns first available if not rxnorm/ndc",
			m: &MedicationRequest{
				Medication: CodeableReference{
					Concept: &CodeableConcept{
						Coding: []Coding{
							{System: "other", Code: "other789"},
						},
					},
				},
			},
			wantSystem: "other",
			wantCode:   "other789",
		},
		{
			name: "nil concept",
			m:    &MedicationRequest{},
			wantSystem: "",
			wantCode:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSystem, gotCode := tt.m.GetMedicationCode()
			if gotSystem != tt.wantSystem || gotCode != tt.wantCode {
				t.Errorf("MedicationRequest.GetMedicationCode() = (%v, %v), want (%v, %v)", gotSystem, gotCode, tt.wantSystem, tt.wantCode)
			}
		})
	}
}

func TestMedicationRequest_GetPrescriberDEA(t *testing.T) {
	tests := []struct {
		name string
		m    *MedicationRequest
		want string
	}{
		{
			name: "dea present",
			m: &MedicationRequest{
				Requester: &Reference{
					Identifier: &Identifier{
						System: SystemDEA,
						Value:  "AB1234567",
					},
				},
			},
			want: "AB1234567",
		},
		{
			name: "wrong system",
			m: &MedicationRequest{
				Requester: &Reference{
					Identifier: &Identifier{
						System: SystemNPI,
						Value:  "1234567890",
					},
				},
			},
			want: "",
		},
		{
			name: "nil requester",
			m:    &MedicationRequest{},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.GetPrescriberDEA(); got != tt.want {
				t.Errorf("MedicationRequest.GetPrescriberDEA() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMedicationRequest_GetNDC(t *testing.T) {
	tests := []struct {
		name string
		m    *MedicationRequest
		want string
	}{
		{
			name: "ndc present",
			m: &MedicationRequest{
				Medication: CodeableReference{
					Concept: &CodeableConcept{
						Coding: []Coding{
							{System: "http://hl7.org/fhir/sid/ndc", Code: "12345-678-90"},
						},
					},
				},
			},
			want: "12345-678-90",
		},
		{
			name: "ndc missing",
			m: &MedicationRequest{
				Medication: CodeableReference{
					Concept: &CodeableConcept{
						Coding: []Coding{
							{System: "http://www.nlm.nih.gov/research/umls/rxnorm", Code: "456"},
						},
					},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.GetNDC(); got != tt.want {
				t.Errorf("MedicationRequest.GetNDC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMedicationRequest_GetRxNorm(t *testing.T) {
	tests := []struct {
		name string
		m    *MedicationRequest
		want string
	}{
		{
			name: "rxnorm present",
			m: &MedicationRequest{
				Medication: CodeableReference{
					Concept: &CodeableConcept{
						Coding: []Coding{
							{System: "http://www.nlm.nih.gov/research/umls/rxnorm", Code: "456"},
						},
					},
				},
			},
			want: "456",
		},
		{
			name: "rxnorm missing",
			m: &MedicationRequest{
				Medication: CodeableReference{
					Concept: &CodeableConcept{
						Coding: []Coding{
							{System: "http://hl7.org/fhir/sid/ndc", Code: "12345"},
						},
					},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.GetRxNorm(); got != tt.want {
				t.Errorf("MedicationRequest.GetRxNorm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMedicationRequest_GetMedicationDisplay(t *testing.T) {
	tests := []struct {
		name string
		m    *MedicationRequest
		want string
	}{
		{
			name: "prefers concept text",
			m: &MedicationRequest{
				Medication: CodeableReference{
					Concept: &CodeableConcept{
						Text: "Amoxicillin 500mg",
						Coding: []Coding{
							{Display: "Amoxicillin"},
						},
					},
				},
			},
			want: "Amoxicillin 500mg",
		},
		{
			name: "falls back to coding display",
			m: &MedicationRequest{
				Medication: CodeableReference{
					Concept: &CodeableConcept{
						Coding: []Coding{
							{Display: "Amoxicillin"},
						},
					},
				},
			},
			want: "Amoxicillin",
		},
		{
			name: "nil concept",
			m:    &MedicationRequest{},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.GetMedicationDisplay(); got != tt.want {
				t.Errorf("MedicationRequest.GetMedicationDisplay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMedicationRequest_GetDaysSupply(t *testing.T) {
	tests := []struct {
		name string
		m    *MedicationRequest
		want int
	}{
		{
			name: "days supply set",
			m: &MedicationRequest{
				DispenseRequest: &DispenseRequest{
					ExpectedSupplyDuration: &Duration{
						Value: 90,
					},
				},
			},
			want: 90,
		},
		{
			name: "nil dispense request",
			m:    &MedicationRequest{},
			want: 0,
		},
		{
			name: "nil expected supply duration",
			m: &MedicationRequest{
				DispenseRequest: &DispenseRequest{},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.GetDaysSupply(); got != tt.want {
				t.Errorf("MedicationRequest.GetDaysSupply() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMedicationRequest_GetRefillsAllowed(t *testing.T) {
	tests := []struct {
		name string
		m    *MedicationRequest
		want int
	}{
		{
			name: "refills allowed set",
			m: &MedicationRequest{
				DispenseRequest: &DispenseRequest{
					NumberOfRepeatsAllowed: 3,
				},
			},
			want: 3,
		},
		{
			name: "nil dispense request",
			m:    &MedicationRequest{},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.GetRefillsAllowed(); got != tt.want {
				t.Errorf("MedicationRequest.GetRefillsAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMedicationRequest_IsSubstitutionAllowed(t *testing.T) {
	tr := true
	fa := false
	tests := []struct {
		name string
		m    *MedicationRequest
		want bool
	}{
		{
			name: "substitution allowed explicitly true",
			m: &MedicationRequest{
				Substitution: &Substitution{
					AllowedBoolean: &tr,
				},
			},
			want: true,
		},
		{
			name: "substitution allowed explicitly false",
			m: &MedicationRequest{
				Substitution: &Substitution{
					AllowedBoolean: &fa,
				},
			},
			want: false,
		},
		{
			name: "substitution nil (defaults to true)",
			m:    &MedicationRequest{},
			want: true,
		},
		{
			name: "substitution allowed boolean nil (defaults to true)",
			m: &MedicationRequest{
				Substitution: &Substitution{},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.IsSubstitutionAllowed(); got != tt.want {
				t.Errorf("MedicationRequest.IsSubstitutionAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMedicationRequest_GetSigText(t *testing.T) {
	tests := []struct {
		name string
		m    *MedicationRequest
		want string
	}{
		{
			name: "rendered dosage instruction takes precedence",
			m: &MedicationRequest{
				RenderedDosageInstruction: "Take 1 tablet daily",
				DosageInstruction: []Dosage{
					{Text: "Take 1 tab daily"},
				},
			},
			want: "Take 1 tablet daily",
		},
		{
			name: "fallback to dosage instruction text",
			m: &MedicationRequest{
				DosageInstruction: []Dosage{
					{Text: "Take 1 tab daily"},
				},
			},
			want: "Take 1 tab daily",
		},
		{
			name: "no sig text",
			m:    &MedicationRequest{},
			want: "",
		},
		{
			name: "empty dosage instruction",
			m: &MedicationRequest{
				DosageInstruction: []Dosage{},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.GetSigText(); got != tt.want {
				t.Errorf("MedicationRequest.GetSigText() = %v, want %v", got, tt.want)
			}
		})
	}
}
