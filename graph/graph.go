package graph

import (
	"github.com/gomodule/redigo/redis"
	"github.com/redislabs/redisgraph-go"
	log "github.com/sirupsen/logrus"

	tpb "github.com/google/kne/proto/topo"
)

type Topology struct {
	g    *redisgraph.Graph
	conn *redis.Conn
}

func New() (*Topology, error) {
	conn, _ := redis.Dial("tcp", "127.0.0.1:6379")
	g := redisgraph.GraphNew("topology", conn)
	return &Topology{
		g: g,
	}, nil
}

func PortKey(d,p string) string {
	return fmt.Sprintf("%s:%s", d,p)
}

func (t *Topology) Load(topo *tpb.Topology) error {
	nodes := redisgraph.NodeNew("Topology", "", map[string]interface{}{
		"name": topo,
	})
	t.g.AddNode(nodes)
	redisgraph.Path()
	for _, n := range topo.Nodes {
		log.Infof("Adding Node: %s:%s", n.Name, n.Type)
		node := redisgraph.NodeNew("Device", n.Name, map[string]interface{}{
			"name", n.Name,
			"pb": n,
		})
		t.g.AddNode(node)
		t.g.AddEdge(redisgraph.EdgeNew("lcontains", nodes, node))
	}
	t.g.Commit()
	q := `MATCH (:Topology)-[:lcontains]->(d:Devices) RETURN d.name`
	r, err := t.g.Query(q)
	if err != nil {
		return err
	}
	r.
	uid := 0
	for _, l := range topo.Links {
		log.Infof("Adding Link: %s:%s %s:%s", l.ANode, l.AInt, l.ZNode, l.ZInt)
		srcInt := redisgraph.NodeNew("port", PortKey(l.ANode,l.AInt), map[string]interface{}{
			"device": l.ANode,
			"port": l.AInt,
		})
		dstInt := redisgraph.NodeNew("port", PortKey(l.ZNode, l.ZInt), map[string]interface{}{
			"device": l.ZNode,
			"port": lZInt,
		})
		e := redisgraph.EdgeNew("contains", sNode, srcInt, map[string]interface{}{
			type: 
		})
		t.g.AddEdge(e)
		t.g.AddNode(srcInt)
		t.g.AddNode(dstInt)
		uid++
	}
}

func (t *Topology) Query(q string) (*redisgraph.QueryResult, error) {
	return t.g.Query(q)
}
