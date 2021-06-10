package g1

import (
	"fmt"
	"os"

	tpb "github.com/google/kne/proto/topo"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type Topology struct {
	driver neo4j.Driver
}

func lookupEnvOrGetDefault(key string, defaultValue string) string {
	if env, found := os.LookupEnv(key); !found {
		return defaultValue
	} else {
		return env
	}
}

type config struct {
	URI      string
	Username string
	Password string
	Database string
}

func New() (*Topology, error) {
	config := &config{
		URI:      lookupEnvOrGetDefault("NEO4J_URI", "neo4j://localhost:7687"),
		Username: lookupEnvOrGetDefault("NEO4J_USER", "neo4j"),
		Password: lookupEnvOrGetDefault("NEO4J_PASSWORD", "test"),
		Database: "",
	}
	driver, err := neo4j.NewDriver(config.URI, neo4j.BasicAuth(config.Username, config.Password, config.Database))
	if err != nil {
		return nil, err
	}
	return &Topology{
		driver: driver,
	}, nil
}

func PortKey(d, p string) string {
	return fmt.Sprintf("%s:%s", d, p)
}

type Device struct {
	pb *tpb.Node
}

func (d *Device) Node() *neo4j.Node {
	return &neo4j.Node{
		Labels: []string{"Device"},
		Props: map[string]interface{}{
			"name": d.pb.Name,
		},
	}
}

type Interface struct {
	Device string
	Name   string
}

func (i *Interface) Node() *neo4j.Node {
	return &neo4j.Node{
		Labels: []string{"Interface"},
		Props: map[string]interface{}{
			"device": i.Device,
			"name":   i.Name,
		},
	}
}

func kv(k string, v interface{}) map[string]interface{} {
	return map[string]interface{}{
		k: v,
	}
}

func (t *Topology) Load(topo *tpb.Topology) error {
	session := t.driver.NewSession(neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeWrite,
		DatabaseName: "",
	})
	results, err := session.Run("MATCH (n) RETURN n", nil)
	if err != nil {
		return err
	}
	for results.Next() {
		fmt.Println(results.Record())
	}
	topology := &neo4j.Node{
		Labels: []string{"Topology"},
		Props: map[string]interface{}{
			"name": topo.Name,
		},
	}
	result, err := session.Run("MATCH (n:Topology {name:$name}) return n", kv("name", topo.Name))
	if err != nil {
		return err
	}
	_, err = result.Single()
	if err == nil {
		return fmt.Errorf("topology %s already in db", topo.Name)
	}
	if _, err := session.Run("CREATE (:Topology {name: $name})", topology.Props); err != nil {
		return err
	}
	for _, n := range topo.Nodes {
		_, err := session.Run("MATCH (t:Topology {name: $tname}) CREATE (:Device {name: $name, type: $type})<-[:contains]-(t)", map[string]interface{}{"name": n.Name, "type": n.Type.String(), "tname": topo.Name})
		if err != nil {
			return err
		}
	}
	for _, l := range topo.Links {
		result, err := session.Run("MATCH (n:Device {name:$name}) return n", kv("name", l.ANode))
		if err != nil {
			return err
		}
		_, err = result.Single()
		if err != nil {
			return fmt.Errorf("node %s not found in db", l.ANode)
		}
		result, err = session.Run("MATCH (n:Device {name:$name}) return n", kv("name", l.ZNode))
		if err != nil {
			return err
		}
		_, err = result.Single()
		if err != nil {
			return fmt.Errorf("node %s not found in db", l.ANode)
		}
		result, err = session.Run("MATCH (d:Device {name:$device}) CREATE (i:Interface {name: $name, device: $device})<-[:contains]-(d) RETURN i", map[string]interface{}{
			"name":   l.AInt,
			"device": l.ANode,
		})
		if err != nil {
			return err
		}
		r, err := result.Single()
		if err != nil {
			return err
		}
		intI, _ := r.Get("i")
		aInt, ok := intI.(neo4j.Node)
		if !ok {
			return fmt.Errorf("failed to assert a interface: %v", intI)
		}
		result, err = session.Run("MATCH (d:Device {name:$device}) CREATE (i:Interface {name: $name, device: $device})<-[:contains]-(d) RETURN i", map[string]interface{}{
			"name":   l.ZInt,
			"device": l.ZNode,
		})
		if err != nil {
			return err
		}
		r, err = result.Single()
		if err != nil {
			return err
		}
		intI, _ = r.Get("i")
		zInt, ok := intI.(neo4j.Node)
		if !ok {
			return fmt.Errorf("failed to assert z interface: %v", intI)
		}
		result, err = session.Run("CREATE (l:Link {a_int: $a_int, a_device: $a_device, z_int: $z_int, z_device: $z_device}) RETURN l", map[string]interface{}{
			"name":     fmt.Sprintf("%s:%s<->%s:%s", l.ANode, l.AInt, l.ZNode, l.ZInt),
			"a_int":    l.AInt,
			"a_device": l.ANode,
			"z_int":    l.ZInt,
			"z_device": l.ZNode,
		})
		if err != nil {
			return err
		}
		r, err = result.Single()
		if err != nil {
			return err
		}
		intI, _ = r.Get("l")
		link, ok := intI.(neo4j.Node)
		if !ok {
			return fmt.Errorf("failed to assert link: %v", intI)
		}
		_, err = session.Run(`MATCH (aI:Interface), (zI:Interface), (l:Link) WHERE id(aI) = $ai_id and id(zI) = $zi_id and id(l) = $l_id 
		CREATE (aI)-[:CONNECTED]->(l), (zI)-[:CONNECTED]->(l)`, map[string]interface{}{
			"ai_id": aInt.Id,
			"zi_id": zInt.Id,
			"l_id":  link.Id,
		})
		if err != nil {
			return err
		}
	}
	session.Close()
	return nil
}

func (t *Topology) Query(q string, params map[string]interface{}) (neo4j.Result, error) {
	session := t.driver.NewSession(neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: "",
	})
	return session.Run(q, params)
}
