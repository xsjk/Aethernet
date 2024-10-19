package utils

const CRC_BITS = 8

func CalCRC8(inputBits []bool) []bool {
	const polynomial = 0x07 // CRC-8 polynomial x^8 + x^2 + x + 1
	var crc uint8

	for i := 0; i < len(inputBits); i += 8 {
		var byte uint8
		for j := 0; j < 8 && (i+j) < len(inputBits); j++ {
			if inputBits[i+j] {
				byte |= 1 << (7 - j)
			}
		}
		crc ^= byte
		for k := 0; k < 8; k++ {
			if crc&0x80 != 0 {
				crc = (crc << 1) ^ polynomial
			} else {
				crc <<= 1
			}
		}
	}

	crcBits := make([]bool, 0, 8)
	for i := 7; i >= 0; i-- {
		crcBits = append(crcBits, ((crc>>i)&1) == 1)
	}
	return crcBits
}
