package cluster

// Node represents a node in nano cluster, which will contains a group of services.
// All services will register to cluster and messages will be forwarded to the node
// which provides respective service
type Node struct {
	isMaster bool
}

// IsMaster returns a boolean to indicate whether the node is a master node
func (n *Node) IsMaster() bool {
	return n.isMaster
}
