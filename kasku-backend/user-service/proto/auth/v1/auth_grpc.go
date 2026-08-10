// Package authv1 menyediakan gRPC client untuk auth-service (subset yang dipakai
// user-service saja).
//
// File ini ditulis manual karena protoc tidak tersedia di build environment —
// pola yang sama dengan api-gateway/proto/billing/v1. Wire encoding memakai
// google.golang.org/protobuf/encoding/protowire sehingga kompatibel penuh dengan
// server yang dibangun dari auth-service/proto/auth/v1/auth.proto.
//
// Kontrak (SUMBER KEBENARAN): auth-service/proto/auth/v1/auth.proto.
// Setiap perubahan di sana wajib diikuti di sini.
package authv1

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protowire"
)

// ─── Messages ────────────────────────────────────────────────────────────────

// UpdateUsernameRequest mirror auth.v1.UpdateUsernameRequest.
type UpdateUsernameRequest struct {
	UserID   string // field 1
	Username string // field 2
}

// UpdateUsernameResponse mirror auth.v1.UpdateUsernameResponse.
type UpdateUsernameResponse struct {
	Username string // field 1
}

// ─── Wire encoding ────────────────────────────────────────────────────────────

func encodeUpdateUsernameRequest(req *UpdateUsernameRequest) []byte {
	var b []byte
	if req.UserID != "" {
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendString(b, req.UserID)
	}
	if req.Username != "" {
		b = protowire.AppendTag(b, 2, protowire.BytesType)
		b = protowire.AppendString(b, req.Username)
	}
	return b
}

func decodeUpdateUsernameResponse(b []byte) (*UpdateUsernameResponse, error) {
	resp := &UpdateUsernameResponse{}
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		b = b[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			s, n := protowire.ConsumeString(b)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			resp.Username = s
			b = b[n:]
		default:
			// Field tak dikenal di-skip, bukan ditolak — aturan forward-compatibility
			// proto3 supaya server versi lebih baru tetap bisa bicara dengan client ini.
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			b = b[n:]
		}
	}
	return resp, nil
}

// ─── Raw bytes codec ──────────────────────────────────────────────────────────

// rawBytesMsg adalah container untuk raw protobuf bytes yang dipakai rawBytesCodec.
type rawBytesMsg struct{ data []byte }

// rawBytesCodec meneruskan raw bytes apa adanya. Name() sengaja "proto" supaya
// server (yang memakai codec proto standar) tetap bisa menafsirkannya.
type rawBytesCodec struct{}

func (rawBytesCodec) Name() string { return "proto" }

func (rawBytesCodec) Marshal(v any) ([]byte, error) {
	m, ok := v.(*rawBytesMsg)
	if !ok {
		return nil, fmt.Errorf("rawBytesCodec: expected *rawBytesMsg, got %T", v)
	}
	return m.data, nil
}

func (rawBytesCodec) Unmarshal(data []byte, v any) error {
	m, ok := v.(*rawBytesMsg)
	if !ok {
		return fmt.Errorf("rawBytesCodec: expected *rawBytesMsg, got %T", v)
	}
	m.data = data
	return nil
}

// ─── Client ───────────────────────────────────────────────────────────────────

// AuthInternalClient adalah subset RPC auth.v1.AuthInternal yang dipakai user-service.
type AuthInternalClient interface {
	UpdateUsername(ctx context.Context, req *UpdateUsernameRequest, opts ...grpc.CallOption) (*UpdateUsernameResponse, error)
}

type authInternalClient struct {
	cc grpc.ClientConnInterface
}

// NewAuthInternalClient membuat client baru di atas koneksi gRPC yang sudah ada.
func NewAuthInternalClient(cc grpc.ClientConnInterface) AuthInternalClient {
	return &authInternalClient{cc: cc}
}

func (c *authInternalClient) UpdateUsername(
	ctx context.Context,
	req *UpdateUsernameRequest,
	opts ...grpc.CallOption,
) (*UpdateUsernameResponse, error) {
	reqMsg := &rawBytesMsg{data: encodeUpdateUsernameRequest(req)}
	respMsg := &rawBytesMsg{}

	callOpts := append([]grpc.CallOption{grpc.ForceCodec(rawBytesCodec{})}, opts...)

	if err := c.cc.Invoke(ctx, "/auth.v1.AuthInternal/UpdateUsername", reqMsg, respMsg, callOpts...); err != nil {
		return nil, err
	}
	return decodeUpdateUsernameResponse(respMsg.data)
}
