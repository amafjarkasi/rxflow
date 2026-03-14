package script2023011

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestNewRx_Validate(t *testing.T) {
	validNewRx := func() NewRx {
		return NewRx{
			Prescriber: Prescriber{
				Name: Name{LastName: "Smith"},
				Identification: Identification{NPI: "1234567890"},
			},
			Pharmacy: Pharmacy{
				Identification: Identification{NPI: "0987654321"},
			},
			Patient: Patient{
				Name: Name{LastName: "Doe"},
			},
			MedicationPrescribed: MedicationPrescribed{
				Product: Product{
					DrugCoded: DrugCoded{
						ProductCode: ProductCode{Code: "12345-678-90"},
					},
				},
				Sig: Sig{
					SigText: "Take 1 tablet daily",
				},
				Quantity: Quantity{
					Value: "30",
				},
			},
		}
	}

	tests := []struct {
		name    string
		modify  func(*NewRx)
		wantErr bool
		errSub  string
	}{
		{
			name:    "valid NewRx",
			modify:  func(n *NewRx) {},
			wantErr: false,
		},
		{
			name: "missing prescriber last name",
			modify: func(n *NewRx) {
				n.Prescriber.Name.LastName = ""
			},
			wantErr: true,
			errSub:  "prescriber last name is required",
		},
		{
			name: "missing prescriber NPI and DEA",
			modify: func(n *NewRx) {
				n.Prescriber.Identification.NPI = ""
				n.Prescriber.Identification.DEANumber = ""
			},
			wantErr: true,
			errSub:  "prescriber NPI or DEA number is required",
		},
		{
			name: "valid prescriber with DEA only",
			modify: func(n *NewRx) {
				n.Prescriber.Identification.NPI = ""
				n.Prescriber.Identification.DEANumber = "AS1234567"
			},
			wantErr: false,
		},
		{
			name: "missing pharmacy identifiers",
			modify: func(n *NewRx) {
				n.Pharmacy.Identification.NPI = ""
				n.Pharmacy.Identification.NCPDPID = ""
			},
			wantErr: true,
			errSub:  "pharmacy NCPDP ID or NPI is required",
		},
		{
			name: "valid pharmacy with NCPDP ID only",
			modify: func(n *NewRx) {
				n.Pharmacy.Identification.NPI = ""
				n.Pharmacy.Identification.NCPDPID = "1234567"
			},
			wantErr: false,
		},
		{
			name: "missing patient last name",
			modify: func(n *NewRx) {
				n.Patient.Name.LastName = ""
			},
			wantErr: true,
			errSub:  "patient last name is required",
		},
		{
			name: "missing medication product code",
			modify: func(n *NewRx) {
				n.MedicationPrescribed.Product.DrugCoded.ProductCode.Code = ""
			},
			wantErr: true,
			errSub:  "medication product code (NDC) is required",
		},
		{
			name: "missing sig text",
			modify: func(n *NewRx) {
				n.MedicationPrescribed.Sig.SigText = ""
			},
			wantErr: true,
			errSub:  "prescription directions (sig) are required",
		},
		{
			name: "missing quantity",
			modify: func(n *NewRx) {
				n.MedicationPrescribed.Quantity.Value = ""
			},
			wantErr: true,
			errSub:  "quantity is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := validNewRx()
			tt.modify(&n)
			err := n.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errSub) {
				t.Errorf("Validate() error = %v, wantSub %v", err, tt.errSub)
			}
		})
	}
}

