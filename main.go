package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

type SimpleServer struct {
	address string
	proxy   *httputil.ReverseProxy
}

type LoadBalancer struct {
	port              string
	roundRobinCount   int
	servers           []Server
	activeConnections map[Server]int
	mutex             sync.Mutex
}

type Server interface {
	Address() string
	isAlive() bool
	Serve(w http.ResponseWriter, r *http.Request)
}

type Config struct {
	Servers   []string `yaml:"servers"`
	Algorithm string   `yaml:"algorithm"`
	Port      string   `yaml:"port"`
}

func newSimpleServer(address string) *SimpleServer {
	serverUrl, err := url.Parse(address)
	handleError(err)
	return &SimpleServer{address: address, proxy: httputil.NewSingleHostReverseProxy(serverUrl)}
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	activeConnections := make(map[Server]int)
	for _, server := range servers {
		activeConnections[server] = 0
	}
	return &LoadBalancer{port: port, roundRobinCount: 0, servers: servers, activeConnections: activeConnections}
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	config, _ := readConfigFile()
	switch config.Algorithm {
	case "round-robin":
		server := lb.servers[lb.roundRobinCount%len(lb.servers)]
		for !server.isAlive() {
			lb.roundRobinCount++
			server = lb.servers[lb.roundRobinCount%len(lb.servers)]
		}
		lb.roundRobinCount++
		return server
	case "least-connections":
		leastConnections := math.MaxInt64
		for _, server := range lb.servers {
			connections := lb.activeConnections[server]
			if connections == 0 && server.isAlive() {
				return server
			}
			if connections < leastConnections && server.isAlive() {
				leastConnections = connections
				return server
			}
		}
	}
	return lb.servers[0]
}

func (lb *LoadBalancer) serveProxy(w http.ResponseWriter, r *http.Request) {
	targetServer := lb.getNextAvailableServer()
	lb.incrementActiveConnections(targetServer)
	log.Printf("Forwading requests to address %s", targetServer.Address())
	defer lb.decrementActiveConnections(targetServer)
	targetServer.Serve(w, r)
}

func (lb *LoadBalancer) incrementActiveConnections(server Server) {
	lb.mutex.Lock()
	lb.activeConnections[server]++
	lb.mutex.Unlock()
}

func (lb *LoadBalancer) decrementActiveConnections(server Server) {
	lb.mutex.Lock()
	if lb.activeConnections[server] > 0 {
		lb.activeConnections[server]--
	}
	lb.mutex.Unlock()
}

func readConfigFile() (*Config, error) {
	filePath := os.Getenv("YALB_CONFIG")
	if filePath == "" {
		return nil, errors.New("Set the YALB_CONFIG environment variable")
	}
	buff, err := os.ReadFile(filePath)
	handleError(err)
	conf := &Config{}
	yamlErr := yaml.Unmarshal(buff, conf)
	handleError(yamlErr)
	return conf, err
}

func (server *SimpleServer) Address() string { return server.address }

func (server *SimpleServer) isAlive() bool {
	resp, err := http.Get(server.address)
	if err != nil {
		return false
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println(err)
		}
	}(resp.Body)

	if resp.StatusCode == http.StatusOK {
		return true
	}
	return false
}

func (server *SimpleServer) Serve(w http.ResponseWriter, r *http.Request) {
	server.proxy.ServeHTTP(w, r)
}

func handleError(err error) {
	if err != nil {
		log.Println(err.Error())
	}
}

func main() {
	config, confErr := readConfigFile()
	if confErr != nil {
		panic(confErr.Error())
	}

	var servers []Server
	for _, address := range config.Servers {
		servers = append(servers, newSimpleServer(address))
	}

	lb := NewLoadBalancer(config.Port, servers)

	handleRedirect := func(w http.ResponseWriter, r *http.Request) {
		lb.serveProxy(w, r)
	}
	http.HandleFunc("/", handleRedirect)
	log.Printf("Proxying Requests at port %s\n", lb.port)
	err := http.ListenAndServe(":"+lb.port, nil)
	handleError(err)
}
