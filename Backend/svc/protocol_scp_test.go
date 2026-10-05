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

func TestParseSCP24(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantVal    string
		wantUnit   string
		wantStable bool
		wantNet    bool
	}{
		{
			name:       "Stable Gross weight ST,GS",
			input:      "ST,GS,  150.00,kg\r\n",
			wantVal:    "150.00",
			wantUnit:   "kg",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Stable Net weight ST,NT",
			input:      "ST,NT,   25.50, g\r\n",
			wantVal:    "25.50",
			wantUnit:   "g",
			wantStable: true,
			wantNet:    true,
		},
		{
			name:       "Unstable Gross weight with negative sign US,GS",
			input:      "US,GS,-  12.34,lb\r\n",
			wantVal:    "-12.34",
			wantUnit:   "lb",
			wantStable: false,
			wantNet:    false,
		},
		{
			name:       "Unstable Net weight US,NT",
			input:      "US,NT, -  0.50,oz\r\n",
			wantVal:    "-0.50",
			wantUnit:   "oz",
			wantStable: false,
			wantNet:    true,
		},
		{
			name:       "Overload OL",
			input:      "ST,GS,   OL   ,kg\r\n",
			wantVal:    "-OL-",
			wantUnit:   "",
			wantStable: false,
			wantNet:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := DispatchProtocolParser("SCP-24", 1, []byte(tt.input))
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
			if msg.IsStable != tt.wantStable {
				t.Errorf("IsStable = %v, want %v", msg.IsStable, tt.wantStable)
			}
			if msg.IsNet != tt.wantNet {
				t.Errorf("IsNet = %v, want %v", msg.IsNet, tt.wantNet)
			}
		})
	}
}

func TestParseSCP25(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantVal    string
		wantUnit   string
		wantStable bool
		wantNet    bool
	}{
		{
			name:       "Standard weight ST,GS",
			input:      "ST,GS,  150.00,kg\r\n",
			wantVal:    "150.00",
			wantUnit:   "kg",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Quantity count mode PCS: ... PCS",
			input:      "PCS:    1000 PCS\r\n",
			wantVal:    "1000",
			wantUnit:   "PCS",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Single weight / ratio mode U.W. ... KG",
			input:      "U.W.   0.500 KG\r\n",
			wantVal:    "0.500",
			wantUnit:   "KG",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Unstable Net weight US,NT",
			input:      "US,NT,-  12.34, g\r\n",
			wantVal:    "-12.34",
			wantUnit:   "g",
			wantStable: false,
			wantNet:    true,
		},
		{
			name:       "Overload OL",
			input:      "ST,GS,   OL   ,kg\r\n",
			wantVal:    "-OL-",
			wantUnit:   "",
			wantStable: false,
			wantNet:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := DispatchProtocolParser("SCP-25", 1, []byte(tt.input))
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
			if msg.IsStable != tt.wantStable {
				t.Errorf("IsStable = %v, want %v", msg.IsStable, tt.wantStable)
			}
			if msg.IsNet != tt.wantNet {
				t.Errorf("IsNet = %v, want %v", msg.IsNet, tt.wantNet)
			}
		})
	}
}

func TestParseSCP26(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantVal    string
		wantUnit   string
		wantStable bool
		wantNet    bool
	}{
		{
			name:       "Stable Gross weight ST,GS",
			input:      "ST,GS  150.00kg\r\n",
			wantVal:    "150.00",
			wantUnit:   "kg",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Stable Net weight ST,NT",
			input:      "ST,NT   25.50 g\r\n",
			wantVal:    "25.50",
			wantUnit:   "g",
			wantStable: true,
			wantNet:    true,
		},
		{
			name:       "Unstable Gross weight UT,GS (negative)",
			input:      "UT,GS-  12.34lb\r\n",
			wantVal:    "-12.34",
			wantUnit:   "lb",
			wantStable: false,
			wantNet:    false,
		},
		{
			name:       "Unstable Net weight UT,NT (negative)",
			input:      "UT,NT -  0.50oz\r\n",
			wantVal:    "-0.50",
			wantUnit:   "oz",
			wantStable: false,
			wantNet:    true,
		},
		{
			name:       "Overload OL",
			input:      "ST,GS  OL   kg\r\n",
			wantVal:    "-OL-",
			wantUnit:   "",
			wantStable: false,
			wantNet:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := DispatchProtocolParser("SCP-26", 1, []byte(tt.input))
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
			if msg.IsStable != tt.wantStable {
				t.Errorf("IsStable = %v, want %v", msg.IsStable, tt.wantStable)
			}
			if msg.IsNet != tt.wantNet {
				t.Errorf("IsNet = %v, want %v", msg.IsNet, tt.wantNet)
			}
		})
	}
}

