package config

import (
	"Aethernet/pkg/device"
	"Aethernet/pkg/fixed"
	"Aethernet/pkg/iface"
	"Aethernet/pkg/layers"
	"Aethernet/pkg/modem"
	"fmt"
	"os"
	"strings"

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

		InputBufferSize  int `yaml:"input_buffer_size"`
		OutputBufferSize int `yaml:"output_buffer_size"`

		PowerMonitor struct {
			Threshold float64 `yaml:"threshold"`
			Window    int     `yaml:"window"`
		} `yaml:"power_monitor"`
	} `yaml:"physical_layer"`

	MACLayer struct {
		Address int `yaml:"address"`
	} `yaml:"mac_layer"`

	Iface struct {
		Type   string `yaml:"type"`
		IP     string `yaml:"ip"`
		Name   string `yaml:"name"`
		Filter string `yaml:"filter"`
	}
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

func CreateNaiveDataLinkLayer(config *Config) *layers.NaiveDataLinkLayer {

	var Preamble = modem.DigitalChripConfig{N: config.PhysicalLayer.Preamble.N, Amplitude: int32(config.PhysicalLayer.Preamble.Amplitude * 0x7fffffff)}.New()

	var Device = &device.ASIOMono{
		DeviceName: config.Device.DeviceName,
		SampleRate: config.Device.SampleRate,
	}

	return &layers.NaiveDataLinkLayer{
		PhysicalLayer: layers.PhysicalLayer{
			Device: Device,
			Decoder: layers.Decoder{
				Demodulator: modem.Demodulator{
					Preamble:                 Preamble,
					CarrierSize:              config.PhysicalLayer.Carrier.Size,
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
		Address: byte(config.MACLayer.Address),
	}
}

func OpenInterface(config *Config) (iface.Interface, error) {
	switch strings.ToLower(config.Iface.Type) {
	case "tun":
		return iface.OpenTUN(config.Iface.IP, config.Iface.Name)
	case "tap":
		return iface.OpenTAP(config.Iface.IP)
	case "pcap":
		return iface.OpenPCAP(config.Iface.Name, config.Iface.Filter)
	default:
		return nil, fmt.Errorf("Unknown interface type: %s, expected 'tun', 'tap' or 'pcap'", config.Iface.Type)
	}
}
