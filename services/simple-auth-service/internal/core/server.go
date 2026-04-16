package core

import "github.com/TSM-061/Raggy/simple-auth-service/internal/config"

type Server struct {
	config *config.Config
}

func NewServer(config *config.Config) *Server {
	return &Server{config: config}
}
