package outline

import (
	"net/http"

	"go.uber.org/zap"
)

// control panel server
// control multiple servers
type Server struct {
	router  *http.ServeMux
	logger  *zap.Logger
	servers map[uint32]*OutlineServer
}

func NewServer(servers map[uint32]*OutlineServer, logger *zap.Logger) *Server {
	s := &Server{
		router:  http.NewServeMux(),
		logger:  logger,
		servers: servers,
	}

	type ServerEntry struct {
		URL     string
		Pattern string
	}
	entrys := make([]ServerEntry, 0, len(servers))

	for _, server := range servers {
		pattern := "/outline/manager"
		server.SetRouter(pattern, s.router)
		entrys = append(entrys, ServerEntry{URL: server.URL, Pattern: pattern})
	}

	return s
}

func (s *Server) Handler(r *http.Request) (http.Handler, bool) {
	handler, pattern := s.router.Handler(r)
	return handler, pattern != ""
}
