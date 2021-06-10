package main

import (
	"flag"
	"io/ioutil"

	tpb "github.com/google/kne/proto/topo"
	"github.com/h-fam/topology/g1"
	"github.com/kr/pretty"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/prototext"
)

var (
	topoFlag = flag.String("topo", "", "topology to load")
)

func main() {
	flag.Parse()
	t, err := g1.New()
	if err != nil {
		log.Fatalf("failed to create topology: %v")
	}
	if len(flag.Args()) == 1 {
		results, err := t.Query(flag.Args()[0], nil)
		if err != nil {
			log.Fatalf("Query Error: %v", err)
		}
		for results.Next() {
			r := results.Record()
			pretty.Println(r)
		}
		return
	}
	b, err := ioutil.ReadFile(*topoFlag)
	if err != nil {
		log.Fatalf("failed to read topo file: %v", err)
	}
	topo := &tpb.Topology{}
	if err := prototext.Unmarshal(b, topo); err != nil {
		log.Fatalf("failed cannot unmarshal topo: %v", err)
	}
	if err = t.Load(topo); err != nil {
		log.Fatalf("failed load topology: %v", err)
	}
}
