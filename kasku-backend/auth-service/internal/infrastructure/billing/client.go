package billing

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protowire"
)

// GRPCClient calls billing-service via gRPC to fetch subscription tier.
// GetTierName always returns "FREE" on any error — auth must not fail to issue a JWT.
type GRPCClient struct {
	conn *grpc.ClientConn
}

func NewGRPCClient(addr string) (*GRPCClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &GRPCClient{conn: conn}, nil
}

func (c *GRPCClient) Close() error { return c.conn.Close() }

func (c *GRPCClient) GetTierName(ctx context.Context, userID string) string {
	var req []byte
	req = protowire.AppendTag(req, 1, protowire.BytesType)
	req = protowire.AppendString(req, userID)

	reqMsg := &rawMsg{data: req}
	respMsg := &rawMsg{}
	if err := c.conn.Invoke(ctx,
		"/billing.v1.BillingInternal/GetUserTierLimits",
		reqMsg, respMsg,
		grpc.ForceCodec(rawCodec{}),
	); err != nil {
		return "FREE"
	}
	return decodeTierName(respMsg.data)
}

func decodeTierName(b []byte) string {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			break
		}
		b = b[n:]
		if num == 7 && typ == protowire.BytesType {
			s, sn := protowire.ConsumeString(b)
			if sn > 0 {
				return s
			}
			break
		}
		fn := protowire.ConsumeFieldValue(num, typ, b)
		if fn < 0 {
			break
		}
		b = b[fn:]
	}
	return "FREE"
}

type rawMsg struct{ data []byte }
type rawCodec struct{}

func (rawCodec) Name() string                              { return "proto" }
func (rawCodec) Marshal(v interface{}) ([]byte, error)    { return v.(*rawMsg).data, nil }
func (rawCodec) Unmarshal(data []byte, v interface{}) error { v.(*rawMsg).data = data; return nil }
