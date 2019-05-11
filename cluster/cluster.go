package cluster

// Cluster represents a nano cluster, which contains a bunch of nano nodes
// and each of them provide a group of different services. All services requests
// from client will send to gate firstly and be forwarded to appropriate node.
type Cluster struct {
	nodes []*Node
}

func (c *Cluster) Nodes() []*Node {
	return c.nodes
}
