package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
)

var pp = fmt.Println

const defaultListenAddr = "6379"

type Config struct {
	ListenAddr string
}

type Server struct {
	Config
	peers     map[*Peer]bool
	ln        net.Listener
	addPeerCh chan *Peer
	quitCh    chan struct{}
	msgCh     chan []byte
}

func NewServer(cfg Config) *Server {
	if len(cfg.ListenAddr) == 0 {
		cfg.ListenAddr = defaultListenAddr
	}
	return &Server{
		Config:    cfg,
		peers:     make(map[*Peer]bool),
		addPeerCh: make(chan *Peer),
		quitCh:    make(chan struct{}),
		msgCh:     make(chan []byte),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return err
	}
	s.ln = ln
	go s.loop()
	return s.acceptloop()
}

func (s *Server) handlerawmessage(rawMsg []byte) error {
	pp(string(rawMsg))
	return nil
}

func (s *Server) loop() {
	for {
		select {
		case rawmsg := <-s.msgCh:
			if err := s.handlerawmessage(rawmsg); err != nil {
				slog.Error("Raw Message error", "err", err)
			}
		case <-s.quitCh:
			return
		case peer := <-s.addPeerCh:
			s.peers[peer] = true
		}
	}
}

func (s *Server) acceptloop() error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			slog.Error("Accept error", "error", err)
			continue
		}
		go s.handleconn(conn)
	}
}

func (s *Server) handleconn(conn net.Conn) {
	peer := NewPeer(conn, s.msgCh)
	s.addPeerCh <- peer
	pp("new peer connected", "remoteAddr: ", conn.RemoteAddr())
	if err := peer.readloop(); err != nil {
		slog.Error("Peer read error", "err", err, "remoteAddr", conn.RemoteAddr())
	}
}

func main() {
	server := NewServer(Config{})
	log.Fatal(server.Start())
}
