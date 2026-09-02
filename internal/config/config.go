package config

import (
	"net"
	"strconv"
	"time"
)

type Server struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (s *Server) Address() string {
	return "http://" + net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
}

type Client struct {
	PollInterval   time.Duration `json:"poll_interval"`
	ReportInterval time.Duration `json:"report_interval"`
}

var ServerConfig = Server{
	Host: "localhost",
	Port: 8080,
}

var ClientConfig = Client{
	PollInterval:   2 * time.Second,
	ReportInterval: 10 * time.Second,
}
