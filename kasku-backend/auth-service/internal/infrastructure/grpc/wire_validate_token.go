package grpc

import (
	"google.golang.org/protobuf/encoding/protowire"
)

// validateTokenRequest mirror dari auth.v1.ValidateTokenRequest.
type validateTokenRequest struct {
	AccessToken string // field 1
}

// validateTokenResponse mirror dari auth.v1.ValidateTokenResponse.
type validateTokenResponse struct {
	UserID           string // field 1
	Email            string // field 2
	TenantSchema     string // field 3
	SubscriptionTier string // field 4
	JTI              string // field 5
	ExpiresAtUnix    int64  // field 6
}

func decodeValidateTokenRequest(b []byte) (*validateTokenRequest, error) {
	req := &validateTokenRequest{}
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
			req.AccessToken = s
			b = b[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			b = b[n:]
		}
	}
	return req, nil
}

func encodeValidateTokenResponse(resp *validateTokenResponse) []byte {
	var b []byte
	if resp.UserID != "" {
		b = protowire.AppendTag(b, 1, protowire.BytesType)
		b = protowire.AppendString(b, resp.UserID)
	}
	if resp.Email != "" {
		b = protowire.AppendTag(b, 2, protowire.BytesType)
		b = protowire.AppendString(b, resp.Email)
	}
	if resp.TenantSchema != "" {
		b = protowire.AppendTag(b, 3, protowire.BytesType)
		b = protowire.AppendString(b, resp.TenantSchema)
	}
	if resp.SubscriptionTier != "" {
		b = protowire.AppendTag(b, 4, protowire.BytesType)
		b = protowire.AppendString(b, resp.SubscriptionTier)
	}
	if resp.JTI != "" {
		b = protowire.AppendTag(b, 5, protowire.BytesType)
		b = protowire.AppendString(b, resp.JTI)
	}
	if resp.ExpiresAtUnix != 0 {
		b = protowire.AppendTag(b, 6, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(resp.ExpiresAtUnix))
	}
	return b
}
