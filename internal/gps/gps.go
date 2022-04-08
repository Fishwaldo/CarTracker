package gps

import (
	"bufio"
	"context"
	"io"
	"time"

	"github.com/Fishwaldo/CarTracker/internal"
	tm "github.com/Fishwaldo/CarTracker/internal/taskmanager"
	"github.com/go-logr/logr"
	"github.com/Fishwaldo/go-taskmanager"
	"github.com/adrianmo/go-nmea"
	"github.com/jacobsa/go-serial/serial"
	"github.com/sasha-s/go-deadlock"
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("gps.port", "/dev/ttyACM0")
	viper.SetDefault("gps.speed", 9600)
	viper.SetDefault("gps.poll", 5)
	internal.RegisterPlugin("gps", &GPS)
}

type GPSData struct {
	Latitude      float64
	Longitude     float64
	Altitude      float64
	Speed         float64
	Track         float64
	NumSatellites int64
	HDOP          float64
	mx            deadlock.RWMutex
}

type GpsS struct {
	logger  logr.Logger
	serial  io.ReadWriteCloser
	scanner *bufio.Scanner
	Data    GPSData
	stop    chan interface{}
}

var GPS GpsS

func (g *GpsS) Start(log logr.Logger) {
	g.stop = make(chan interface{})
	g.logger = log
	options := serial.OpenOptions{
		PortName:        viper.GetString("gps.port"),
		BaudRate:        viper.GetUint("gps.speed"),
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 4,
	}
	var err error
	if g.serial, err = serial.Open(options); err != nil {
		g.logger.Error(err, "Can't Open GPS Serial Port")
	}
	g.scanner = bufio.NewScanner(bufio.NewReader(g.serial))
	go g.Scan()

	fixedTimer, err := taskmanager.NewFixed(viper.GetDuration("gps.poll") * time.Second)
	if err != nil {
		g.logger.Error(err, "invalid interval")
	}
	err = tm.GetScheduler().Add(context.Background(), "GPS", fixedTimer, g.Poll)
	if err != nil {
		g.logger.Error(err, "Can't Initilize Scheduler for GPS")
	}
	g.logger.Info("Added GPS Polling Schedule")

}

func (g *GpsS) Scan() {
	for g.scanner.Scan() {
		switch {
		case <-g.stop:
			g.logger.Info("Exiting GPS Scanner")
			return
		default:
			scanText := g.scanner.Text()
			g.logger.V(1).Info("Scanning... ", "data", scanText)
			s, err := nmea.Parse(scanText)
			if err == nil {
				g.logger.V(1).Info("Got NMEA Type", "type", s.DataType())
				switch s.DataType() {
				case nmea.TypeRMC:
					data := s.(nmea.RMC)
					g.Data.mx.Lock()
					g.Data.Latitude = data.Latitude
					g.Data.Longitude = data.Longitude
					g.Data.mx.Unlock()
					g.logger.V(1).Info("RMC Data", "lat", data.Latitude, "long", data.Longitude)

				case nmea.TypeGGA:
					data := s.(nmea.GGA)
					g.Data.mx.Lock()
					g.Data.Altitude = data.Altitude
					g.Data.HDOP = data.HDOP
					g.Data.NumSatellites = data.NumSatellites
					g.Data.mx.Unlock()
					g.logger.V(1).Info("GAA Data", "altitide", data.Altitude, "hdop", data.HDOP, "satellites", data.NumSatellites)
				case nmea.TypeVTG:
					data := s.(nmea.VTG)
					g.Data.mx.Lock()
					g.Data.Track = data.TrueTrack
					g.Data.Speed = data.GroundSpeedKPH
					g.Data.mx.Unlock()
					g.logger.V(1).Info("VTG Data", "track", data.TrueTrack, "speed", data.GroundSpeedKPH)
				}
			} else {
				g.logger.Error(err, "GPS Read Failed", "scantext", scanText)
			}
		}
	}
}
func (g *GpsS) Stop() {
	g.Data.mx.Lock()
	defer g.Data.mx.Unlock()
	g.stop <-
	g.serial.Close()
}

func (g *GpsS) Poll(ctx context.Context) {
	g.Data.mx.RLock()
	internal.ProcessUpdate("gps", &g.Data)
	g.Data.mx.RUnlock()
}
