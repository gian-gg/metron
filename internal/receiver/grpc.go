package receiver

import (
	"context"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type traceService struct {
	coltracepb.UnimplementedTraceServiceServer
	receiver *Receiver
}

func (s *traceService) Export(
	ctx context.Context, req *coltracepb.ExportTraceServiceRequest,
) (*coltracepb.ExportTraceServiceResponse, error) {
	if err := s.receiver.accept(ctx, req); err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}
	return &coltracepb.ExportTraceServiceResponse{}, nil
}
