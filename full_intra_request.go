package rtcp

import (
	"encoding/binary"
	"fmt"
)

// A FIREntry is a (SSRC, seqno) pair, as carried by FullIntraRequest.
type FIREntry struct {
	SSRC           uint32
	SequenceNumber uint8
}

// The FullIntraRequest packet is used to reliably request an Intra frame
// in a video stream.  See RFC 5104 Section 3.5.1.  This is not for loss
// recovery, which should use PictureLossIndication (PLI) instead.
type FullIntraRequest struct {
	SenderSSRC uint32
	MediaSSRC  uint32

	FIR []FIREntry
}

const (
	firOffset = headerSize + 8
)

var _ Packet = (*FullIntraRequest)(nil)

// MarshalSize returns the size of the packet once marshaled.
func (p FullIntraRequest) MarshalSize() int {
	return firOffset + len(p.FIR)*8
}

// MarshalTo encodes the FullIntraRequest
func (p FullIntraRequest) MarshalTo(buf []byte) (int, error) {
	_, err := p.Header().MarshalTo(buf)
	if err != nil {
		return 0, err
	}

	binary.BigEndian.PutUint32(buf[headerSize:], p.SenderSSRC)
	binary.BigEndian.PutUint32(buf[headerSize+4:], p.MediaSSRC)

	for i, fir := range p.FIR {
		binary.BigEndian.PutUint32(buf[firOffset+8*i:], fir.SSRC)
		buf[firOffset+8*i+4] = fir.SequenceNumber
	}

	return firOffset + len(p.FIR)*8, nil
}

// Marshal encodes the FullIntraRequest in binary
func (p FullIntraRequest) Marshal() ([]byte, error) {
	buf := make([]byte, p.MarshalSize())

	_, err := p.MarshalTo(buf)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// Unmarshal decodes the TransportLayerNack
func (p *FullIntraRequest) Unmarshal(rawPacket []byte) error {
	if len(rawPacket) < (headerSize + ssrcLength) {
		return errPacketTooShort
	}

	var h Header
	if err := h.Unmarshal(rawPacket); err != nil {
		return err
	}

	if len(rawPacket) < (headerSize + int(4*h.Length)) {
		return errPacketTooShort
	}

	if h.Type != TypePayloadSpecificFeedback || h.Count != FormatFIR {
		return errWrongType
	}

	p.SenderSSRC = binary.BigEndian.Uint32(rawPacket[headerSize:])
	p.MediaSSRC = binary.BigEndian.Uint32(rawPacket[headerSize+ssrcLength:])

	for i := firOffset; i < (headerSize + int(h.Length*4)); i += 8 {
		p.FIR = append(p.FIR, FIREntry{
			binary.BigEndian.Uint32(rawPacket[i:]),
			rawPacket[i+4],
		})
	}
	return nil
}

// Header returns the Header associated with this packet.
func (p *FullIntraRequest) Header() Header {
	return Header{
		Count:  FormatFIR,
		Type:   TypePayloadSpecificFeedback,
		Length: uint16((p.MarshalSize() / 4) - 1),
	}
}

func (p *FullIntraRequest) String() string {
	out := fmt.Sprintf("FullIntraRequest %x %x",
		p.SenderSSRC, p.MediaSSRC)
	for _, e := range p.FIR {
		out += fmt.Sprintf(" (%x %v)", e.SSRC, e.SequenceNumber)
	}
	return out
}

// DestinationSSRC returns an array of SSRC values that this packet refers to.
func (p *FullIntraRequest) DestinationSSRC() []uint32 {
	ssrcs := make([]uint32, 0, len(p.FIR))
	for _, entry := range p.FIR {
		ssrcs = append(ssrcs, entry.SSRC)
	}
	return ssrcs
}