func TestParseSCP27(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantVal    string
		wantUnit   string
		wantStable bool
		wantNet    bool
	}{
		{
			name:       "Stable Gross weight ST,GS",
			input:      "ST,GS,  150.00kg\r\n",
			wantVal:    "150.00",
			wantUnit:   "kg",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Unstable Gross weight US,GA with GA header",
			input:      "US,GA,-  12.34lb\r\n",
			wantVal:    "-12.34",
			wantUnit:   "lb",
			wantStable: false,
			wantNet:    false,
		},
		{
			name:       "Unstable Net weight US,NT",
			input:      "US,NT,   25.50kg\r\n",
			wantVal:    "25.50",
			wantUnit:   "kg",
			wantStable: false,
			wantNet:    true,
		},
		{
			name:       "Counting mode PCS without colon",
			input:      "PCS    1000 PCS\r\n",
			wantVal:    "1000",
			wantUnit:   "PCS",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Single weight U.W.",
			input:      "U.W.   0.500 KG\r\n",
			wantVal:    "0.500",
			wantUnit:   "KG",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Overload OL",
			input:      "ST,GS,   OL   ,kg\r\n",
			wantVal:    "-OL-",
			wantUnit:   "",
			wantStable: false,
			wantNet:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := DispatchProtocolParser("SCP-27", 1, []byte(tt.input))
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
			if msg.IsStable != tt.wantStable {
				t.Errorf("IsStable = %v, want %v", msg.IsStable, tt.wantStable)
			}
			if msg.IsNet != tt.wantNet {
				t.Errorf("IsNet = %v, want %v", msg.IsNet, tt.wantNet)
			}
		})
	}
}

func TestParseSCP28(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantVal    string
		wantUnit   string
		wantStable bool
		wantNet    bool
	}{
		{
			name:       "Stable Gross weight ST,GS",
			input:      "ST,GS  150.00kg\r\n",
			wantVal:    "150.00",
			wantUnit:   "kg",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Stable Net weight ST,NT",
			input:      "ST,NT   25.50 g\r\n",
			wantVal:    "25.50",
			wantUnit:   "g",
			wantStable: true,
			wantNet:    true,
		},
		{
			name:       "Unstable Gross weight UT,GS (negative)",
			input:      "UT,GS-  12.34lb\r\n",
			wantVal:    "-12.34",
			wantUnit:   "lb",
			wantStable: false,
			wantNet:    false,
		},
		{
			name:       "Unstable Net weight UT,NT (negative)",
			input:      "UT,NT -  0.50oz\r\n",
			wantVal:    "-0.50",
			wantUnit:   "oz",
			wantStable: false,
			wantNet:    true,
		},
		{
			name:       "Counting mode PCS",
			input:      "PCS    1000 PCS\r\n",
			wantVal:    "1000",
			wantUnit:   "PCS",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Single weight U.W.",
			input:      "U.W.   0.500 KG\r\n",
			wantVal:    "0.500",
			wantUnit:   "KG",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Overload OL",
			input:      "ST,GS  OL   kg\r\n",
			wantVal:    "-OL-",
			wantUnit:   "",
			wantStable: false,
			wantNet:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := DispatchProtocolParser("SCP-28", 1, []byte(tt.input))
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
			if msg.IsStable != tt.wantStable {
				t.Errorf("IsStable = %v, want %v", msg.IsStable, tt.wantStable)
			}
			if msg.IsNet != tt.wantNet {
				t.Errorf("IsNet = %v, want %v", msg.IsNet, tt.wantNet)
			}
		})
	}
}

