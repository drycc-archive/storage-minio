package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"

	"github.com/drycc/minio/src/healthsrv"
	"github.com/drycc/pkg/utils"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	localMinioInsecure = false
	defaultMinioHost   = "localhost"
	defaultMinioPort   = "9000"
	defaultMinioExec   = "/opt/drycc/minio/bin/minio"
)

var (
	errHealthSrvExited = errors.New("healthcheck server exited with unknown status")
	errMinioExited     = errors.New("minio server exited with unknown status")
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

func newMinioClient(host, port, accessKey, accessSecret string, insecure bool) (*minio.Client, error) {
	return minio.New(fmt.Sprintf("%s:%s", host, port), &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, accessSecret, ""),
		Secure: insecure,
	})
}

func startServer(runErrCh chan error) {
	err := os.Setenv("MINIO_ROOT_USER", os.Getenv("DRYCC_MINIO_ACCESSKEY"))
	checkError(err)
	err = os.Setenv("MINIO_ROOT_PASSWORD", os.Getenv("DRYCC_MINIO_SECRETKEY"))
	checkError(err)

	os.Args[0] = defaultMinioExec
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
	accesskey := os.Getenv("DRYCC_MINIO_ACCESSKEY")
	secretkey := os.Getenv("DRYCC_MINIO_SECRETKEY")

	minioHost := os.Getenv("MINIO_HOST")
	if minioHost == "" {
		minioHost = defaultMinioHost
	}
	minioPort := os.Getenv("MINIO_PORT")
	if minioPort == "" {
		minioPort = defaultMinioPort
	}
	minioClient, err := newMinioClient(minioHost, minioPort, accesskey, secretkey, localMinioInsecure)
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
		if err := healthsrv.Start(healthSrvHost, healthSrvPort, minioClient); err != nil {
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

func main() {
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
