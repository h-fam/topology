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

type mi map[string]interface{}

func MI(k string, v interface{}) *mi {
	return &mi{k: v}
}

func (m *mi) A(k string, v interface{}) *mi {
	(*m)[k] = v
	return m
}

func (m *mi) M() map[string]interface{} {
	return *m
}

func (t *Topology) Load(topo *tpb.Topology) error {
	session := t.driver.NewSession(neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeWrite,
		DatabaseName: "",
	})
	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		result, err := tx.Run("MATCH (n:Topology {name:$name}) return n", MI("name", topo.Name).M())
		if err != nil {
			return nil, err
		}
		_, err = result.Single()
		if err == nil {
			return nil, fmt.Errorf("topology %s already in db", topo.Name)
		}
		if _, err := tx.Run("CREATE (:Topology {name: $name})", MI("name", topo.Name).M()); err != nil {
			return nil, err
		}
		for _, n := range topo.Nodes {
			_, err := tx.Run("MATCH (t:Topology {name: $tname}) CREATE (:Device {name: $name, type: $type})<-[:CONTAINS]-(t)", (&mi{"name": n.Name, "type": n.Type.String(), "tname": topo.Name}).M())
			if err != nil {
				return nil, err
			}
		}
		for _, l := range topo.Links {
			result, err := tx.Run("MATCH (n:Device {name:$name}) return n", MI("name", l.ANode).M())
			if err != nil {
				return nil, err
			}
			_, err = result.Single()
			if err != nil {
				return nil, fmt.Errorf("node %s not found in db", l.ANode)
			}
			result, err = tx.Run("MATCH (n:Device {name:$name}) return n", MI("name", l.ZNode).M())
			if err != nil {
				return nil, err
			}
			_, err = result.Single()
			if err != nil {
				return nil, fmt.Errorf("node %s not found in db", l.ANode)
			}
			result, err = tx.Run("MATCH (d:Device {name:$device}) CREATE (i:Interface {name: $name, device: $device})<-[:CONTAINS]-(d) RETURN i", (&mi{
				"name":   l.AInt,
				"device": l.ANode,
			}).M())
			if err != nil {
				return nil, err
			}
			r, err := result.Single()
			if err != nil {
				return nil, err
			}
			intI, _ := r.Get("i")
			aInt, ok := intI.(neo4j.Node)
			if !ok {
				return nil, fmt.Errorf("failed to assert a interface: %v", intI)
			}
			result, err = tx.Run("MATCH (d:Device {name:$device}) CREATE (i:Interface {name: $name, device: $device})<-[:CONTAINS]-(d) RETURN i", (&mi{
				"name":   l.ZInt,
				"device": l.ZNode,
			}).M())
			if err != nil {
				return nil, err
			}
			r, err = result.Single()
			if err != nil {
				return nil, err
			}
			intI, _ = r.Get("i")
			zInt, ok := intI.(neo4j.Node)
			if !ok {
				return nil, fmt.Errorf("failed to assert z interface: %v", intI)
			}
			result, err = tx.Run("CREATE (l:Link {name: $name, a_int: $a_int, a_device: $a_device, z_int: $z_int, z_device: $z_device}) RETURN l", (&mi{
				"name":     fmt.Sprintf("%s:%s<->%s:%s", l.ANode, l.AInt, l.ZNode, l.ZInt),
				"a_int":    l.AInt,
				"a_device": l.ANode,
				"z_int":    l.ZInt,
				"z_device": l.ZNode,
			}).M())
			if err != nil {
				return nil, err
			}
			r, err = result.Single()
			if err != nil {
				return nil, err
			}
			intI, _ = r.Get("l")
			link, ok := intI.(neo4j.Node)
			if !ok {
				return nil, fmt.Errorf("failed to assert link: %v", intI)
			}
			_, err = tx.Run(`MATCH (aI:Interface), (zI:Interface), (l:Link) WHERE id(aI) = $ai_id and id(zI) = $zi_id and id(l) = $l_id 
		CREATE (aI)-[:CONNECTED]->(l), (zI)-[:CONNECTED]->(l)`, (&mi{
				"ai_id": aInt.Id,
				"zi_id": zInt.Id,
				"l_id":  link.Id,
			}).M())
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	if err != nil {
		return err
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
