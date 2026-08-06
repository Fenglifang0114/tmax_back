package svc

import (
	"os"
	"testing"
)

func TestNewDbScaleConn(t *testing.T) {
	type args struct {
		dbName string
	}
	tests := []struct {
		name    string
		args    args
		want    *DbScaleConn
		wantErr bool
	}{
		{name: "new DbScaleConn", args: args{dbName: "test.db"}, want: &DbScaleConn{dbName: "test.db"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDbScaleConn(tt.args.dbName)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDbScaleConn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil || tt.want == nil || got.dbName != tt.want.dbName {
				t.Errorf("NewDbScaleConn() = %v, want %v", got, tt.want)
			}
		})
	}
}

// func TestDbScaleConn_GetScaleConnList(t *testing.T) {
// 	tests := []struct {
// 		name    string
// 		d       *DbScaleConn
// 		want    []ScaleConnMedia
// 		wantErr bool
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := tt.d.GetScaleConnList()
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("DbScaleConn.GetScaleConnList() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("DbScaleConn.GetScaleConnList() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

func TestDbScaleConn_InsertScaleConn(t *testing.T) {
	os.Remove("test.db")
	db, _ := NewDbScaleConn("test.db")
	type args struct {
		conn ScaleConnMedia
	}
	tests := []struct {
		name    string
		d       *DbScaleConn
		args    args
		wantErr bool
	}{
		{name: "insert scale connection", d: db, args: args{conn: ScaleConnMedia{ScaleModel: "QTP", ScaleSn: "1234", TMedia: MEDIA_COM, MediaConf: MediaConf{
			Type:          MEDIA_COM,
			MediaInfoJson: `"DevPath":"COM6", "Baud": 9600, "DataBits": 8, "StopBits": 1, "Parity": 0`,
		}}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.d.InsertScaleConn(tt.args.conn); (err != nil) != tt.wantErr {
				t.Errorf("DbScaleConn.InsertScaleConn() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDbScaleConn_UpdateScaleConn(t *testing.T) {
	db, _ := NewDbScaleConn("test.db")
	type args struct {
		conn ScaleConnMedia
	}
	tests := []struct {
		name    string
		d       *DbScaleConn
		args    args
		wantErr bool
	}{
		{name: "update scale connection", d: db, args: args{conn: ScaleConnMedia{ScaleModel: "QTP", ScaleSn: "1234", TMedia: MEDIA_COM, MediaConf: MediaConf{
			Type:          MEDIA_COM,
			MediaInfoJson: `"DevPath":"COM6", "Baud": 115200, "DataBits": 8, "StopBits": 1, "Parity": 0`,
		}}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.d.UpdateScaleConn(tt.args.conn); (err != nil) != tt.wantErr {
				t.Errorf("DbScaleConn.UpdateScaleConn() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDbScaleConn_DeleteScaleConn(t *testing.T) {
	db, _ := NewDbScaleConn("test.db")
	type args struct {
		inConn ScaleConnMedia
	}
	tests := []struct {
		name    string
		d       *DbScaleConn
		args    args
		wantErr bool
	}{
		{name: "delete scale connection", d: db, args: args{inConn: ScaleConnMedia{ScaleModel: "QTP", ScaleSn: "1234", TMedia: MEDIA_COM, MediaConf: MediaConf{
			Type:          MEDIA_COM,
			MediaInfoJson: `"DevPath":"COM6", "Baud": 9600, "DataBits": 8, "StopBits": 1, "Parity": 0`,
		}}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.d.DeleteScaleConn(tt.args.inConn); (err != nil) != tt.wantErr {
				t.Errorf("DbScaleConn.DeleteScaleConn() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
