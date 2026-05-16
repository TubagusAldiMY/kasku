package grpc

import (
	"fmt"

	"google.golang.org/grpc/encoding"
)

func init() {
	// Daftarkan rawServerCodec sebagai codec "proto" sebelum server apapun dibuat.
	// Ini menggantikan codec proto default sehingga server bisa menerima raw bytes
	// dari api-gateway yang menggunakan grpc.ForceCodec(rawBytesCodec{}) di sisi client.
	encoding.RegisterCodec(rawServerCodec{})
}

// rawServerCodec adalah server-side codec yang meneruskan raw bytes tanpa transformasi.
// Nama "proto" diperlukan agar codec ini menggantikan default proto codec pada server.
// Simetris dengan rawBytesCodec di api-gateway/proto/billing/v1/billing_grpc.go.
type rawServerCodec struct{}

func (rawServerCodec) Name() string { return "proto" }

func (rawServerCodec) Marshal(v any) ([]byte, error) {
	m, ok := v.(*rawBytesMsg)
	if !ok {
		return nil, fmt.Errorf("rawServerCodec: tipe tidak didukung %T", v)
	}
	return m.data, nil
}

func (rawServerCodec) Unmarshal(data []byte, v any) error {
	m, ok := v.(*rawBytesMsg)
	if !ok {
		return fmt.Errorf("rawServerCodec: tipe tidak didukung %T", v)
	}
	m.data = data
	return nil
}

// rawBytesMsg adalah container untuk raw protobuf wire-format bytes.
type rawBytesMsg struct {
	data []byte
}
