package audiostream

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"math"
)

const (
	DataURIPrefix = "data:audio/vnd.shazam.sig;base64,"
	Magic1        = 0xCAFE2580
	Magic2        = 0x94119C00
)

// RawSignatureHeader represents the header structure for Shazam signatures
type RawSignatureHeader struct {
	Magic1                       uint32
	CRC32                        uint32
	SizeMinusHeader              uint32
	Magic2                       uint32
	Void1                        [3]uint32
	ShiftedSampleRateID          uint32
	Void2                        [2]uint32
	NumberSamplesPlusDividedRate uint32
	FixedValue                   uint32
}

// FrequencyPeak represents a single frequency peak in the audio signature
type FrequencyPeak struct {
	FFTPassNumber             int
	PeakMagnitude             int
	CorrectedPeakFrequencyBin int
	SampleRateHz              int
}

// GetFrequencyHz converts the frequency bin to Hz
func (fp *FrequencyPeak) GetFrequencyHz() float64 {
	return float64(fp.CorrectedPeakFrequencyBin) * (float64(fp.SampleRateHz) / 2 / 1024 / 64)
}

// GetAmplitudePCM calculates the amplitude in PCM format
func (fp *FrequencyPeak) GetAmplitudePCM() float64 {
	return math.Sqrt(math.Exp(float64(fp.PeakMagnitude-6144)/1477.3)*(1<<17)/2) / 1024
}

// GetSeconds calculates the time position in seconds
func (fp *FrequencyPeak) GetSeconds() float64 {
	return float64(fp.FFTPassNumber*128) / float64(fp.SampleRateHz)
}

// DecodedMessage represents the decoded Shazam signature message
type DecodedMessage struct {
	SampleRateHz              int
	NumberSamples             int
	FrequencyBandToSoundPeaks map[FrequencyBand][]FrequencyPeak
}

// DecodeFromBinary decodes a binary signature into a DecodedMessage
func DecodeFromBinary(data []byte) (*DecodedMessage, error) {
	msg := &DecodedMessage{
		FrequencyBandToSoundPeaks: make(map[FrequencyBand][]FrequencyPeak),
	}

	buf := bytes.NewReader(data)
	buf.Seek(8, io.SeekStart)
	_, err := io.ReadAll(buf)
	if err != nil {
		return nil, err
	}

	buf.Seek(0, io.SeekStart)
	header := &RawSignatureHeader{}
	if err := binary.Read(buf, binary.LittleEndian, header); err != nil {
		return nil, err
	}

	if header.Magic1 != Magic1 {
		return nil, fmt.Errorf("invalid magic1: %x", header.Magic1)
	}
	if header.SizeMinusHeader != uint32(len(data)-48) {
		return nil, fmt.Errorf("invalid size: %d", header.SizeMinusHeader)
	}
	if header.Magic2 != Magic2 {
		return nil, fmt.Errorf("invalid magic2: %x", header.Magic2)
	}

	msg.SampleRateHz = int(header.ShiftedSampleRateID >> 27)
	msg.NumberSamples = int(float64(header.NumberSamplesPlusDividedRate) - float64(msg.SampleRateHz)*0.24)

	// Read the type-length-value sequence
	var tlvHeader [8]byte
	for {
		if _, err := buf.Read(tlvHeader[:]); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		frequencyBandID := binary.LittleEndian.Uint32(tlvHeader[:4])
		frequencyPeaksSize := binary.LittleEndian.Uint32(tlvHeader[4:])
		frequencyPeaksPadding := -int(frequencyPeaksSize) % 4

		peaksBuf := make([]byte, frequencyPeaksSize)
		if _, err := buf.Read(peaksBuf); err != nil {
			return nil, err
		}
		buf.Seek(int64(frequencyPeaksPadding), io.SeekCurrent)

		frequencyBand := FrequencyBand(frequencyBandID - 0x60030040)
		fftPassNumber := 0
		peaksReader := bytes.NewReader(peaksBuf)

		for {
			rawFFTPass := make([]byte, 1)
			if _, err := peaksReader.Read(rawFFTPass); err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}

			fftPassOffset := int(rawFFTPass[0])
			if fftPassOffset == 0xFF {
				var newFFTPassNumber uint32
				if err := binary.Read(peaksReader, binary.LittleEndian, &newFFTPassNumber); err != nil {
					return nil, err
				}
				fftPassNumber = int(newFFTPassNumber)
				continue
			}

			fftPassNumber += fftPassOffset
			var peakMagnitude uint16
			var correctedPeakFrequencyBin uint16

			if err := binary.Read(peaksReader, binary.LittleEndian, &peakMagnitude); err != nil {
				return nil, err
			}
			if err := binary.Read(peaksReader, binary.LittleEndian, &correctedPeakFrequencyBin); err != nil {
				return nil, err
			}

			msg.FrequencyBandToSoundPeaks[frequencyBand] = append(msg.FrequencyBandToSoundPeaks[frequencyBand],
				FrequencyPeak{
					FFTPassNumber:             fftPassNumber,
					PeakMagnitude:             int(peakMagnitude),
					CorrectedPeakFrequencyBin: int(correctedPeakFrequencyBin),
					SampleRateHz:              msg.SampleRateHz,
				})
		}
	}

	return msg, nil
}

