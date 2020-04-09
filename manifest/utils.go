package manifest

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/golang/protobuf/proto"
)

func MarshalSized(pb proto.Message) ([]byte, error) {
	b := proto.NewBuffer(make([]byte, 4))
	err := b.Marshal(pb)
	if err != nil {
		return nil, err
	}
	rv := b.Bytes()
	if len(rv) > math.MaxInt32 {
		return nil, fmt.Errorf("protobuf too large")
	}
	if len(rv) < 4 {
		panic("unexpected")
	}
	binary.BigEndian.PutUint32(rv[:4], uint32(len(rv)-4))
	return rv, nil
}

func UnmarshalSized(source io.Reader, pb proto.Message) error {
	var sizeBytes [4]byte
	_, err := io.ReadFull(source, sizeBytes[:])
	if err != nil {
		return err
	}
	size := binary.BigEndian.Uint32(sizeBytes[:])
	data := make([]byte, size)
	_, err = io.ReadFull(source, data)
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return err
	}
	return proto.Unmarshal(data, pb)
}