func TestParseSCP29(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantVal    string
		wantUnit   string
		wantStable bool
		wantNet    bool
	}{
		{
			name:       "Stable Gross weight ST,GS",
			input:      "ST,GS  150.00kg\r\n",
			wantVal:    "150.00",
			wantUnit:   "kg",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Stable Net weight ST,NT",
			input:      "ST,NT   25.50 g\r\n",
			wantVal:    "25.50",
			wantUnit:   "g",
			wantStable: true,
			wantNet:    true,
		},
		{
			name:       "Unstable Gross weight US,GS (negative)",
			input:      "US,GS-  12.34lb\r\n",
			wantVal:    "-12.34",
			wantUnit:   "lb",
			wantStable: false,
			wantNet:    false,
		},
		{
			name:       "Unstable Net weight US,NT (negative)",
			input:      "US,NT -  0.50oz\r\n",
			wantVal:    "-0.50",
			wantUnit:   "oz",
			wantStable: false,
			wantNet:    true,
		},
		{
			name:       "Overload OL",
			input:      "ST,GS  OL   kg\r\n",
			wantVal:    "-OL-",
			wantUnit:   "",
			wantStable: false,
			wantNet:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := DispatchProtocolParser("SCP-29", 1, []byte(tt.input))
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
			if msg.IsStable != tt.wantStable {
				t.Errorf("IsStable = %v, want %v", msg.IsStable, tt.wantStable)
			}
			if msg.IsNet != tt.wantNet {
				t.Errorf("IsNet = %v, want %v", msg.IsNet, tt.wantNet)
			}
		})
	}
}

func TestParseSCP30(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantVal    string
		wantUnit   string
		wantStable bool
		wantNet    bool
	}{
		{
			name:       "Stable Gross weight ST,GS,",
			input:      "ST,GS,  150.00kg\r\n",
			wantVal:    "150.00",
			wantUnit:   "kg",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Stable Net weight ST,NT,",
			input:      "ST,NT,   25.50 g\r\n",
			wantVal:    "25.50",
			wantUnit:   "g",
			wantStable: true,
			wantNet:    true,
		},
		{
			name:       "Unstable Gross weight UT,GS, (negative)",
			input:      "UT,GS,-  12.34lb\r\n",
			wantVal:    "-12.34",
			wantUnit:   "lb",
			wantStable: false,
			wantNet:    false,
		},
		{
			name:       "Unstable Net weight UT,NT, (negative)",
			input:      "UT,NT, -  0.50oz\r\n",
			wantVal:    "-0.50",
			wantUnit:   "oz",
			wantStable: false,
			wantNet:    true,
		},
		{
			name:       "Overload OL",
			input:      "ST,GS,   OL   kg\r\n",
			wantVal:    "-OL-",
			wantUnit:   "",
			wantStable: false,
			wantNet:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := DispatchProtocolParser("SCP-30", 1, []byte(tt.input))
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
			if msg.IsStable != tt.wantStable {
				t.Errorf("IsStable = %v, want %v", msg.IsStable, tt.wantStable)
			}
			if msg.IsNet != tt.wantNet {
				t.Errorf("IsNet = %v, want %v", msg.IsNet, tt.wantNet)
			}
		})
	}
}

