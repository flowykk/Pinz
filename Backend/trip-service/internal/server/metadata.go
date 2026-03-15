package server

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const MetadataUserIDKey = "x-user-id"

type contextKey string

const userIDContextKey contextKey = "user_id"

// UserIDFromContext returns the user ID set by the auth interceptor.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDContextKey).(string)
	return v, ok
}

// AuthUnaryInterceptor reads x-user-id from gRPC metadata and puts it in context.
// Skips auth for health check; returns Unauthenticated for trip methods if key is missing.
func AuthUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	if info != nil && (info.FullMethod == "/grpc.health.v1.Health/Check" || info.FullMethod == "/grpc.health.v1.Health/Watch") {
		return handler(ctx, req)
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	vals := md.Get(MetadataUserIDKey)
	if len(vals) == 0 || vals[0] == "" {
		return nil, status.Error(codes.Unauthenticated, "missing x-user-id")
	}
	ctx = context.WithValue(ctx, userIDContextKey, vals[0])
	return handler(ctx, req)
}
