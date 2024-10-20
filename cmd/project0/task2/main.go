package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"time"

	"Aethernet/internel/callbacks"
	"Aethernet/internel/utils"

	"github.com/xsjk/go-asio"
)

func parse_args() (float64, float64, string, string, float64) {
	amplifier_var := flag.Float64("a", 1.0, "Set the amplifier")
	input_path_var := flag.String("i", "input.bin", "Set the path for input binary")
	output_path_var := flag.String("o", "output.bin", "Set the path for output binary")
	sampleRate_var := flag.Float64("s", 44100.0, "Set the sample rate")
	duration_var := flag.Float64("t", 0, "Set the duration")
	flag.Parse()
	amplifier := *amplifier_var
	input_path := *input_path_var
	output_path := *output_path_var
	sampleRate := *sampleRate_var
	duration := *duration_var
	fmt.Printf("Amplifier: %f\nInputPath: %s\nOutputPath: %s\nSample Rate: %f\nDuration: %f\n", amplifier, input_path, output_path, sampleRate, duration)
	return sampleRate, amplifier, input_path, output_path, duration
}

func main() {

	sampleRate, amplifier, input_path, output_path, duration := parse_args()

	var pause func()
	if duration > 0 {
		pause = func() { time.Sleep(time.Duration(duration) * time.Second) }
	} else {
		pause = func() {
			fmt.Println("press enter to continue...")
			bufio.NewReader(os.Stdin).ReadBytes('\n')
		}
	}

	// read the audio file
	audio, _ := utils.ReadBinary[int32](input_path)
	if len(audio) == 0 {
		fmt.Println("No audio data")
	} else {
		fmt.Printf("Audio data loaded (%d samples)\n", len(audio))
	}

	d := asio.Device{}
	d.Load("ASIO4ALL v2")
	defer d.Unload()
	defer time.Sleep(100 * time.Millisecond)
	d.SetSampleRate(sampleRate)
	d.Open()
	defer d.Close()

	player := callbacks.Player{Track: audio}
	recorder := callbacks.Recorder{}
	d.Start(func(in, out [][]int32) {
		recorder.Update(in, out)
		player.Update(in, out)
	})
	pause()
	d.Stop()

	utils.WriteBinary(output_path, recorder.Track)

	for i := range recorder.Track {
		recorder.Track[i] = int32(max(min(int64(recorder.Track[i])*int64(amplifier), 0x7fffffff), -0x80000000))
	}

	player = callbacks.Player{Track: recorder.Track}
	d.Start(player.Update)
	pause()
	d.Stop()

}
