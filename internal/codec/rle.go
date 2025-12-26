package codec

type RLECodec struct{}

func (RLECodec) EncodeBlock(src []byte) ([]byte, uint8, error) {
	outByteLength := 2        // how long will output be
	srcByteLength := len(src) // how long is the input
	currentRunByte := src[0]  // keep track of the current repeating byte
	for _, srcByte := range src {
		if currentRunByte != srcByte {
			outByteLength += 2
			currentRunByte = srcByte
		}
	}
	currentRunLength := uint8(0)                  // keep track of the current length of run
	encodedMessage := make([]byte, outByteLength) // make a place to store the encoded message
	currentRunByte = src[0]                       // keep track of the current repeating byte
	currentEncodedLength := 0
	for srcIndex := range srcByteLength + 1 {
		if srcIndex == srcByteLength {
			encodedMessage[currentEncodedLength] = currentRunLength // write the number of reapeats
			currentEncodedLength++                                  // move to the next element
			encodedMessage[currentEncodedLength] = currentRunByte   // write the byte repeated
		} else if src[srcIndex] == currentRunByte && currentRunLength < uint8((1<<8)-1) {
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
	return encodedMessage, 0, nil
}

func (RLECodec) DecodeBlock(src []byte, padBits uint8) ([]byte, error) {
	outByteLength := 0        // how long will the output be
	srcByteLength := len(src) // how long is the input
	srcIndex := 0
	for srcIndex < srcByteLength { // for each (count, byte) pair
		outByteLength += int(src[srcIndex]) // count how many bytes will be added
		srcIndex += 2                       // jump to the next pair
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
