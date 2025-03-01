package server

import (
	"fmt"
	"github.com/atareversei/network-course-projects/pkg/cli"
	"github.com/atareversei/network-course-projects/pkg/http"
	"io"
	"net"
	"os"
	"strings"
)

// Server is used to spawn an HTTP server.
type Server struct {
	// port indicates on which port the server will start.
	port int
	// router holds the routing information.
	// The structure can be simplified as -> [PATH][METHOD]handler
	router map[string]map[string]HandlerFunc
	// fileHandler holds the information about file handlers.
	// The structure can be simplified as [REQUEST_PATH]FILESYSTEM_DIRECTORY_PATH
	fileHandler map[string]string
}

// New returns a server structure.
func New(port int) Server {
	return Server{port: port, router: make(map[string]map[string]HandlerFunc), fileHandler: make(map[string]string)}
}

// Start is used to listen on the specified port.
func (s *Server) Start() {
	cli.Success(fmt.Sprintf("tcp server started at :%d", s.port))
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		cli.Error("server could not be started", err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			cli.Error("connection resulted in an error", err)
			continue
		}
		go s.parseHTTP(conn)
	}
}

type HandlerFunc func(req http.Request, res *http.Response)

// All is a catch-all handler registrar.
func (s *Server) All(pattern string, handler func(req http.Request, res *http.Response)) {
	s.checkResourceEntry(pattern)
	s.router[pattern]["ALL"] = handler
}

// Get is a GET method handler registrar.
func (s *Server) Get(pattern string, handler func(req http.Request, res *http.Response)) {
	s.checkResourceEntry(pattern)
	s.router[pattern]["GET"] = handler
}

// Post is a POST method handler registrar.
func (s *Server) Post(pattern string, handler func(req http.Request, res *http.Response)) {
	s.checkResourceEntry(pattern)
	s.router[pattern]["POST"] = handler
}

// FileHandler is file handler registrar.
func (s *Server) FileHandler(pattern string, directory string) {
	s.fileHandler[pattern] = directory
}

// parseHTTP is used to parse streams of bytes received from a
// TCP connection into a meaningful HTTP request.
func (s *Server) parseHTTP(conn net.Conn) {
	httpRequest := http.NewRequest(conn)
	httpRequest.Parse()
	response := http.NewResponse(conn, httpRequest.Version())
	s.handleRequest(httpRequest, &response)
	conn.Close()
}

// handleRequest is used to handle the routing phase of the request and
// deliver the request-response information to the right handler.
// TODO - refactor the code
func (s *Server) handleRequest(request http.Request, response *http.Response) {
	cli.Info(fmt.Sprintf("%s %s", request.Method(), request.Path()))
	// TODO - enhance type checking
	// simple check to see if the request points to a file
	if strings.Contains(request.Path(), ".") {
		// TODO - enhance inefficient lookup
		for k, _ := range s.fileHandler {
			if strings.Contains(request.Path(), k) {
				i := strings.Index(request.Path(), k)
				filePath := request.Path()[i+len(k):]
				directoryPath := s.fileHandler[k]
				fullPath := directoryPath + filePath
				f, err := os.Open(fullPath)
				// TODO - check for 404 and faulty files
				if err != nil {
					response.Write([]byte{})
				}
				defer f.Close()
				buf := make([]byte, 1024)
				for {
					n, err := f.Read(buf)
					if err != nil && err != io.EOF {
						// TODO - add proper response
						response.Write([]byte{})
					}
					if n == 0 {
						break
					}
					response.Write(buf[:n])
				}
			}
		}
	} else {
		resource, resOk := s.router[request.Path()]
		if !resOk {
			// TODO - add proper response
			response.WriteHeader(http.NotFound)
			return
		}
		handler, handlerOk := resource[strings.ToUpper(request.Method())]
		if !handlerOk {
			catchAll, allOk := resource["ALL"]
			if allOk {
				catchAll(request, response)
				return
			}
			// TODO - add proper response
			response.WriteHeader(http.MethodNotAllowed)
			return
		}
		handler(request, response)
	}
}

// checkResourceEntry is used to initialize the inner map of a router
// if it has not yet been initialized.
func (s *Server) checkResourceEntry(pattern string) {
	_, ok := s.router[pattern]
	if !ok {
		s.router[pattern] = make(map[string]HandlerFunc)
	}
}
