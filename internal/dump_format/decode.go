package dump_format

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
)

type DecodedLine struct {
	value interface{}
}

func (d *DecodedLine) Get() interface{} {
	return d.value
}

func (d *DecodedLine) UnmarshalJSON(data []byte) error {
	c := &Common{}
	err := json.Unmarshal(data, c)
	if err != nil {
		return err
	}

	switch c.Type {
	case RPCLine:
		d.value = &RPC{}
	case MessageLine:
		d.value = &Message{}
	case StatusLine:
		d.value = &Status{}
	default:
		return errors.Errorf("unknown line type %s", c.Type)
	}

	return json.Unmarshal(data, &d.value)
}

type decider struct {
	Common
	json.RawMessage
	Contents map[string]interface{}
}

type Decoder struct {
	decoder *json.Decoder
}

func NewDecoder(r io.Reader) Decoder {
	return Decoder{
		decoder: json.NewDecoder(r),
	}
}

// Returns a pointer to one of the concrete line type structs
func (dec *Decoder) Decode() (interface{}, error) {
	c := &decider{}
	err := dec.decoder.Decode(c)
	if err != nil {
		// return unwrapped because it will eventually be the io.EOF sentinel error
		return nil, err
	}
	fmt.Println("read", c)

	// This double decode is bad but I think can be fixed in future with a custom unmarshaller
	raw, err := json.Marshal(c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to re-marshal line")
	}
	fmt.Println("remarshalled as", string(raw))

	var line interface{}
	switch c.Type {
	case RPCLine:
		line = &RPC{}
	case MessageLine:
		line = &Message{}
	case StatusLine:
		line = &Status{}
	default:
		return nil, errors.Errorf("unknown line type %s", c.Type)
	}

	err = json.Unmarshal(raw, line)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal into concrete line type")
	}

	return line, nil
}
