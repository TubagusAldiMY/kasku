package grpc

import (
	"google.golang.org/protobuf/encoding/protowire"
)

// getUserByIDRequest mirror auth.v1.GetUserByIDRequest.
type getUserByIDRequest struct {
	UserID string // field 1
}

// getUserByEmailRequest mirror auth.v1.GetUserByEmailRequest.
type getUserByEmailRequest struct {
	Email string // field 1
}

// userResponse mirror auth.v1.UserResponse.
type userResponse struct {
	ID            string // field 1
	Email         string // field 2
	Username      string // field 3
	IsActive      bool   // field 4
	EmailVerified bool   // field 5
	CreatedAtUnix int64  // field 6
}

func decodeStringField1(b []byte) (string, error) {
	var out string
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return "", protowire.ParseError(n)
		}
		b = b[n:]
		switch {
		case num == 1 && typ == protowire.BytesType:
			s, n := protowire.ConsumeString(b)
			if n < 0 {
				return "", protowire.ParseError(n)
			}
			out = s
			b = b[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return "", protowire.ParseError(n)
			}
			b = b[n:]
		}
	}
	return out, nil
}

func decodeGetUserByIDRequest(b []byte) (*getUserByIDRequest, error) {
	s, err := decodeStringField1(b)
	if err != nil {
		return nil, err
	}
	return &getUserByIDRequest{UserID: s}, nil
}

func decodeGetUserByEmailRequest(b []byte) (*getUserByEmailRequest, error) {
	s, err := decodeStringField1(b)
	if err != nil {
		return nil, err
	}
	return &getUserByEmailRequest{Email: s}, nil
}

func encodeUserResponse(resp *userResponse) []byte {
	var b []byte
	if resp.ID != "" {
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendString(b, resp.ID)
	}
	if resp.Email != "" {
		b = protowire.AppendTag(b, 2, protowire.BytesType)
		b = protowire.AppendString(b, resp.Email)
	}
	if resp.Username != "" {
		b = protowire.AppendTag(b, 3, protowire.BytesType)
		b = protowire.AppendString(b, resp.Username)
	}
	if resp.IsActive {
		b = protowire.AppendTag(b, 4, protowire.VarintType)
		b = protowire.AppendVarint(b, 1)
	}
	if resp.EmailVerified {
		b = protowire.AppendTag(b, 5, protowire.VarintType)
		b = protowire.AppendVarint(b, 1)
	}
	if resp.CreatedAtUnix != 0 {
		b = protowire.AppendTag(b, 6, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(resp.CreatedAtUnix))
	}
	return b
}
