package enc

func calcNonce(blockNum int64) *[24]byte {
	var nonce [24]byte
	pos := 0
	for blockNum > 0 {
		nonce[24-pos-1] = byte(blockNum & 0xff)
		pos++
		blockNum >>= 8
	}
	return &nonce
}
