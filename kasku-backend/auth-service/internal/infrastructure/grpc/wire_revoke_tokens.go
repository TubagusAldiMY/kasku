package grpc

import (
	"google.golang.org/protobuf/encoding/protowire"
)

// revokeUserTokensRequest mirror auth.v1.RevokeUserTokensRequest.
type revokeUserTokensRequest struct {
	UserID string // field 1
}

// revokeUserTokensResponse mirror auth.v1.RevokeUserTokensResponse.
type revokeUserTokensResponse struct {
	Success bool // field 1
}

func decodeRevokeUserTokensRequest(b []byte) (*revokeUserTokensRequest, error) {
	s, err := decodeStringField1(b)
	if err != nil {
		return nil, err
	}
	return &revokeUserTokensRequest{UserID: s}, nil
}

func encodeRevokeUserTokensResponse(resp *revokeUserTokensResponse) []byte {
	var b []byte
	if resp.Success {
		b = protowire.AppendTag(b, 1, protowire.VarintType)
		b = protowire.AppendVarint(b, 1)
	}
	return b
}
