package grpc

import (
	"google.golang.org/grpc"
)

// authInternalServer adalah marker interface untuk grpc.ServiceDesc.HandlerType.
// Sejak grpc-go v1.67, HandlerType WAJIB pointer ke interface (bukan struct) —
// google.golang.org/grpc/server.go memanggil reflect.Type.Implements() yang
// panic kalau diberi non-interface type.
type authInternalServer any

// authInternalServiceDesc adalah deskriptor manual untuk service auth.v1.AuthInternal.
// Nama method DI SINI HARUS PERSIS dengan yang ditulis client di service lain:
//
//	/auth.v1.AuthInternal/ValidateToken
//	/auth.v1.AuthInternal/GetUserByID
//	/auth.v1.AuthInternal/GetUserByEmail
//	/auth.v1.AuthInternal/RevokeUserTokens
//	/auth.v1.AuthInternal/IsTokenBlacklisted
var authInternalServiceDesc = grpc.ServiceDesc{
	ServiceName: "auth.v1.AuthInternal",
	HandlerType: (*authInternalServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "ValidateToken", Handler: validateTokenHandler},
		{MethodName: "GetUserByID", Handler: getUserByIDHandler},
		{MethodName: "GetUserByEmail", Handler: getUserByEmailHandler},
		{MethodName: "RevokeUserTokens", Handler: revokeUserTokensHandler},
		{MethodName: "IsTokenBlacklisted", Handler: isTokenBlacklistedHandler},
	},
	Streams: []grpc.StreamDesc{},
}
