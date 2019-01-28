package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"os/exec"
	"strconv"
	"strings"
	"text/template"

	"github.com/drycc/minio/src/healthsrv"
	"github.com/drycc/pkg/utils"
	minio "github.com/minio/minio-go"
)

const (
	localMinioInsecure = true
	defaultMinioHost   = "localhost"
	defaultMinioPort   = "9000"
)

var (
	errHealthSrvExited = errors.New("healthcheck server exited with unknown status")
	errMinioExited     = errors.New("Minio server exited with unknown status")
)

const configdir = "/home/minio/.minio/"

func run(cmd string) error {
	var cmdBuf bytes.Buffer
	tmpl := template.Must(template.New("cmd").Parse(cmd))
	if err := tmpl.Execute(&cmdBuf, nil); err != nil {
		log.Fatal(err)
	}
	cmdString := cmdBuf.String()
	fmt.Println(cmdString)
	var cmdl *exec.Cmd
	cmdl = exec.Command("sh", "-c", cmdString)
	if _, _, err := utils.RunCommandWithStdoutStderr(cmdl); err != nil {
		return err
	}
	return nil
}

func readSecrets() (string, string) {
	key := readConfig("/var/run/secrets/drycc/objectstore/creds/accesskey")
	secret := readConfig("/var/run/secrets/drycc/objectstore/creds/secretkey")
	return key, secret
}

func readConfig(filename string) string {
	value, err := ioutil.ReadFile(filename)
	checkError(err)
	return strings.TrimSpace(string(value))
}

func newMinioClient(host, port, accessKey, accessSecret string, insecure bool) (*minio.Client, error) {
	return minio.New(
		fmt.Sprintf("%s:%s", host, port),
		accessKey,
		accessSecret,
		insecure,
	)
}

func startServer(runErrCh chan error) {

	key, access := readSecrets()

        err := os.Setenv("MINIO_ACCESS_KEY", key)
        checkError(err)
        err = os.Setenv("MINIO_SECRET_KEY", access)
        checkError(err)

	os.Args[0] = "minio"
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

func startGateway(runErrCh chan error) {
	key, access := readSecrets()
	err := os.Setenv("MINIO_ACCESS_KEY", key)
	checkError(err)
	err = os.Setenv("MINIO_SECRET_KEY", access)
	checkError(err)

	storage := os.Args[2]
	if storage == "gcs" {
		projectid := readConfig("/var/run/secrets/drycc/objectstore/creds/projectid")
		os.Args = append(os.Args, projectid)
		err = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/var/run/secrets/drycc/objectstore/creds/key.json")
		checkError(err)
	} else {
		endpoint := readConfig("/var/run/secrets/drycc/objectstore/creds/endpoint")
		checkError(err)
		os.Args = append(os.Args, endpoint)
	}

	os.Args[0] = "minio"
	mc := strings.Join(os.Args, " ")
	log.Printf("starting Minio gateway")
	go func() {
		if err := run(mc); err != nil {
			runErrCh <- err
		} else {
			runErrCh <- errMinioExited
		}
	}()
}

func startHealth(healthSrvErrCh chan error) {
	key, access := readSecrets()

	minioHost := os.Getenv("MINIO_HOST")
	if minioHost == "" {
		minioHost = defaultMinioHost
	}
	minioPort := os.Getenv("MINIO_PORT")
	if minioPort == "" {
		minioPort = defaultMinioPort
	}
	minioClient, err := newMinioClient(minioHost, minioPort, key, access, localMinioInsecure)
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

	service := regexp.MustCompile("[ \t]").Split(os.Args[1], -1)[0]
	if service == "server" {
		startServer(runErrCh)
	} else if service == "gateway" {
		startGateway(runErrCh)
	}

	startHealth(healthSrvErrCh)
	select {
	case err := <-runErrCh:
		log.Printf("Minio server error (%s)", err)
		os.Exit(1)
	case err := <-healthSrvErrCh:
		log.Printf("Healthcheck server error (%s)", err)
		os.Exit(1)
	}
}
