package modem

type CRC8Checker struct {
	gen  uint8
	prod [256]uint8

	q uint8
}

func MakeCRC8Checker(gen uint8) CRC8Checker {
	prod := [256]uint8{}

	for i := 0; i < 256; i++ {
		crc := uint8(i)
		for j := 0; j < 8; j++ {
			if crc&0x80 != 0 {
				crc = (crc << 1) ^ gen
			} else {
				crc <<= 1
			}
		}
		prod[i] = uint8(crc)
	}

	return CRC8Checker{
		gen:  gen,
		prod: prod,
	}
}

func (c *CRC8Checker) Reset() {
	c.q = 0
}

func (c *CRC8Checker) Update(b byte) {
	c.q = c.prod[c.q^b]
}

func (c CRC8Checker) Get() byte {
	return c.q
}

func (c CRC8Checker) Calculate(inputBytes []byte) byte {
	c.Reset()
	for _, b := range inputBytes {
		c.Update(b)
	}
	return c.q
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
		crcBits = append(crcBits, ((int(c.q)>>i)&1) == 1)
	}
	return crcBits
}