// EncodeToBinary encodes a DecodedMessage to binary format
func (msg *DecodedMessage) EncodeToBinary() ([]byte, error) {
	header := &RawSignatureHeader{
		Magic1:                       Magic1,
		Magic2:                       Magic2,
		ShiftedSampleRateID:          uint32(msg.SampleRateHz) << 27,
		FixedValue:                   (15 << 19) + 0x40000,
		NumberSamplesPlusDividedRate: uint32(float64(msg.NumberSamples) + float64(msg.SampleRateHz)*0.24),
	}

	contentsBuf := new(bytes.Buffer)
	for frequencyBand, frequencyPeaks := range msg.FrequencyBandToSoundPeaks {
		peaksBuf := new(bytes.Buffer)
		fftPassNumber := 0

		for _, peak := range frequencyPeaks {
			if peak.FFTPassNumber-fftPassNumber >= 255 {
				peaksBuf.WriteByte(0xFF)
				binary.Write(peaksBuf, binary.LittleEndian, uint32(peak.FFTPassNumber))
				fftPassNumber = peak.FFTPassNumber
			}

			peaksBuf.WriteByte(byte(peak.FFTPassNumber - fftPassNumber))
			binary.Write(peaksBuf, binary.LittleEndian, uint16(peak.PeakMagnitude))
			binary.Write(peaksBuf, binary.LittleEndian, uint16(peak.CorrectedPeakFrequencyBin))
			fftPassNumber = peak.FFTPassNumber
		}

		binary.Write(contentsBuf, binary.LittleEndian, uint32(0x60030040+int(frequencyBand)))
		binary.Write(contentsBuf, binary.LittleEndian, uint32(peaksBuf.Len()))
		contentsBuf.Write(peaksBuf.Bytes())
		contentsBuf.Write(make([]byte, -peaksBuf.Len()%4))
	}

	header.SizeMinusHeader = uint32(contentsBuf.Len() + 8)
	finalBuf := new(bytes.Buffer)
	binary.Write(finalBuf, binary.LittleEndian, header)
	binary.Write(finalBuf, binary.LittleEndian, uint32(0x40000000))
	binary.Write(finalBuf, binary.LittleEndian, uint32(contentsBuf.Len()+8))
	finalBuf.Write(contentsBuf.Bytes())

	// Calculate and write CRC32
	checkSummableData := finalBuf.Bytes()[8:]
	header.CRC32 = crc32.ChecksumIEEE(checkSummableData)

	// Create a new buffer with the updated CRC32
	updatedBuf := new(bytes.Buffer)
	binary.Write(updatedBuf, binary.LittleEndian, header)
	updatedBuf.Write(finalBuf.Bytes()[12:])

	return updatedBuf.Bytes(), nil
}

// EncodeToURI encodes the signature to a data URI
func (msg *DecodedMessage) EncodeToURI() (string, error) {
	binary, err := msg.EncodeToBinary()
	if err != nil {
		return "", err
	}
	return DataURIPrefix + base64.StdEncoding.EncodeToString(binary), nil
}
