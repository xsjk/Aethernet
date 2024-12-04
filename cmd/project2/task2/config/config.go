package config

import (
	"Aethernet/pkg/device"
	"Aethernet/pkg/fixed"
	"Aethernet/pkg/layers"
	"Aethernet/pkg/modem"
	"time"
)

const (
	BYTE_PER_FRAME_PHY = 125 + 1
	BYTE_PER_FRAME_MAC = 125 * 50

	FRAME_INTERVAL = 1
	CARRIER_SIZE   = 2
	INTERVAL_SIZE  = 10

	INPUT_BUFFER_SIZE  = 10000
	OUTPUT_BUFFER_SIZE = 100

	POWER_THRESHOLD = 20

	POWER_MONITOR_THRESHOLD = 0.4
	POWER_MONITOR_WINDOW    = 10

	ACK_TIMEOUT        = 120 * time.Millisecond
	MAX_RETRY_ATTEMPTS = 3

	MIN_BACKOFF = 0
	MAX_BACKOFF = 1000 * time.Millisecond

	DATA_AMPLITUDE    = 0x5fffffff
	PRAMBLE_AMPLITUDE = 0x7fffffff

	// SAMPLE_RATE = 48000
	SAMPLE_RATE = 44100
)

var Preamble = modem.DigitalChripConfig{N: 4, Amplitude: 0x7fffffff}.New()

var Device = &device.ASIOMono{
	DeviceName: "ASIO4ALL v2",
	SampleRate: SAMPLE_RATE,
}

var Layer = layers.MACLayer{
	BytePerFrame: BYTE_PER_FRAME_MAC,
	PhysicalLayer: layers.PhysicalLayer{
		Device: Device,
		Decoder: layers.Decoder{
			Demodulator: modem.Demodulator{
				Preamble:                 Preamble,
				CarrierSize:              CARRIER_SIZE,
				DemodulatePowerThreshold: fixed.FromFloat(POWER_THRESHOLD),
			},
			BufferSize: INPUT_BUFFER_SIZE,
		},
		Encoder: layers.Encoder{
			Modulator: modem.Modulator{
				Preamble:      Preamble,
				CarrierSize:   CARRIER_SIZE,
				BytePerFrame:  BYTE_PER_FRAME_PHY,
				FrameInterval: FRAME_INTERVAL,
				Amplitude:     DATA_AMPLITUDE,
			},
			BufferSize: OUTPUT_BUFFER_SIZE,
		},
		PowerMonitor: layers.PowerMonitor{
			Threshold:  fixed.FromFloat(POWER_MONITOR_THRESHOLD),
			WindowSize: POWER_MONITOR_WINDOW,
		},
	},
	ACKTimeout: ACK_TIMEOUT,
	MaxRetries: MAX_RETRY_ATTEMPTS,
	BackoffTimer: layers.RandomBackoffTimer{
		MinDelay: MIN_BACKOFF,
		MaxDelay: MAX_BACKOFF,
	},
}
