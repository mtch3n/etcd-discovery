package machineinfo

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"
)

// todo: consider limit struct size

type Discovery struct {
	IP       string
	Name     string
	Hostname string
	Port     string
}

// todo: refactor code

func (d *Discovery) CanonicalName() string {
	return d.Name
}

func (d *Discovery) Identifier() string {
	return fmt.Sprintf("%s:%s:%s:%s", d.Name, d.Hostname, normalizeIp(d.IP), d.Port)
}

func normalizeIp(ip string) string {
	return strings.Replace(ip, ".", "_", -1)
}

func (d *Discovery) String() string {
	return string(d.Serialize())
}

func (d *Discovery) Serialize() []byte {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(d); err != nil {
		return nil
	}

	return b.Bytes()
}

func Deserialize(data []byte) *Discovery {
	var d Discovery
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&d); err != nil {
		return nil
	}

	return &d
}
