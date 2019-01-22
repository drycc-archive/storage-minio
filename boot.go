package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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

// Secret is a secret for the remote object storage
type Secret struct {
	Host      string
	KeyID     string
	AccessKey string
	Region    string
}

const configdir = "/home/minio/.minio/"

const templv2 = `{
	"version": "2",
	"credentials": {
  {{range .}}
		"accessKeyId": "{{.KeyID}}",
		"secretAccessKey": "{{.AccessKey}}",
		"region": "{{.Region}}"
  {{end}}
	},
	"mongoLogger": {
		"addr": "",
		"db": "",
		"collection": ""
	},
	"syslogLogger": {
		"network": "",
		"addr": ""
	},
	"fileLogger": {
		"filename": ""
	}
}`

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
	keyID, err := ioutil.ReadFile("/var/run/secrets/drycc/minio/user/accesskey")
	checkError(err)
	accessKey, err := ioutil.ReadFile("/var/run/secrets/drycc/minio/user/secretkey")
	checkError(err)
	return strings.TrimSpace(string(keyID)), strings.TrimSpace(string(accessKey))
}

func newMinioClient(host, port, accessKey, accessSecret string, insecure bool) (minio.CloudStorageClient, error) {
	return minio.New(
		fmt.Sprintf("%s:%s", host, port),
		accessKey,
		accessSecret,
		insecure,
	)
}

func main() {
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

	secrets := []Secret{
		{
			KeyID:     key,
			AccessKey: access,
			Region:    "us-east-1",
		},
	}
	t := template.New("MinioTpl")
	t, err = t.Parse(templv2)
	checkError(err)

	err = os.MkdirAll(configdir, 0755)
	checkError(err)
	output, err := os.Create(configdir + "config.json")
	checkError(err)
	err = t.Execute(output, secrets)
	checkError(err)
	os.Args[0] = "minio"
	mc := strings.Join(os.Args, " ")
	runErrCh := make(chan error)
	log.Printf("starting Minio server")
	go func() {
		if err := run(mc); err != nil {
			runErrCh <- err
		} else {
			runErrCh <- errMinioExited
		}
	}()

	healthSrvHost := os.Getenv("HEALTH_SERVER_HOST")
	if healthSrvHost == "" {
		healthSrvHost = healthsrv.DefaultHost
	}
	healthSrvPort, err := strconv.Atoi(os.Getenv("HEALTH_SERVER_PORT"))
	if err != nil {
		healthSrvPort = healthsrv.DefaultPort
	}

	log.Printf("starting health check server on %s:%d", healthSrvHost, healthSrvPort)

	healthSrvErrCh := make(chan error)
	go func() {
		if err := healthsrv.Start(healthSrvHost, healthSrvPort, minioClient); err != nil {
			healthSrvErrCh <- err
		} else {
			healthSrvErrCh <- errHealthSrvExited
		}
	}()

	select {
	case err := <-runErrCh:
		log.Printf("Minio server error (%s)", err)
		os.Exit(1)
	case err := <-healthSrvErrCh:
		log.Printf("Healthcheck server error (%s)", err)
		os.Exit(1)
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}
