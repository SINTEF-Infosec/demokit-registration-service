package registration

import (
	"github.com/SINTEF-Infosec/demokit/core"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func RegisterNode(server *RegistrationServer) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Getting the info
		var nodeInfo core.NodeInfo
		if err := ctx.BindJSON(&nodeInfo); err != nil {
			log.Printf("could not bind node data: %v\n", err)
			ctx.Status(http.StatusBadRequest)
			ctx.Abort()
			return
		}

		server.mutex.Lock()
		defer server.mutex.Unlock()

		// checking that the client IP and the localIP are the same
		clientIP := ctx.ClientIP()
		if clientIP != nodeInfo.LocalIp {
			log.Printf("warning: client IP and provided local IP are different!")
		}

		// Registering the info
		// We still check if there is already a node with that name registered
		for k, node := range server.nodes {
			if node.NodeInfo.Name == nodeInfo.Name && node.NodeInfo.LocalIp != nodeInfo.LocalIp {
				log.Printf("Warning: local IPs differ but nodes' names are the same, %s (%s) / %s (%s)",
					node.NodeInfo.Name, node.NodeInfo.LocalIp,
					nodeInfo.Name, nodeInfo.LocalIp)
				server.nodes[k].NodeInfo = nodeInfo
				ctx.Status(http.StatusOK)
				return
			}
		}

		// If not found, we add a new node
		server.nodes = append(server.nodes, &RegisteredNode{
			NodeInfo: nodeInfo,
		})

		// And we force a refresh but ignore the update failure threshold
		// to prevent deletion in less than (threshold * RefreshPeriod)
		go server.refresh(true)

		ctx.Status(http.StatusOK)
		return
	}
}

func ServeNodesInformation(server *RegistrationServer) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, server.nodes)
	}
}
