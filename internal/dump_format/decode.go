package dump_format

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io"
)

type decider struct {
	Common
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
	c := decider{}
	err := dec.decoder.Decode(c)
	if err != nil {
		// return unwrapped because it will eventually be the io.EOF sentinel error
		return nil, err
	}

	// This double decode is bad but I think can be fixed in future with a custom unmarshaller
	raw, err := json.Marshal(c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to re-marshal line")
	}

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
