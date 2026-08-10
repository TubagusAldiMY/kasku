package grpc

import (
	"google.golang.org/protobuf/encoding/protowire"
)

// updateUsernameRequest mirror auth.v1.UpdateUsernameRequest.
type updateUsernameRequest struct {
	UserID   string // field 1
	Username string // field 2
}

// updateUsernameResponse mirror auth.v1.UpdateUsernameResponse.
type updateUsernameResponse struct {
	Username string // field 1
}

// decodeUpdateUsernameRequest membaca dua field string sekaligus.
//
// Field yang tidak dikenal sengaja di-skip, bukan ditolak: itu aturan
// forward-compatibility proto3 — client versi lebih baru yang menambah field
// tetap bisa bicara dengan server versi lama.
func decodeUpdateUsernameRequest(b []byte) (*updateUsernameRequest, error) {
	out := &updateUsernameRequest{}
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
			out.UserID = s
			b = b[n:]
		case num == 2 && typ == protowire.BytesType:
			s, n := protowire.ConsumeString(b)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			out.Username = s
			b = b[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			b = b[n:]
		}
	}
	return out, nil
}

func encodeUpdateUsernameResponse(resp *updateUsernameResponse) []byte {
	var b []byte
	if resp.Username != "" {
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendString(b, resp.Username)
	}
	return b
}
