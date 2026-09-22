package config

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (s *Server) Address() string {
	return "http://" + s.String()
}

func (s *Server) String() string {
	return net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
}

func (s *Server) Set(value string) error {
	hp := strings.Split(value, ":")
	if len(hp) != 2 {
		return errors.New("need address in a form host:port")
	}
	port, err := strconv.Atoi(hp[1])
	if err != nil {
		return err
	}
	s.Host = hp[0]
	s.Port = port
	return nil
}

var ServerConfig = &Server{
	Host: "localhost",
	Port: 8080,
}

type StorageInterval struct {
	Interval time.Duration
}

func (s *StorageInterval) String() string {
	return s.Interval.String()
}

func (s *StorageInterval) Set(value string) error {
	seconds, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	if seconds < 0 {
		return errors.New("invalid store interval")
	}
	s.Interval = time.Duration(seconds) * time.Second
	return nil
}

type StoragePath struct {
	Path string
}

func (s *StoragePath) String() string {
	return s.Path
}

func (s *StoragePath) Set(value string) error {
	s.Path = value
	return nil // todo validate
}

type StorageRestore bool

func (s *StorageRestore) String() string {
	if s == nil {
		return "false"
	}
	return strconv.FormatBool(bool(*s))
}

func (s *StorageRestore) Set(value string) error {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	*s = StorageRestore(b)
	return nil
}

type Storage struct {
	Restore       StorageRestore
	StoreInterval StorageInterval
	StoragePath   StoragePath
}

type ClientInterval struct {
	Interval time.Duration `json:"interval"`
}

func (c *ClientInterval) String() string {
	return c.Interval.String()
}

func (c *ClientInterval) Set(value string) error {
	seconds, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	if seconds <= 0 {
		return errors.New("invalid client interval")
	}
	c.Interval = time.Duration(seconds) * time.Second
	return nil
}

type Client struct {
	Server         *Server         `json:"server"`
	PollInterval   *ClientInterval `json:"poll_interval"`
	ReportInterval *ClientInterval `json:"report_interval"`
}

var ClientConfig = Client{
	Server: &Server{
		Host: "localhost",
		Port: 8080,
	},
	PollInterval:   &ClientInterval{2 * time.Second},
	ReportInterval: &ClientInterval{10 * time.Second},
}

var StorageConfig = Storage{
	StoreInterval: StorageInterval{300 * time.Second},
}
