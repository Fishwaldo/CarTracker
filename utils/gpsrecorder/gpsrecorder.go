package main

import (
	"bufio"
	"log"
	"os"
	"os/signal"
	"syscall"
	"fmt"

	"github.com/jacobsa/go-serial/serial"
)

func main() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	f, err := os.Create("./gps.txt")
	if err != nil {
		log.Panicf("Can't Open GPS File: %s", err)
	}
	defer f.Close()
	options := serial.OpenOptions{
		PortName:       "/dev/ttyACM0",
		BaudRate:        9600,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 4,
	}
	serialport, err := serial.Open(options);
	if err != nil {
		log.Printf("Can't Open GPS Serial Port: %s", err)
		return;
	}
	defer serialport.Close()
	fmt.Printf("Capturing Serial Port Data\n")
	fmt.Printf("Press Ctrl-C to exit\n")
	scanner := bufio.NewScanner(bufio.NewReader(serialport))
	for scanner.Scan() {
		scanText := scanner.Text()
		f.WriteString(scanText)
		f.WriteString("\n")
		select {
		case <- signalChan:
			log.Printf("Caught Signal - Exiting...")
			return
		default:
		}
		fmt.Printf(".")
	}
}