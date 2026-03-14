package r5

import (
	"testing"
)

func BenchmarkPractitioner_GetFullName(b *testing.B) {
	p := &Practitioner{
		Name: []HumanName{
			{
				Use:    "official",
				Prefix: []string{"Dr.", "Prof."},
				Given:  []string{"Jane", "Mary"},
				Family: "Smith",
				Suffix: []string{"MD", "PhD"},
			},
		},
	}

	for i := 0; i < b.N; i++ {
		_ = p.GetFullName()
	}
}

func BenchmarkPatient_GetFullName(b *testing.B) {
	p := &Patient{
		Name: []HumanName{
			{
				Use:    "official",
				Given:  []string{"John", "Jacob"},
				Family: "Jingleheimer-Schmidt",
			},
		},
	}

	for i := 0; i < b.N; i++ {
		_ = p.GetFullName()
	}
}

func TestPractitioner_GetFullName(t *testing.T) {
	tests := []struct {
		name string
		p    *Practitioner
		want string
	}{
		{
			name: "full name with all parts",
			p: &Practitioner{
				Name: []HumanName{
					{
						Use:    "official",
						Prefix: []string{"Dr."},
						Given:  []string{"Jane"},
						Family: "Smith",
						Suffix: []string{"MD"},
					},
				},
			},
			want: "Dr. Jane Smith, MD",
		},
		{
			name: "no name",
			p:    &Practitioner{},
			want: "",
		},
		{
			name: "text name",
			p: &Practitioner{
				Name: []HumanName{
					{
						Use:  "official",
						Text: "Dr. Jane Smith MD",
					},
				},
			},
			want: "Dr. Jane Smith MD",
		},
		{
			name: "multiple given names",
			p: &Practitioner{
				Name: []HumanName{
					{
						Use:   "official",
						Given: []string{"Jane", "Mary"},
					},
				},
			},
			want: "Jane Mary",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.GetFullName(); got != tt.want {
				t.Errorf("Practitioner.GetFullName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPatient_GetFullName(t *testing.T) {
	tests := []struct {
		name string
		p    *Patient
		want string
	}{
		{
			name: "full name",
			p: &Patient{
				Name: []HumanName{
					{
						Use:    "official",
						Given:  []string{"John", "Jacob"},
						Family: "Smith",
					},
				},
			},
			want: "John Jacob Smith",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.GetFullName(); got != tt.want {
				t.Errorf("Patient.GetFullName() = %v, want %v", got, tt.want)
			}
		})
	}
}
