package grpc

import (
	"google.golang.org/grpc"
)

// billingInternalServer adalah marker interface untuk grpc.ServiceDesc.HandlerType.
// grpc-go memerlukan pointer ke interface (bukan struct) karena
// google.golang.org/grpc/server.go memanggil reflect.Type.Implements() yang
// panic kalau diberi non-interface type.
type billingInternalServer any

// billingInternalServiceDesc adalah deskriptor manual untuk service billing.v1.BillingInternal.
// Harus cocok dengan full method name yang digunakan di api-gateway:
// /billing.v1.BillingInternal/GetUserTierLimits
var billingInternalServiceDesc = grpc.ServiceDesc{
	ServiceName: "billing.v1.BillingInternal",
	HandlerType: (*billingInternalServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetUserTierLimits",
			Handler:    getUserTierLimitsHandler,
		},
	},
	Streams: []grpc.StreamDesc{},
}
