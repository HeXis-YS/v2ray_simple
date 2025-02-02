package quic

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/tls"
	"encoding/binary"

	"github.com/e1732a364fed/v2ray_simple/tlsLayer"
	"github.com/e1732a364fed/v2ray_simple/utils"
	"github.com/quic-go/quic-go/quicvarint"
	"github.com/marten-seemann/qtls"
	"golang.org/x/crypto/hkdf"
)

const (
	versionDraft29 uint32 = 0xff00001d
	version1       uint32 = 0x1
)

var (
	quicSaltOld  = []byte{0xaf, 0xbf, 0xec, 0x28, 0x99, 0x93, 0xd2, 0x4c, 0x9e, 0x97, 0x86, 0xf1, 0x9c, 0x61, 0x11, 0xe0, 0x43, 0x90, 0xa8, 0x99}
	quicSalt     = []byte{0x38, 0x76, 0x2c, 0xf7, 0xf5, 0x59, 0x34, 0xb3, 0x4d, 0x17, 0x9a, 0xe6, 0xa4, 0xc8, 0x0c, 0xad, 0xcc, 0xbb, 0x7f, 0x0a}
	initialSuite = &qtls.CipherSuiteTLS13{
		ID:     tls.TLS_AES_128_GCM_SHA256,
		KeyLen: 16,
		AEAD:   qtls.AEADAESGCMTLS13,
		Hash:   crypto.SHA256,
	}
)

//来自v2ray, 实际上就是先嗅探quic，之后再嗅探tls层。也就是说quic握手包是套在tls握手包外面的。
func SniffQUIC(b []byte) (sni string) {
	buffer := bytes.NewBuffer(b)
	typeByte, err := buffer.ReadByte()
	if err != nil {
		return
	}
	isLongHeader := typeByte&0x80 > 0
	if !isLongHeader || typeByte&0x40 == 0 {
		return
	}

	vb, err := buffer.ReadBytes(4)
	if err != nil {
		return
	}

	versionNumber := binary.BigEndian.Uint32(vb)

	if versionNumber != 0 && typeByte&0x40 == 0 {
		return
	} else if versionNumber != versionDraft29 && versionNumber != version1 {
		return
	}

	if (typeByte&0x30)>>4 != 0x0 {
		return
	}

	var destConnID []byte
	var l byte
	if l, err = buffer.ReadByte(); err != nil {
		return
	} else if destConnID = buffer.Next(int(l)); len(destConnID) != int(l) {
		return
	}

	if l, err := buffer.ReadByte(); err != nil {
		return
	} else if tmpBs := buffer.Next(int(l)); len(tmpBs) != int(l) {
		return
	}

	tokenLen, err := quicvarint.Read(buffer)
	if err != nil || tokenLen > uint64(len(b)) {
		return
	}

	if tmpBs := buffer.Next(int(tokenLen)); len(tmpBs) != int(tokenLen) {
		return
	}

	packetLen, err := quicvarint.Read(buffer)
	if err != nil {
		return
	}

	hdrLen := len(b) - int(buffer.Len())

	origPNBytes := make([]byte, 4)
	copy(origPNBytes, b[hdrLen:hdrLen+4])

	var salt []byte
	if versionNumber == version1 {
		salt = quicSalt
	} else {
		salt = quicSaltOld
	}
	initialSecret := hkdf.Extract(crypto.SHA256.New, destConnID, salt)
	secret := hkdfExpandLabel(crypto.SHA256, initialSecret, []byte{}, "client in", crypto.SHA256.Size())
	hpKey := hkdfExpandLabel(initialSuite.Hash, secret, []byte{}, "quic hp", initialSuite.KeyLen)
	block, err := aes.NewCipher(hpKey)
	if err != nil {
		return
	}

	wholeCacheBs := utils.GetPacket()
	defer utils.PutPacket(wholeCacheBs)

	cache := bytes.NewBuffer(wholeCacheBs)

	mask := cache.Next(int(block.BlockSize()))
	block.Encrypt(mask, b[hdrLen+4:hdrLen+4+16])
	b[0] ^= mask[0] & 0xf
	for i := range b[hdrLen : hdrLen+4] {
		b[hdrLen+i] ^= mask[i+1]
	}
	packetNumberLength := b[0]&0x3 + 1
	if packetNumberLength != 1 {
		return
	}
	var packetNumber uint32
	{
		n, err := buffer.ReadByte()
		if err != nil {
			return
		}
		packetNumber = uint32(n)
	}

	if packetNumber != 0 {
		return
	}

	extHdrLen := hdrLen + int(packetNumberLength)
	copy(b[extHdrLen:hdrLen+4], origPNBytes[packetNumberLength:])
	data := b[extHdrLen : int(packetLen)+hdrLen]

	key := hkdfExpandLabel(crypto.SHA256, secret, []byte{}, "quic key", 16)
	iv := hkdfExpandLabel(crypto.SHA256, secret, []byte{}, "quic iv", 12)
	cipher := qtls.AEADAESGCMTLS13(key, iv)
	nonce := cache.Next(int(cipher.NonceSize()))
	binary.BigEndian.PutUint64(nonce[len(nonce)-8:], uint64(packetNumber))
	decrypted, err := cipher.Open(b[extHdrLen:extHdrLen], nonce, data, b[:extHdrLen])
	if err != nil {
		return
	}
	buffer = bytes.NewBuffer(decrypted)
	frameType, err := buffer.ReadByte()
	if err != nil {
		return
	}
	if frameType != 0x6 {
		// not crypto frame
		return
	}
	if _, err := quicvarint.Read(buffer); err != nil {
		return
	}
	dataLen, err := quicvarint.Read(buffer)
	if err != nil {
		return
	}
	if dataLen > uint64(buffer.Len()) {
		return
	}
	frameData := buffer.Next(int(dataLen))
	if len(frameData) != int(dataLen) {
		return
	}

	cs := tlsLayer.ComSniff{Isclient: true}
	cs.CommonDetect(frameData, true, true)

	return cs.SniffedServerName
}

func hkdfExpandLabel(hash crypto.Hash, secret, context []byte, label string, length int) []byte {
	b := make([]byte, 3, 3+6+len(label)+1+len(context))
	binary.BigEndian.PutUint16(b, uint16(length))
	b[2] = uint8(6 + len(label))
	b = append(b, []byte("tls13 ")...)
	b = append(b, []byte(label)...)
	b = b[:3+6+len(label)+1]
	b[3+6+len(label)] = uint8(len(context))
	b = append(b, context...)

	out := make([]byte, length)
	n, err := hkdf.Expand(hash.New, secret, b).Read(out)
	if err != nil || n != length {
		panic("quic: HKDF-Expand-Label invocation failed unexpectedly")
	}
	return out
}
