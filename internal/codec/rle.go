package codec

import "fmt"

const (
	maxRunLength uint8 = 255
	pairSize     int   = 2
)

type RLECodec struct{}

func (RLECodec) EncodeBlock(src []byte) ([]byte, error) {
	srcByteLength := len(src) // how long is the input
	if srcByteLength == 0 {
		return []byte{}, nil
	}
	outByteLength := pairSize    // how long will output be
	currentRunLength := uint8(0) // keep track of the current length of run
	currentRunByte := src[0]     // keep track of the current repeating currentRunByte
	for _, srcByte := range src {
		if currentRunByte != srcByte || currentRunLength >= maxRunLength {
			outByteLength += pairSize
			currentRunByte = srcByte
			currentRunLength = 1
		} else {
			currentRunLength += 1
		}
	}
	currentRunLength = 0                          // keep track of the current length of run
	encodedMessage := make([]byte, outByteLength) // make a place to store the encoded message
	currentRunByte = src[0]                       // keep track of the current repeating byte
	currentEncodedLength := 0
	for srcIndex := range srcByteLength + 1 {
		if srcIndex == srcByteLength {
			encodedMessage[currentEncodedLength] = currentRunLength // write the number of reapeats
			currentEncodedLength++                                  // move to the next element
			encodedMessage[currentEncodedLength] = currentRunByte   // write the byte repeated
		} else if src[srcIndex] == currentRunByte && currentRunLength < maxRunLength {
			currentRunLength += 1 // increment counter if byte repeats and not over the repeat limit
		} else {
			encodedMessage[currentEncodedLength] = currentRunLength // write the number of reapeats
			currentEncodedLength++                                  // move to the next element
			encodedMessage[currentEncodedLength] = currentRunByte   // write the byte repeated
			currentEncodedLength++                                  // move to the next element
			currentRunLength = 1                                    // start your counter over
			currentRunByte = src[srcIndex]                          // reset the byte tracked
		}
	}
	return encodedMessage, nil
}

func (RLECodec) DecodeBlock(src []byte) ([]byte, error) {
	outByteLength := 0        // how long will the output be
	srcByteLength := len(src) // how long is the input
	if srcByteLength == 0 {
		return []byte{}, nil
	}
	if srcByteLength%pairSize != 0 {
		return []byte{}, fmt.Errorf("malformed RLE input for decoding")
	}
	srcIndex := 0
	for srcIndex < srcByteLength { // for each (count, byte) pair
		outByteLength += int(src[srcIndex]) // count how many bytes will be added
		srcIndex += pairSize                // jump to the next pair
	}
	decodedMessage := make([]byte, outByteLength) // make the array you need for output
	currentOutIndex := 0                          // keep track of where you are in the output
	srcIndex = 0                                  // keep track of where you are in the input
	for srcIndex < srcByteLength {
		for range src[srcIndex] {
			decodedMessage[currentOutIndex] = src[srcIndex+1] // add the byte, count times
			currentOutIndex++
		}
		srcIndex += 2 // jump to the next pair
	}
	return decodedMessage, nil
}

func (RLECodec) IsLossless() bool {
	return true
}