func TestNewRx_IsControlledSubstance(t *testing.T) {
	tests := []struct {
		name     string
		newRx    NewRx
		expected bool
	}{
		{
			name: "not controlled",
			newRx: NewRx{
				MedicationPrescribed: MedicationPrescribed{
					ControlledSubstanceSchedule: "",
				},
			},
			expected: false,
		},
		{
			name: "controlled by schedule field",
			newRx: NewRx{
				MedicationPrescribed: MedicationPrescribed{
					ControlledSubstanceSchedule: DEAScheduleII,
				},
			},
			expected: true,
		},
		{
			name: "controlled by drug coded DEA schedule",
			newRx: NewRx{
				MedicationPrescribed: MedicationPrescribed{
					Product: Product{
						DrugCoded: DrugCoded{
							DEASchedule: &DEASchedule{Code: DEAScheduleIII},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "specifically non-controlled",
			newRx: NewRx{
				MedicationPrescribed: MedicationPrescribed{
					ControlledSubstanceSchedule: DEAScheduleNone,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.newRx.IsControlledSubstance(); got != tt.expected {
				t.Errorf("IsControlledSubstance() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewRx_GetDEASchedule(t *testing.T) {
	tests := []struct {
		name     string
		newRx    NewRx
		expected string
	}{
		{
			name: "none",
			newRx: NewRx{
				MedicationPrescribed: MedicationPrescribed{},
			},
			expected: "",
		},
		{
			name: "from schedule field",
			newRx: NewRx{
				MedicationPrescribed: MedicationPrescribed{
					ControlledSubstanceSchedule: DEAScheduleII,
				},
			},
			expected: DEAScheduleII,
		},
		{
			name: "from drug coded",
			newRx: NewRx{
				MedicationPrescribed: MedicationPrescribed{
					Product: Product{
						DrugCoded: DrugCoded{
							DEASchedule: &DEASchedule{Code: DEAScheduleIV},
						},
					},
				},
			},
			expected: DEAScheduleIV,
		},
		{
			name: "schedule field takes precedence",
			newRx: NewRx{
				MedicationPrescribed: MedicationPrescribed{
					ControlledSubstanceSchedule: DEAScheduleII,
					Product: Product{
						DrugCoded: DrugCoded{
							DEASchedule: &DEASchedule{Code: DEAScheduleV},
						},
					},
				},
			},
			expected: DEAScheduleII,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.newRx.GetDEASchedule(); got != tt.expected {
				t.Errorf("GetDEASchedule() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewNewRxMessage(t *testing.T) {
	n := &NewRx{
		Prescriber: Prescriber{Name: Name{LastName: "Smith"}},
	}
	msg := NewNewRxMessage("MSG123", n)

	if msg.Header.MessageID != "MSG123" {
		t.Errorf("expected MessageID MSG123, got %s", msg.Header.MessageID)
	}
	if msg.Body.NewRx != n {
		t.Error("NewRx body not set correctly")
	}
	if msg.Xmlns != NamespaceScript {
		t.Errorf("expected namespace %s, got %s", NamespaceScript, msg.Xmlns)
	}
}

func TestXMLMarshaling(t *testing.T) {
	n := &NewRx{
		Prescriber: Prescriber{
			Name: Name{LastName: "Smith", FirstName: "John"},
		},
	}
	msg := NewNewRxMessage("MSG123", n)

	// Test ToXML
	data, err := msg.ToXML()
	if err != nil {
		t.Fatalf("ToXML failed: %v", err)
	}
	if !strings.Contains(string(data), "<MessageID>MSG123</MessageID>") {
		t.Error("XML output missing MessageID")
	}
	if !strings.Contains(string(data), "<LastName>Smith</LastName>") {
		t.Error("XML output missing LastName")
	}

	// Test FromXML
	msg2, err := FromXML(data)
	if err != nil {
		t.Fatalf("FromXML failed: %v", err)
	}
	if msg2.Header.MessageID != "MSG123" {
		t.Errorf("FromXML: expected MessageID MSG123, got %s", msg2.Header.MessageID)
	}
	if msg2.Body.NewRx.Prescriber.Name.LastName != "Smith" {
		t.Errorf("FromXML: expected LastName Smith, got %s", msg2.Body.NewRx.Prescriber.Name.LastName)
	}

	// Test ToXMLCompact
	compactData, err := msg.ToXMLCompact()
	if err != nil {
		t.Fatalf("ToXMLCompact failed: %v", err)
	}
	if strings.Contains(string(compactData), "  ") {
		t.Error("Compact XML should not contain indentation")
	}

	// Test invalid XML for FromXML
	_, err = FromXML([]byte("<Message><Invalid></Message>"))
	if err == nil {
		t.Error("FromXML should fail on invalid XML")
	}
}

func TestXMLUnmarshalIntoInterface(t *testing.T) {
	xmlData := `
<Message xmlns="http://www.ncpdp.org/schema/SCRIPT" version="2023011">
  <Header>
    <To><Pharmacy><Identification><NPI>123</NPI></Identification></Pharmacy></To>
    <From><Prescriber><Identification><NPI>456</NPI></Identification><Name><LastName>Jones</LastName><FirstName>Bob</FirstName></Name></Prescriber></From>
    <MessageID>999</MessageID>
    <SentTime>20230101120000</SentTime>
  </Header>
  <Body>
    <NewRx>
      <Prescriber><Identification><NPI>456</NPI></Identification><Name><LastName>Jones</LastName><FirstName>Bob</FirstName></Name></Prescriber>
      <Pharmacy><Identification><NPI>123</NPI></Identification></Pharmacy>
      <Patient><Name><LastName>Doe</LastName><FirstName>Jane</FirstName></Name></Patient>
      <MedicationPrescribed>
        <Product><DrugCoded><ProductCode><Code>111</Code></ProductCode></DrugCoded></Product>
        <Quantity><Value>10</Value></Quantity>
        <WrittenDate><Date>20230101</Date></WrittenDate>
        <Sig><SigText>once</SigText></Sig>
        <Refills><Value>0</Value></Refills>
      </MedicationPrescribed>
    </NewRx>
  </Body>
</Message>`

	var msg Message
	err := xml.Unmarshal([]byte(xmlData), &msg)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if msg.Body.NewRx == nil {
		t.Fatal("NewRx body is nil")
	}
	if msg.Body.NewRx.Prescriber.Name.LastName != "Jones" {
		t.Errorf("Expected Jones, got %s", msg.Body.NewRx.Prescriber.Name.LastName)
	}
}
