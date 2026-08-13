package svc

import (
	"testing"
	m "tmaxsrv/comm"
)

func TestParseSCP23(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantVal     string
		wantUnit    string
		wantOverload bool
	}{
		{
			name:     "Positive sign +",
			input:    "+  150.0kg\r\n",
			wantVal:  "150.0",
			wantUnit: "kg",
		},
		{
			name:     "Positive sign 1",
			input:    "1   50.0kg\r\n",
			wantVal:  "50.0",
			wantUnit: "kg",
		},
		{
			name:     "Negative sign -",
			input:    "-  150.0kg\r\n",
			wantVal:  "-150.0",
			wantUnit: "kg",
		},
		{
			name:     "Overload O L",
			input:    "O L      kg\r\n",
			wantVal:  "-OL-",
			wantUnit: "",
		},
		{
			name:     "Underload U L",
			input:    "U L      kg\r\n",
			wantVal:  "-UL-",
			wantUnit: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := DispatchProtocolParser("SCP-23", 1, []byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.MsgType != m.WEIGHT_DATA {
				t.Fatalf("expected MsgType WEIGHT_DATA, got %v", resp.MsgType)
			}
			msg, ok := resp.MsgBody.(WeightMsg)
			if !ok {
				t.Fatalf("expected WeightMsg in MsgBody, got %T", resp.MsgBody)
			}
			if msg.WeightVal != tt.wantVal {
				t.Errorf("WeightVal = %q, want %q", msg.WeightVal, tt.wantVal)
			}
			if msg.WeightUnit != tt.wantUnit {
				t.Errorf("WeightUnit = %q, want %q", msg.WeightUnit, tt.wantUnit)
			}
		})
	}
}

func TestDispatchFallback(t *testing.T) {
	tests := []struct {
		name             string
		selectedProtocol string
		input            string
		wantVal          string
		wantUnit         string
	}{
		{
			name:             "User selected SCP-23 but scale sends ST,NT,+   12.345kg (Tier 2 Standard Fallback)",
			selectedProtocol: "SCP-23",
			input:            "ST,NT,+   12.345kg\r\n",
			wantVal:          "12.345",
			wantUnit:         "kg",
		},
		{
			name:             "User selected SCP-01 but scale sends simple + 99.8g (Tier 3 Flexible Fallback)",
			selectedProtocol: "SCP-01",
			input:            "+ 99.8g\r\n",
			wantVal:          "99.8",
			wantUnit:         "g",
		},
		{
			name:             "User selected non-existent SCP-99 but scale sends ST,GS, 88.5g",
			selectedProtocol: "SCP-99",
			input:            "ST,GS, 88.5g\r\n",
			wantVal:          "88.5",
			wantUnit:         "g",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := DispatchProtocolParser(tt.selectedProtocol, 1, []byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error during fallback: %v", err)
			}
			if resp.MsgType != m.WEIGHT_DATA {
				t.Fatalf("expected MsgType WEIGHT_DATA, got %v", resp.MsgType)
			}
			msg, ok := resp.MsgBody.(WeightMsg)
			if !ok {
				t.Fatalf("expected WeightMsg in MsgBody, got %T", resp.MsgBody)
			}
			if msg.WeightVal != tt.wantVal {
				t.Errorf("WeightVal = %q, want %q", msg.WeightVal, tt.wantVal)
			}
			if msg.WeightUnit != tt.wantUnit {
				t.Errorf("WeightUnit = %q, want %q", msg.WeightUnit, tt.wantUnit)
			}
		})
	}
}

