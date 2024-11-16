package config

import (
	"Aethernet/internel/utils"
	"Aethernet/pkg/async"
	"Aethernet/pkg/device"
	"Aethernet/pkg/fixed"
	"Aethernet/pkg/layers"
	"Aethernet/pkg/modem"
	"fmt"
	"time"
)

const (
	BYTE_PER_FRAME = 125
	FRAME_INTERVAL = 256
	CARRIER_SIZE   = 3
	INTERVAL_SIZE  = 10
	PAYLOAD_SIZE   = 32

	INPUT_BUFFER_SIZE  = 10000
	OUTPUT_BUFFER_SIZE = 1

	POWER_THRESHOLD      = 30
	CORRECTION_THRESHOLD = 0.8

	POWER_MONITOR_THRESHOLD = 0.4
	POWER_MONITOR_WINDOW    = 10

	ACK_TIMEOUT        = 1000 * time.Millisecond
	MAX_RETRY_ATTEMPTS = 5

	MIN_BACKOFF = 0
	MAX_BACKOFF = 100 * time.Millisecond

	DATA_AMPLITUDE    = 0x7fffffff
	PRAMBLE_AMPLITUDE = 0x7fffffff

	// SAMPLE_RATE = 48000
	SAMPLE_RATE = 44100
)

var Preamble = modem.DigitalChripConfig{N: 5, Amplitude: 0x7fffffff}.New()

var Device = &device.ASIOMono{
	DeviceName: "ASIO4ALL v2",
	SampleRate: SAMPLE_RATE,
}

var Layer = layers.MACLayer{
	PhysicalLayer: layers.PhysicalLayer{
		Device: Device,
		Decoder: layers.Decoder{
			Demodulator: modem.Demodulator{
				Preamble:                 Preamble,
				CarrierSize:              CARRIER_SIZE,
				CorrectionThreshold:      fixed.FromFloat(CORRECTION_THRESHOLD),
				DemodulatePowerThreshold: fixed.FromFloat(POWER_THRESHOLD),
				OutputChan:               make(chan []byte, 10),
			},
			BufferSize: INPUT_BUFFER_SIZE,
		},
		Encoder: layers.Encoder{
			Modulator: modem.Modulator{
				Preamble:      Preamble,
				CarrierSize:   CARRIER_SIZE,
				BytePerFrame:  BYTE_PER_FRAME,
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
	OutputChan: make(chan []byte, 10),
}

func Main(myAddress, targetAddress layers.MACAddress) {

	inputFile := fmt.Sprintf("INPUT%dto%d.bin", myAddress, targetAddress)
	outputFile := fmt.Sprintf("OUTPUT%dto%d.bin", targetAddress, myAddress)

	fmt.Printf("Reading input from %s\n", inputFile)
	inputBytes, err := utils.ReadBinary[byte](inputFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	Layer.Address = myAddress
	Layer.Open()
	defer Layer.Close()

	go func() {
		outputBytes := Layer.Receive()
		fmt.Printf("Received %d bytes at %s\n", len(outputBytes), time.Now().Format(time.RFC3339))
		utils.WriteBinary(outputFile, outputBytes)
		fmt.Printf("Output written to %s\n", outputFile)
	}()

	fmt.Print("Press Enter to send the packet\n")
	<-async.EnterKey()

	go func() {
		startTime := time.Now()
		err := Layer.Send(targetAddress, inputBytes)
		if err != nil {
			fmt.Printf("Error sending packet: %v\n", err)
		} else {
			fmt.Printf("Time taken to send packet: %v\n", time.Since(startTime))
		}
	}()

	<-async.EnterKey()
	fmt.Println("Exiting...")

}
