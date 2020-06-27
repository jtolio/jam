package enc

import (
	"fmt"
	"strings"
)

type CodecMap struct {
	codecsBySuffix map[string]Codec
	defaultCodec   Codec
}

func NewCodecMap(defaultCodec Codec) *CodecMap {
	return &CodecMap{
		codecsBySuffix: map[string]Codec{},
		defaultCodec:   defaultCodec,
	}
}

func (m *CodecMap) Register(suffix string, c Codec) {
	if suffix == "" {
		panic("empty suffix")
	}
	if _, exists := m.codecsBySuffix[suffix]; exists {
		panic(fmt.Sprintf("suffix %q already registered", suffix))
	}
	m.codecsBySuffix[suffix] = c
}

func (m *CodecMap) CodecForPath(path string) Codec {
	for suffix, codec := range m.codecsBySuffix {
		if strings.HasSuffix(path, suffix) {
			return codec
		}
	}
	return m.defaultCodec
}
