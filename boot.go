package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/drycc/pkg/utils"

	"github.com/drycc/storage/src/csi/driver"
	"github.com/drycc/storage/src/healthsrv"
	"github.com/drycc/storage/src/storage"
)

const (
	localMinioInsecure = false
	defaultMinioHost   = "localhost"
	defaultMinioPort   = "9000"
	defaultMinioExec   = "/opt/drycc/minio/bin/minio"
	// Help template for minfs.
	mainHelpTemplate = `NAME:
Drycc Storage - A open-source, distributed storage system.

USAGE:
boot [command] [command options] [arguments...]

VERSION:
   3.0.1

COMMANDS:
{{- range $command := .}}
    {{$command}}
{{- end}}

COPYRIGHT:
   Apache License 2.0
`
)

var (
	commandRunners = map[string]func(command string){
		"driver":      runDriver,
		"minio":       runMinio,
		"pd-server":   startPDServer,
		"tikv-ctl":    runCommand,
		"pd-ctl":      runCommand,
		"tikv-server": runCommand,
	}
	errMinioExited     = errors.New("minio server exited with unknown status")
	errHealthSrvExited = errors.New("healthcheck server exited with unknown status")
)

func run(cmd string) error {
	var cmdBuf bytes.Buffer
	tmpl := template.Must(template.New("cmd").Parse(cmd))
	if err := tmpl.Execute(&cmdBuf, nil); err != nil {
		log.Fatal(err)
	}
	cmdString := cmdBuf.String()
	fmt.Println(cmdString)
	var cmdl = exec.Command("bash", "-c", cmdString)
	if _, _, err := utils.RunCommandWithStdoutStderr(cmdl); err != nil {
		return err
	}
	return nil
}

func startServer(runErrCh chan error) {
	err := os.Setenv("MINIO_ROOT_USER", os.Getenv("DRYCC_STORAGE_ACCESSKEY"))
	checkError(err)
	err = os.Setenv("MINIO_ROOT_PASSWORD", os.Getenv("DRYCC_STORAGE_SECRETKEY"))
	checkError(err)

	mc := strings.Join(os.Args, " ")
	log.Printf("starting Minio server")
	go func() {
		if err := run(mc); err != nil {
			runErrCh <- err
		} else {
			runErrCh <- errMinioExited
		}
	}()
}

func startHealth(healthSrvErrCh chan error) {
	accesskey := os.Getenv("DRYCC_STORAGE_ACCESSKEY")
	secretkey := os.Getenv("DRYCC_STORAGE_SECRETKEY")

	minioHost := os.Getenv("MINIO_HOST")
	if minioHost == "" {
		minioHost = defaultMinioHost
	}
	minioPort := os.Getenv("MINIO_PORT")
	if minioPort == "" {
		minioPort = defaultMinioPort
	}
	healthChecker, err := storage.NewHealthChecker(minioHost, minioPort, accesskey, secretkey, localMinioInsecure)
	if err != nil {
		log.Printf("Error creating minio client (%s)", err)
		os.Exit(1)
	}

	healthSrvHost := os.Getenv("HEALTH_SERVER_HOST")
	if healthSrvHost == "" {
		healthSrvHost = healthsrv.DefaultHost
	}
	healthSrvPort, err := strconv.Atoi(os.Getenv("HEALTH_SERVER_PORT"))
	if err != nil {
		healthSrvPort = healthsrv.DefaultPort
	}

	log.Printf("starting health check server on %s:%d", healthSrvHost, healthSrvPort)

	go func() {
		if err := healthsrv.Start(healthSrvHost, healthSrvPort, healthChecker); err != nil {
			healthSrvErrCh <- err
		} else {
			healthSrvErrCh <- errHealthSrvExited
		}
	}()
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}

func runDriver(command string) {
	nodeID := os.Getenv("DRYCC_STORAGE_CSI_NODE_ID")
	provider := os.Getenv("DRYCC_STORAGE_CSI_PROVIDER")
	endpoint := os.Getenv("DRYCC_STORAGE_CSI_ENDPOINT")
	if nodeID == "" {
		log.Fatal("env DRYCC_STORAGE_CSI_NODE_ID is required.")
	}
	if provider == "" {
		log.Fatal("env DRYCC_STORAGE_CSI_PROVIDER is required.")
	}
	if endpoint == "" {
		log.Fatal("env DRYCC_STORAGE_CSI_ENDPOINT is required.")
	}
	driver, err := driver.New(nodeID, provider, endpoint)
	if err != nil {
		log.Fatal(err)
	}
	driver.Run()
	os.Exit(0)
}

func runMinio(command string) {
	os.Args[0] = defaultMinioExec
	runErrCh := make(chan error)
	healthSrvErrCh := make(chan error)
	startServer(runErrCh)
	startHealth(healthSrvErrCh)
	select {
	case err := <-runErrCh:
		log.Printf("minio server error (%s)", err)
		os.Exit(1)
	case err := <-healthSrvErrCh:
		log.Printf("healthcheck server error (%s)", err)
		os.Exit(1)
	}
}

func checkConnect(host string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", host, timeout*time.Second)
	if err != nil {
		fmt.Println("Connecting error:", host, err)
	}
	if conn != nil {
		defer conn.Close()
		return true
	}
	return false
}

func startPDServer(command string) {
	endpointsString := os.Getenv("DRYCC_STORAGE_PD_ENDPOINTS")
	endpoints := strings.Split(endpointsString, ",")
	for index := range endpoints {
		if endpoint, err := url.Parse(endpoints[index]); err != nil {
			log.Fatal(err)
		} else {
			if checkConnect(endpoint.Host, 5) {
				os.Args = append(os.Args, "--join", endpointsString)
				break
			}
		}
	}
	runCommand("pd-server")
}

func runCommand(cmd string) {
	runErrCh := make(chan error)
	os.Args[0] = cmd
	command := strings.Join(os.Args, " ")
	log.Printf("starting %s server", cmd)
	go func() {
		if err := run(command); err != nil {
			runErrCh <- err
		} else {
			runErrCh <- errMinioExited
		}
	}()
	if err := <-runErrCh; err != nil {
		log.Printf("run %s error (%s)", cmd, err)
		os.Exit(1)
	}
}

func help() {
	if tpl, err := template.New("help").Parse(mainHelpTemplate); err != nil {
		log.Fatal(err)
	} else {
		tpl.Execute(os.Stdout, commandRunners)
	}
}

func main() {
	if len(os.Args) == 1 {
		help()
	} else {
		command := os.Args[1]
		os.Args = append(os.Args[:1], os.Args[1+1:]...)
		runner := commandRunners[command]
		if runner != nil {
			runner(command)
		} else {
			help()
		}
	}
}
