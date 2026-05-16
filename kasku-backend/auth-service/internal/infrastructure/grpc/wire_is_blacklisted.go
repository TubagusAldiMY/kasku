package grpc

import (
	"google.golang.org/protobuf/encoding/protowire"
)

// isTokenBlacklistedRequest mirror auth.v1.IsTokenBlacklistedRequest.
type isTokenBlacklistedRequest struct {
	JTI string // field 1
}

// isTokenBlacklistedResponse mirror auth.v1.IsTokenBlacklistedResponse.
type isTokenBlacklistedResponse struct {
	Blacklisted bool // field 1
}

func decodeIsTokenBlacklistedRequest(b []byte) (*isTokenBlacklistedRequest, error) {
	s, err := decodeStringField1(b)
	if err != nil {
		return nil, err
	}
	return &isTokenBlacklistedRequest{JTI: s}, nil
}

func encodeIsTokenBlacklistedResponse(resp *isTokenBlacklistedResponse) []byte {
	var b []byte
	if resp.Blacklisted {
		b = protowire.AppendTag(b, 1, protowire.VarintType)
		b = protowire.AppendVarint(b, 1)
	}
	return b
}