func TestParseSCP31(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantVal    string
		wantUnit   string
		wantStable bool
		wantNet    bool
	}{
		{
			name:       "Stable Gross weight ST,GS+",
			input:      "ST,GS+  150.00kg\r\n",
			wantVal:    "150.00",
			wantUnit:   "kg",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Unstable Gross weight US,GA-",
			input:      "US,GA-   12.34lb\r\n",
			wantVal:    "-12.34",
			wantUnit:   "lb",
			wantStable: false,
			wantNet:    false,
		},
		{
			name:       "Counting mode PCS:",
			input:      "PCS:+   1000 PCS\r\n",
			wantVal:    "1000",
			wantUnit:   "PCS",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Single weight UW:",
			input:      "UW:   0.500 G/P\r\n",
			wantVal:    "0.500",
			wantUnit:   "G/P",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Percentage mode PERC:",
			input:      "PERC:+   100.00 %\r\n",
			wantVal:    "100.00",
			wantUnit:   "%",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Catty Tael Qian Minute mode ST,GS,1,8,5,2",
			input:      "ST,GS,1,8,5,2\r\n",
			wantVal:    "1.852",
			wantUnit:   "斤",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Overload OL",
			input:      "ST,GS+  OL   kg\r\n",
			wantVal:    "-OL-",
			wantUnit:   "",
			wantStable: false,
			wantNet:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := DispatchProtocolParser("SCP-31", 1, []byte(tt.input))
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
			if msg.IsStable != tt.wantStable {
				t.Errorf("IsStable = %v, want %v", msg.IsStable, tt.wantStable)
			}
			if msg.IsNet != tt.wantNet {
				t.Errorf("IsNet = %v, want %v", msg.IsNet, tt.wantNet)
			}
		})
	}
}

func TestParseSCP32(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantVal    string
		wantUnit   string
		wantStable bool
		wantNet    bool
	}{
		{
			name:       "Stable Gross weight ST,GS:",
			input:      "ST,GS:  150.00kg\r\n",
			wantVal:    "150.00",
			wantUnit:   "kg",
			wantStable: true,
			wantNet:    false,
		},
		{
			name:       "Stable Net weight ST,NT:",
			input:      "ST,NT:   25.50 g\r\n",
			wantVal:    "25.50",
			wantUnit:   "g",
			wantStable: true,
			wantNet:    true,
		},
		{
			name:       "Unstable Gross weight UT,GS: (negative)",
			input:      "UT,GS:-  12.34lb\r\n",
			wantVal:    "-12.34",
			wantUnit:   "lb",
			wantStable: false,
			wantNet:    false,
		},
		{
			name:       "Unstable Net weight UT,NT: (negative)",
			input:      "UT,NT: -  0.50oz\r\n",
			wantVal:    "-0.50",
			wantUnit:   "oz",
			wantStable: false,
			wantNet:    true,
		},
		{
			name:       "Overload --01--",
			input:      "ST,GS:--01--kg\r\n",
			wantVal:    "-OL-",
			wantUnit:   "",
			wantStable: false,
			wantNet:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := DispatchProtocolParser("SCP-32", 1, []byte(tt.input))
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
			if msg.IsStable != tt.wantStable {
				t.Errorf("IsStable = %v, want %v", msg.IsStable, tt.wantStable)
			}
			if msg.IsNet != tt.wantNet {
				t.Errorf("IsNet = %v, want %v", msg.IsNet, tt.wantNet)
			}
		})
	}
}

func TestParseSCP33(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantVal    string
		wantUnit   string
		wantStable bool
	}{
		{
			name:       "Positive weight +",
			input:      "+  150.00kg\r\n",
			wantVal:    "150.00",
			wantUnit:   "kg",
			wantStable: true,
		},
		{
			name:       "Negative weight -",
			input:      "-   12.34 g\r\n",
			wantVal:    "-12.34",
			wantUnit:   "g",
			wantStable: true,
		},
		{
			name:       "Space signed weight",
			input:      "   100.50lb\r\n",
			wantVal:    "100.50",
			wantUnit:   "lb",
			wantStable: true,
		},
		{
			name:       "Overload OL",
			input:      "  OL   kg\r\n",
			wantVal:    "-OL-",
			wantUnit:   "",
			wantStable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := DispatchProtocolParser("SCP-33", 1, []byte(tt.input))
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
			if msg.IsStable != tt.wantStable {
				t.Errorf("IsStable = %v, want %v", msg.IsStable, tt.wantStable)
			}
		})
	}
}











