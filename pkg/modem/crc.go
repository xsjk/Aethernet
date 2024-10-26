package modem

type CRC8Checker uint8

const gen = 0x07

var prod = func() [256]uint8 {
	prod := [256]uint8{}

	for i := range prod {
		crc := uint8(i)
		for j := 0; j < 8; j++ {
			if crc&0x80 != 0 {
				crc = (crc << 1) ^ gen
			} else {
				crc <<= 1
			}
		}
		prod[i] = crc
	}
	return prod
}()

func (c *CRC8Checker) Reset() {
	*c = 0
}

func (c *CRC8Checker) Update(b byte) {
	*c = CRC8Checker(prod[byte(*c)^b])
}

func (c CRC8Checker) Get() byte {
	return byte(c)
}

func (c CRC8Checker) Calculate(inputBytes []byte) byte {
	c.Reset()
	for _, b := range inputBytes {
		c.Update(b)
	}
	return byte(c)
}

func (c CRC8Checker) Check(inputBytes []byte, crc byte) bool {
	c.Reset()
	return c.Calculate(inputBytes) == crc
}

func (c CRC8Checker) CalculateBits(inputBits []bool) []bool {
	c.Reset()

	for i := 0; i < len(inputBits); i += 8 {
		var byte uint8
		for j := 0; j < 8 && (i+j) < len(inputBits); j++ {
			if inputBits[i+j] {
				byte |= 1 << (7 - j)
			}
		}
		c.Update(byte)
	}

	crcBits := make([]bool, 0, 8)
	for i := 7; i >= 0; i-- {
		crcBits = append(crcBits, ((byte(c)>>i)&1) == 1)
	}
	return crcBits
}
