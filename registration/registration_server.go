package registration

import (
	"encoding/json"
	"fmt"
	"github.com/SINTEF-Infosec/demokit/core"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	ServerAddr                   = ":4000"
	RefreshPeriod                = 60 // in seconds
	FailRefreshEjectionThreshold = 3
)

type RegisteredNode struct {
	NodeInfo        core.NodeInfo   `json:"info"`
	NodeStatus      core.NodeStatus `json:"status"`
	failUpdateCount int
}

type RegistrationServer struct {
	router *gin.Engine
	nodes  []*RegisteredNode
	mutex  *sync.Mutex
}

func NewRegistrationServer() *RegistrationServer {
	server := &RegistrationServer{
		nodes: make([]*RegisteredNode, 0),
		mutex: &sync.Mutex{},
	}

	rsRouter := gin.Default()
	rsRouter.GET("/nodes", ServeNodesInformation(server))
	rsRouter.POST("/register", RegisterNode(server))

	server.router = rsRouter
	return server
}

func (rs *RegistrationServer) Start() {
	// Start monitoring for nodes
	go rs.monitor()

	// Start gin server
	if err := rs.router.Run(ServerAddr); err != nil {
		log.Fatalf("could not start registration server: %v", err)
	}
}

func (rs *RegistrationServer) monitor() {
	// executes the refresh function every X seconds/minutes
	ticker := time.NewTicker(RefreshPeriod * time.Second)
	for {
		select {
		case <-ticker.C:
			rs.refresh(false)
		}
	}
}

func (rs *RegistrationServer) refresh(ignoreThreshold bool) {
	log.Println("Refreshing nodes' information...")
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	rsClient := http.Client{
		Timeout: 2 * time.Second,
	}

	wg := sync.WaitGroup{}
	for k, node := range rs.nodes {
		if node.NodeInfo.LocalIp != "" {
			wg.Add(1)
			go func() {
				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s:8081/status", node.NodeInfo.LocalIp), nil)
				if err != nil {
					log.Printf("error :could not create request for node %s: %v\n", node.NodeInfo.Name, err)
					if !ignoreThreshold {
						rs.nodes[k].failUpdateCount += 1
					}
					wg.Done()
					return
				}

				req.Header.Set("User-Agent", "registration-service")

				res, err := rsClient.Do(req)
				if err != nil {
					log.Printf("error: could not make request for node %s: %v\n", node.NodeInfo.Name, err)
					if !ignoreThreshold {
						rs.nodes[k].failUpdateCount += 1
					}
					wg.Done()
					return
				}

				if res.Body != nil {
					defer res.Body.Close()
				}

				body, readErr := ioutil.ReadAll(res.Body)
				if readErr != nil {
					log.Printf("could not read response body for node %s, %v\n", node.NodeInfo.Name, err)
					if !ignoreThreshold {
						rs.nodes[k].failUpdateCount += 1
					}
					wg.Done()
					return
				}

				var nodeStatus core.NodeStatus

				if err := json.Unmarshal(body, &nodeStatus); err != nil {
					log.Printf("could not unmarshal node status for node %s: %v", node.NodeInfo.Name, err)
					if !ignoreThreshold {
						rs.nodes[k].failUpdateCount += 1
					}
					wg.Done()
					return
				}

				// Updating node
				rs.nodes[k].NodeStatus = nodeStatus
				rs.nodes[k].failUpdateCount = 0 // We reset the count (we don't ignore the threshold here)
				wg.Done()
			}()
		}
	}

	wg.Wait()

	// Removing nodes that went above the threshold
	for k, node := range rs.nodes {
		if node.failUpdateCount > FailRefreshEjectionThreshold {
			log.Printf("No signs of node %s, removing it\n", node.NodeInfo.Name)
			rs.nodes[k] = rs.nodes[len(rs.nodes)-1]
			rs.nodes = rs.nodes[:len(rs.nodes)-1]
		}
	}
}
