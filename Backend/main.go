package main

import (
	"fmt"
	"net"
	"os"
	"runtime/pprof"
	"strings"
	"time"
	"tmaxsrv/log"
	"tmaxsrv/svc"
	"tmaxsrv/util"

	"github.com/kardianos/service"
)

var Version = "1.0.0"

const (
	INSTANCE_PORT = 9292
)

var (
	OUR_USED_APP_NAMES []string = []string{"BootCommander.exe"}
)

func killZombieApp() error {
	for _, app := range OUR_USED_APP_NAMES {
		if err := util.KillApp(app); err == nil {
			fmt.Println("wait 5 seconds...")
			time.Sleep(5 * time.Second)
			fmt.Println("done")
		}
	}
	return nil
}

type program struct{}

func (p *program) Start(s service.Service) error {
	fmt.Println("服务运行...")
	go p.run()

	return nil
}
func (p *program) run() {
	// 具体的服务实现
	killZombieApp()
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", INSTANCE_PORT))
	if err != nil {
		if strings.Contains(err.Error(), "in use") {
			// optionally send command line arguments to the other instance
			fmt.Fprintln(os.Stderr, "Already running.")
			return
		} else {
			panic(err)
		}
	}
	defer listener.Close()

	runMode := os.Getenv("RUN_MODE")
	if runMode == "DEV" {
		cpuf, err := os.Create("cpu_profile")
		if err != nil {
			log.Log.Error(err)
		}
		pprof.StartCPUProfile(cpuf)
		defer pprof.StopCPUProfile()
	}

	s := svc.NewScaleMgr() // srvMgr should be set via SetScaleMgr
	// channel to inform application to quit
	quitch := make(chan bool)
	h := *svc.NewSrvMgr(s, quitch)

	s.SetSrvMsg(&h)
	go s.Run()

	go h.Run()
	ws := svc.NewWsServer()
	go ws.Start(&h)

	for {
		fmt.Println("Service is running")
		time.Sleep(5 * time.Second)
	}

}
func (p *program) Stop(s service.Service) error {
	return nil
}

func main() {
	srvConfig := &service.Config{
		Name:        "TmaxService",
		DisplayName: "TmaxService",
		Description: "TmaxService",
	}
	prg := &program{}
	s, err := service.New(prg, srvConfig)
	if err != nil {
		fmt.Println(err)
	}
	if len(os.Args) > 1 {
		serviceAction := os.Args[1]
		switch serviceAction {
		case "install":
			err := s.Install()
			if err != nil {
				fmt.Println("install fail: ", err.Error())
			} else {
				fmt.Println("install ok")
			}
			return
		case "uninstall":
			err := s.Uninstall()
			if err != nil {
				fmt.Println("uninstall fail: ", err.Error())
			} else {
				fmt.Println("uninstall ok")
			}
			return
		case "start":
			err := s.Start()
			if err != nil {
				fmt.Println("running fail: ", err.Error())
			} else {
				fmt.Println("running ok")
			}
			return
		case "stop":
			err := s.Stop()
			if err != nil {
				fmt.Println("stop fail: ", err.Error())
			} else {
				fmt.Println("stop ok")
			}
			return
		}
	}

	err = s.Run()
	if err != nil {
		fmt.Println("service.Run error, fallback to direct run:", err)
		prg.run()
	}
}

//下面是不用服务启动的

// package main

// import (
// 	"fmt"
// 	"net"
// 	_ "net/http/pprof"
// 	"os"
// 	"os/exec"
// 	"os/signal"
// 	"runtime/pprof"
// 	"strings"
// 	"syscall"
// 	"time"
// 	"tmaxsrv/build"
// 	"tmaxsrv/log"
// 	"tmaxsrv/svc"
// 	"tmaxsrv/util"
// )

// var Version = "1.0.0"

// const (
// 	INSTANCE_PORT = 9292
// )

// var (
// 	OUR_USED_APP_NAMES []string = []string{"BootCommander.exe"}
// )

// func killZombieApp() error {
// 	for _, app := range OUR_USED_APP_NAMES {
// 		if err := util.KillApp(app); err == nil {
// 			fmt.Println("wait 5 seconds...")
// 			time.Sleep(5 * time.Second)
// 			fmt.Println("done")
// 		}
// 	}
// 	return nil
// }

// func main() {
// 	fmt.Println("Version:\t", Version)
// 	fmt.Println("build.Time:\t", build.Time)
// 	fmt.Println("build.User:\t", build.User)

// 	killZombieApp()

// 	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", INSTANCE_PORT))
// 	if err != nil {
// 		if strings.Contains(err.Error(), "in use") {
// 			// optionally send command line arguments to the other instance
// 			fmt.Fprintln(os.Stderr, "Already running.")
// 			return
// 		} else {
// 			panic(err)
// 		}
// 	}
// 	defer listener.Close()

// 	// Create a channel to receive the completion status of the application
// 	doneChan := make(chan error, 1)
// 	app := exec.Command("./ui/_ui.exe")
// 	go app.Run()

// 	runMode := os.Getenv("RUN_MODE")
// 	if runMode == "DEV" {
// 		cpuf, err := os.Create("cpu_profile")
// 		if err != nil {
// 			log.Log.Error(err)
// 		}
// 		pprof.StartCPUProfile(cpuf)
// 		defer pprof.StopCPUProfile()
// 	}

// 	s := svc.NewScaleMgr() // srvMgr should be set via SetScaleMgr
// 	// channel to inform application to quit
// 	quitch := make(chan bool)
// 	h := *svc.NewSrvMgr(s, quitch)

// 	s.SetSrvMsg(&h)
// 	go s.Run()

// 	go h.Run()
// 	ws := svc.NewWsServer()
// 	go ws.Start(&h)

// 	// Create a channel to receive signals
// 	sigChan := make(chan os.Signal, 10)

// 	// Notify the signal channel for specific OS signals
// 	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

// 	// Wait for the process to complete and send the completion status to the doneChan channel
// 	go func() {
// 		time.Sleep(1000000 * time.Second)
// 		doneChan <- app.Wait()
// 	}()

// 	// Wait for either completion or a signal
// 	select {
// 	case <-sigChan:
// 		// Signal received, handle it as needed
// 		log.Log.Errorln("Received signal. Terminating...")
// 		err := app.Process.Kill()
// 		if err != nil {
// 			fmt.Println("Failed to kill the application:", err)
// 			return
// 		}
// 	case err := <-doneChan:
// 		// Application completed, handle the completion status
// 		if err != nil {
// 			log.Log.Errorf("Application completed with an error: %v", err)
// 		} else {
// 			log.Log.Infoln("Application completed successfully.")
// 		}
// 	}
// 	log.Log.Info("Process completed.")

// 	if runMode == "DEV" {
// 		memprof, _ := os.Create("mem.pprof")
// 		pprof.WriteHeapProfile(memprof)
// 		memprof.Close()
// 	}

// 	// Exit the program gracefully
// 	os.Exit(0)
// }
