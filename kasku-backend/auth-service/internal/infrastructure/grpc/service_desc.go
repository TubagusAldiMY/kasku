package grpc

import (
	"google.golang.org/grpc"
)

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
	HandlerType: (*AuthGRPCServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "ValidateToken", Handler: validateTokenHandler},
		{MethodName: "GetUserByID", Handler: getUserByIDHandler},
		{MethodName: "GetUserByEmail", Handler: getUserByEmailHandler},
		{MethodName: "RevokeUserTokens", Handler: revokeUserTokensHandler},
		{MethodName: "IsTokenBlacklisted", Handler: isTokenBlacklistedHandler},
	},
	Streams: []grpc.StreamDesc{},
}
