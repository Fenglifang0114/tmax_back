package svc

import (
	"reflect"
	"testing"
)

func TestNewDbScaleRec(t *testing.T) {
	type args struct {
		dbName string
	}
	tests := []struct {
		name    string
		args    args
		want    *DbScaleRec
		wantErr bool
	}{
		{name: "new DbScaleRec", args: args{dbName: "testrec.db"}, want: &DbScaleRec{dbName: "testrec.db"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDbScaleRec(tt.args.dbName)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDbScaleRec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil || tt.want == nil || got.dbName != tt.want.dbName {
				t.Errorf("NewDbScaleRec() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDbScaleRec_InsertScaleRec(t *testing.T) {
	db, _ := NewDbScaleRec("testrec.db")
	type args struct {
		rec ScaleRec
	}
	tests := []struct {
		name    string
		d       *DbScaleRec
		args    args
		wantErr bool
	}{
		{name: "InsertScaleRec", d: db, args: args{rec: ScaleRec{ScaleModel: "QTP", ScaleSn: "1234", ProductName: "Apple", Weight: "1.230", Price: "3.25"}}, wantErr: false},
		{name: "InsertScaleRec", d: db, args: args{rec: ScaleRec{ScaleModel: "QTP", ScaleSn: "1234", ProductName: "Banana", Weight: "1.230", Price: "3.25"}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.d.InsertScaleRec(tt.args.rec); (err != nil) != tt.wantErr {
				t.Errorf("DbScaleRec.InsertScaleRec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDbScaleRec_UpdateScaleRec(t *testing.T) {
	db, _ := NewDbScaleRec("testrec.db")
	type args struct {
		rec ScaleRec
	}
	tests := []struct {
		name    string
		d       *DbScaleRec
		args    args
		wantErr bool
	}{
		{name: "UpdateScaleRec", d: db, args: args{rec: ScaleRec{RecId: 1, ScaleModel: "QTP", ScaleSn: "1234", ProductName: "Banana"}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.d.UpdateScaleRec(tt.args.rec); (err != nil) != tt.wantErr {
				t.Errorf("DbScaleRec.UpdateScaleRec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDbScaleRec_DeleteScaleRec(t *testing.T) {
	db, _ := NewDbScaleRec("testrec.db")

	type args struct {
		RecId uint
	}
	tests := []struct {
		name    string
		d       *DbScaleRec
		args    args
		wantErr bool
	}{
		{name: "DeleteScaleRec", d: db, args: args{RecId: 2}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.d.DeleteScaleRec(tt.args.RecId); (err != nil) != tt.wantErr {
				t.Errorf("DbScaleRec.DeleteScaleRec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDbScaleRec_GetScaleRecs(t *testing.T) {
	db, _ := NewDbScaleRec("testrec.db")
	type args struct {
		model    string
		sn       string
		start    int
		quantity int
	}
	tests := []struct {
		name    string
		d       *DbScaleRec
		args    args
		want    []ScaleRec
		wantErr bool
	}{
		{name: "GetScaleRecs", d: db, args: args{model: "QTP", sn: "1234", start: 0, quantity: 1}, want: []ScaleRec{}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.d.GetScaleRecs(tt.args.model, tt.args.sn, tt.args.start, tt.args.quantity)
			if (err != nil) != tt.wantErr {
				t.Errorf("DbScaleRec.GetScaleRecs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != 1 {
				t.Errorf("DbScaleRec.GetScaleRecs() return quantitiy is: %v not as expected %v", len(got), tt.args.quantity)
			}
		})
	}
}

func TestDbScaleRec_GetScaleRecsList(t *testing.T) {
	db, _ := NewDbScaleRec("testrec.db")
	type args struct {
		model string
		sn    string
	}
	tests := []struct {
		name    string
		d       *DbScaleRec
		args    args
		want    []ScaleRec
		wantErr bool
	}{
		{name: "GetScaleRecsList", d: db, args: args{model: "QTP", sn: "1234"}, want: []ScaleRec{
			{RecId: 1, ScaleModel: "QTP", ScaleSn: "1234", ProductName: "Banana", Weight: "1.230", Price: "3.25"},
			{RecId: 2, ScaleModel: "QTP", ScaleSn: "1234", ProductName: "Apple", Weight: "1.230", Price: "3.25"},
			{RecId: 3, ScaleModel: "QTP", ScaleSn: "1234", ProductName: "Banana", Weight: "1.230", Price: "3.25"},
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.d.GetScaleRecsList(tt.args.model, tt.args.sn, "", "", "", "", "")
			if (err != nil) != tt.wantErr {
				t.Errorf("DbScaleRec.GetScaleRecsList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// for i := 0; i < len(got); i++ {
			for i := 0; i < 2; i++ {
				if !compare(&got[i], &tt.want[i]) {
					t.Errorf("DbScaleRec.GetScaleRecsList() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func compare(a, b *ScaleRec) bool {
	a2 := new(ScaleRec)
	*a2 = *a
	a2.RecId = b.RecId
	a2.CreatedAt = b.CreatedAt
	return reflect.DeepEqual(a2, b)
}
