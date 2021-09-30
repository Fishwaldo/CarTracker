package powersupply

import (
	"context"
	"time"

	"net/http"

	"github.com/Fishwaldo/CarTracker/internal"
	"github.com/Fishwaldo/CarTracker/internal/natsconnection"
	tm "github.com/Fishwaldo/CarTracker/internal/taskmanager"
	"github.com/Fishwaldo/CarTracker/internal/web"
	dcdcusb "github.com/Fishwaldo/go-dcdc200"
	"github.com/Fishwaldo/go-logadapter"
	"github.com/Fishwaldo/go-taskmanager"
	"github.com/labstack/echo/v4"
	"github.com/sasha-s/go-deadlock"
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("powersupply.poll", 5)
	internal.RegisterPlugin("powersupply", &PowerSupply)
}

type PowerSupplyS struct {
	PowerSupply dcdcusb.DcDcUSB
	ok          bool
	Params      dcdcusb.Params
	log         logadapter.Logger
	mx          deadlock.RWMutex
}

var PowerSupply PowerSupplyS

func (PS *PowerSupplyS) Start(log logadapter.Logger) {
	PS.mx.Lock()
	defer PS.mx.Unlock()
	PS.log = log
	PS.PowerSupply = dcdcusb.DcDcUSB{}
	PS.PowerSupply.SetLogger(PS.log)
	PS.PowerSupply.Init()
	ok, err := PS.PowerSupply.Scan()
	if err != nil {
		PS.log.Warn("Power Supply Scan Failed: %s", err)
		return
	}
	if !ok {
		PS.log.Warn("Power Supply Scan Returned False")
	}
	PS.ok = true
	ctx := context.Background()

	PS.PowerSupply.GetAllParam(ctx)
	fixedTimer1second, err := taskmanager.NewFixed(viper.GetDuration("powersupply.poll") * time.Second)
	if err != nil {
		PS.log.Panic("invalid interval: %s", err.Error())
	}

	err = tm.GetScheduler().Add(ctx, "PowerSupply", fixedTimer1second, PS.Poll)
	if err != nil {
		PS.log.Panic("Can't Initilize Scheduler for PowerSupply: %s", err)
	}
	PS.log.Info("Added PowerSupply Polling Schedule")
	web.Web.GetEchoServer().GET("/power", PS.Publish)
	//echo.GET("/", PS.Publish)
	//.Echo.GET("/", PS.Publish)
}

func (PS *PowerSupplyS) Stop() {
	PS.mx.Lock()
	defer PS.mx.Unlock()
	tm.GetScheduler().Stop("PowerSupply")
	PS.PowerSupply.Close()
	PS.ok = false
}

func (PS *PowerSupplyS) Poll(ctx context.Context) {
	PS.mx.Lock()
	defer PS.mx.Unlock()
	if !PS.ok {
		PS.log.Warn("Polling for PowerSupply - Not Ready")
		return
	}
	ctxnew, _ := context.WithTimeout(ctx, 1*time.Second)
	PS.Params = PS.PowerSupply.GetAllParam(ctxnew)
	natsconnection.Nats.SendStats("Power", PS.Params)
}

func (PS *PowerSupplyS) Publish(c echo.Context) error {
	PS.mx.Lock()
	defer PS.mx.Unlock()
	return c.JSON(http.StatusOK, PS.Params)
}
