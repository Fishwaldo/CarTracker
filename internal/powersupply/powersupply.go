package powersupply

import (
	"context"
	"time"

	"net/http"

	"github.com/Fishwaldo/CarTracker/internal"
	tm "github.com/Fishwaldo/CarTracker/internal/taskmanager"
	"github.com/Fishwaldo/CarTracker/internal/web"
	dcdcusb "github.com/Fishwaldo/go-dcdc200"
	"github.com/go-logr/logr"
	"github.com/Fishwaldo/go-taskmanager"
	"github.com/labstack/echo/v4"
	"github.com/sasha-s/go-deadlock"
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("powersupply.poll", 5)
	viper.SetDefault("powersupply.sim", true)
	internal.RegisterPlugin("powersupply", &PowerSupply)
}

type PowerSupplyS struct {
	PowerSupply dcdcusb.DcDcUSB
	ok          bool
	Params      dcdcusb.Params
	log         logr.Logger
	mx          deadlock.RWMutex
}

var PowerSupply PowerSupplyS

func (PS *PowerSupplyS) Start(log logr.Logger) {
	PS.mx.Lock()
	defer PS.mx.Unlock()
	PS.log = log
	PS.PowerSupply = dcdcusb.DcDcUSB{}
	//PS.PowerSupply.SetLogger(PS.log)
	if viper.GetBool("powersupply.sim") {
//		if err := dcdcusbsim.SetCaptureFile("dcdcusb.txt"); err != nil {
//			PS.log.Warn("Can't Open DCDCUSB Capture File for Simulation")
//		}
		PS.PowerSupply.Init(PS.log, true)
	} else {
		PS.PowerSupply.Init(PS.log, false);
	}
	ok, err := PS.PowerSupply.Scan()
	if err != nil {
		PS.log.Error(err, "Power Supply Scan Failed")
		return
	}
	if !ok {
		PS.log.Error(nil, "Power Supply Scan Returned False")
	}
	PS.ok = true
	ctx := context.Background()

	PS.PowerSupply.GetAllParam(ctx)
	fixedTimer1second, err := taskmanager.NewFixed(viper.GetDuration("powersupply.poll") * time.Second)
	if err != nil {
		PS.log.Error(err, "invalid interval")
	}

	err = tm.GetScheduler().Add(ctx, "PowerSupply", fixedTimer1second, PS.Poll)
	if err != nil {
		PS.log.Error(err, "Can't Initilize Scheduler for PowerSupply")
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
		PS.log.Error(nil, "Polling for PowerSupply - Not Ready")
		return
	}
	ctxnew, _ := context.WithTimeout(ctx, 1*time.Second)
	var err error
	if PS.Params, err = PS.PowerSupply.GetAllParam(ctxnew); err != nil {
		PS.log.Error(err, "GetAllParams Failed")
	}
	internal.ProcessUpdate("Power", PS.Params)
}

func (PS *PowerSupplyS) Publish(c echo.Context) error {
	PS.mx.Lock()
	defer PS.mx.Unlock()
	return c.JSON(http.StatusOK, PS.Params)
}

func (PS *PowerSupplyS) Process(string, interface{}) error {
	return nil;
}