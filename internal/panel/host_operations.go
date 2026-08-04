package panel

import "context"

// HostOperationService is the single transport boundary for structured host
// operations. HTTP handlers and AI tools share it so request IDs, Agent calls,
// validation helpers and error mapping cannot drift into a second host API.
type HostOperationService struct {
	server *Server
}

func newHostOperationService(server *Server) *HostOperationService {
	return &HostOperationService{server: server}
}

func (s *HostOperationService) Get(ctx context.Context, path, rawQuery, requestID string) (AgentResponse, error) {
	return s.server.agent.Get(ctx, path, rawQuery, requestID)
}

func (s *HostOperationService) Do(ctx context.Context, method, path, rawQuery, requestID string, body []byte) (AgentResponse, error) {
	return s.server.agent.Do(ctx, method, path, rawQuery, requestID, body)
}
