package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	fs "github.com/peragwin/vuzicgo/audio/sensors/freqsensor"
)

type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func NewVec3(x, y, z float64) Vec3 {
	return Vec3{x, y, z}
}

type CanopyMessageType int

const (
	BangMessageType   = 0
	ScalarMessageType = 1
)

type CanopyMessage struct {
	MessageType CanopyMessageType `json:"message_type"`
	Position    Vec3              `json:"position"`
	Value       float64           `json:"value"`
}

func NewCanopyMessage(typ CanopyMessageType, pos Vec3, val float64) *CanopyMessage {
	return &CanopyMessage{MessageType: typ, Position: pos, Value: val}
}

func (c *CanopyMessage) publish(client mqtt.Client, topic string) error {
	bs, err := json.Marshal(c)
	if err != nil {
		return err
	}
	// log.Printf("[Debug] publish %s: %s\n", topic, string(bs))
	token := client.Publish(topic, 0, false, string(bs))
	token.WaitTimeout(time.Millisecond * 10)
	if err := token.Error(); err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}
	return nil
}

type Entity struct {
	Position Vec3
}

type Nocturne struct {
	client   mqtt.Client
	src      *fs.FrequencySensor
	params   *fs.Parameters
	entities []Entity
}

func NewNocturne(broker string, params *fs.Parameters, src *fs.FrequencySensor) (*Nocturne, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	brokerurl, err := url.Parse(broker)
	if err != nil {
		return nil, err
	}

	client := mqtt.NewClient(&mqtt.ClientOptions{
		Servers:  []*url.URL{brokerurl},
		ClientID: fmt.Sprintf("visualizer-%s", hostname),
	})
	log.Println("Connected to mqtt broker...")
	conn := client.Connect()
	conn.Wait()
	if err := conn.Error(); err != nil {
		log.Fatalf("failed to connect: %s", err)
	}

	f, err := os.Open("entities.json")
	if err != nil {
		return nil, err
	}
	var data []struct{ Position []float64 }
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return nil, err
	}
	entities := make([]Entity, len(data))
	for i, e := range data {
		p := e.Position
		entities[i] = Entity{Position: NewVec3(p[0], p[1], p[2])}
	}

	return &Nocturne{client: client, src: src, params: params, entities: entities}, nil
}

func (n *Nocturne) render() error {
	amp := n.src.Amplitude[0]
	scales := n.src.Scales
	nrg := 0.0
	bass := 0.0
	mids := 0.0
	treb := 0.0
	for i := range amp {
		a := scales[i] * (amp[i] - 1.0)
		nrg += a
		if i < 4 {
			bass += a
		} else if i < 8 {
			mids += a
		} else {
			treb += a
		}
	}
	bass /= 4.0
	mids /= 4.0
	treb /= 8.0
	nrg /= float64(len(amp))

	var err error

	// err := NewCanopyMessage(ScalarMessageType, NewVec3(0, 0, 0), bass).
	// 	publish(n.client, "vuzic/scalars/bass")
	// if err != nil {
	// 	return err
	// }

	eidx := rand.Intn(len(n.entities))
	pos := n.entities[eidx].Position

	if bass > 1.0 {
		err = NewCanopyMessage(BangMessageType, pos, 1).
			publish(n.client, "vuzic/bangs/bass")
		if err != nil {
			return err
		}
	}

	// err = NewCanopyMessage(ScalarMessageType, NewVec3(0, 0, 0), mids).
	// 	publish(n.client, "vuzic/scalars/mids")
	// if err != nil {
	// 	return err
	// }
	if mids > 1.0 {
		err = NewCanopyMessage(BangMessageType, pos, 1).
			publish(n.client, "vuzic/bangs/mids")
		if err != nil {
			return err
		}
	}

	// err = NewCanopyMessage(ScalarMessageType, NewVec3(0, 0, 0), treb).
	// 	publish(n.client, "vuzic/scalars/treb")
	// if err != nil {
	// 	return err
	// }
	if treb > 1.0 {
		err = NewCanopyMessage(BangMessageType, pos, 1).
			publish(n.client, "vuzic/bangs/treb")
		if err != nil {
			return err
		}
	}

	// err = NewCanopyMessage(ScalarMessageType, NewVec3(0, 0, 0), nrg).
	// 	publish(n.client, "vuzic/scalars/energy")
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (n *Nocturne) StartRenderer(frameRate int, done chan struct{}) {
	defer close(done)

	delay := time.Second / time.Duration(frameRate)
	ticker := time.NewTicker(delay)

	// go func() {
	for {
		<-ticker.C
		if err := n.render(); err != nil {
			log.Println("[ERROR] failed to render:", err)
		}
	}
	// }()
}
