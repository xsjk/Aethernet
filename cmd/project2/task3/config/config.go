package config

import (
	"Aethernet/internel/utils"
	"Aethernet/pkg/async"
	"Aethernet/pkg/device"
	"Aethernet/pkg/fixed"
	"Aethernet/pkg/layers"
	"Aethernet/pkg/modem"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Device struct {
		DeviceName string  `yaml:"device_name"`
		SampleRate float64 `yaml:"sample_rate"`
	} `yaml:"device"`

	PhysicalLayer struct {
		BytePerFrame  int `yaml:"byte_per_frame"`
		FrameInterval int `yaml:"frame_interval"`

		Preamble struct {
			Amplitude float64 `yaml:"amplitude"`
			N         int     `yaml:"n"`
			Threshold float64 `yaml:"threshold"`
		} `yaml:"preamble"`

		Carrier struct {
			Amplitude float64 `yaml:"amplitude"`
			Size      int     `yaml:"size"`
		} `yaml:"carrier"`

		InputBufferSize   int `yaml:"input_buffer_size"`
		OutputBufferSize  int `yaml:"output_buffer_size"`
		ReceiveBufferSize int `yaml:"receive_buffer_size"`

		PowerMonitor struct {
			Threshold float64 `yaml:"threshold"`
			Window    int     `yaml:"window"`
		} `yaml:"power_monitor"`
	} `yaml:"physical_layer"`

	MACLayer struct {
		BytePerFrame     int           `yaml:"byte_per_frame"`
		AckTimeout       time.Duration `yaml:"ack_timeout"`
		MaxRetryAttempts int           `yaml:"max_retry_attempts"`
		BackoffTimer     struct {
			MinBackoff time.Duration `yaml:"min_backoff"`
			MaxBackoff time.Duration `yaml:"max_backoff"`
		} `yaml:"backoff_timer"`
		ReceiveBufferSize int `yaml:"receive_buffer_size"`
	} `yaml:"mac_layer"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func CreateMACLayer(config *Config) *layers.ReliableDataLinkLayer {

	var Preamble = modem.DigitalChripConfig{N: config.PhysicalLayer.Preamble.N, Amplitude: int32(config.PhysicalLayer.Preamble.Amplitude * 0x7fffffff)}.New()

	var Device = &device.ASIOMono{
		DeviceName: config.Device.DeviceName,
		SampleRate: config.Device.SampleRate,
	}

	var layer = layers.ReliableDataLinkLayer{
		BytePerFrame: config.MACLayer.BytePerFrame,
		PhysicalLayer: layers.PhysicalLayer{
			Device: Device,
			Decoder: layers.Decoder{
				Demodulator: modem.Demodulator{
					Preamble:                 Preamble,
					CarrierSize:              config.PhysicalLayer.Carrier.Size,
					BufferSize:               config.PhysicalLayer.ReceiveBufferSize,
					DemodulatePowerThreshold: fixed.FromFloat(config.PhysicalLayer.Preamble.Threshold),
				},
				BufferSize: config.PhysicalLayer.InputBufferSize,
			},
			Encoder: layers.Encoder{
				Modulator: modem.Modulator{
					Preamble:      Preamble,
					CarrierSize:   config.PhysicalLayer.Carrier.Size,
					BytePerFrame:  config.PhysicalLayer.BytePerFrame,
					FrameInterval: config.PhysicalLayer.FrameInterval,
					Amplitude:     int32(config.PhysicalLayer.Carrier.Amplitude * 0x7fffffff),
				},
				BufferSize: config.PhysicalLayer.OutputBufferSize,
			},
			PowerMonitor: layers.PowerMonitor{
				Threshold:  fixed.FromFloat(config.PhysicalLayer.PowerMonitor.Threshold),
				WindowSize: config.PhysicalLayer.PowerMonitor.Window,
			},
		},
		ACKTimeout: config.MACLayer.AckTimeout,
		MaxRetries: config.MACLayer.MaxRetryAttempts,
		BackoffTimer: layers.RandomBackoffTimer{
			MinDelay: config.MACLayer.BackoffTimer.MinBackoff,
			MaxDelay: config.MACLayer.BackoffTimer.MaxBackoff,
		},
		BufferSize: config.MACLayer.ReceiveBufferSize,
	}

	return &layer
}

func Main(myAddress, targetAddress layers.ReliableDataLinkAddress) {

	config, err := LoadConfig("config.yml")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	fmt.Printf("Config: %+v\n", config)

	inputFile := fmt.Sprintf("INPUT%dto%d.bin", myAddress, targetAddress)
	outputFile := fmt.Sprintf("OUTPUT%dto%d.bin", targetAddress, myAddress)

	fmt.Printf("Reading input from %s\n", inputFile)
	inputBytes, err := utils.ReadBinary[byte](inputFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	layer := CreateMACLayer(config)
	layer.Address = myAddress
	layer.Open()
	defer layer.Close()

	go func() {
		outputBytes := layer.Receive()
		fmt.Printf("Received %d bytes at %s\n", len(outputBytes), time.Now().Format(time.RFC3339))
		utils.WriteBinary(outputFile, outputBytes)
		fmt.Printf("Output written to %s\n", outputFile)
	}()

	fmt.Print("Press Enter to send the packet\n")
	<-async.EnterKey()

	go func() {
		startTime := time.Now()
		err := layer.Send(targetAddress, inputBytes)
		if err != nil {
			fmt.Printf("Error sending packet: %v\n", err)
		} else {
			fmt.Printf("Time taken to send packet: %v\n", time.Since(startTime))
		}
	}()

	<-async.EnterKey()
	fmt.Println("Exiting...")

}
