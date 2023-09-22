package provider

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/golang/glog"
)

func init() {
	globalProviders["juicefs"] = &juicefsProvider{
		BaseProvider: BaseProvider{},
	}
}

type juicefsProvider struct {
	BaseProvider
	Name      string
	MetaURL   string
	BlockSize uint64
	TrashDays uint64
}

func (provider *juicefsProvider) ParseFlag() error {
	name := flag.String("name", "drycc", "name is the prefix of all objects in data storage.")
	metaURL := flag.String("meta-url", "", "meta-url is used to set up the metadata engine.")
	blockSize := flag.Uint64("block-size", 4096, "size of block in KiB.")
	trashDays := flag.Uint64("trash-days", 0, "number of days after which removed files will be permanently deleted.")
	flag.Parse()
	if *metaURL == "" {
		return errors.New("meta-url is required")
	}
	provider.Name = *name
	provider.MetaURL = *metaURL
	provider.BlockSize = *blockSize
	provider.TrashDays = *trashDays
	return nil
}

func (provider *juicefsProvider) NodeMountVolume(bucket, prefix, path string, capacity uint64, context map[string]string, options ...string) error {
	metaURL, err := provider.formatJuicefs(bucket, prefix, capacity, context)
	if err != nil {
		return err
	}
	args := []string{
		"mount",
		metaURL,
		path,
		"--background",
	}
	args = append(args, options...)
	cmd := exec.Command("juicefs", args...)
	cmd.Stderr = os.Stderr
	glog.V(3).Infof("juicefs format with command: %s and args: %s", "juicefs", args)

	if out, err := cmd.Output(); err != nil {
		return fmt.Errorf("error exec command: %s\nargs: %s\noutput: %s", "juicefs", args, out)
	}

	return provider.NodeWaitMountVolume(path, 10*time.Second)
}

func (provider *juicefsProvider) ControllerExpandVolume(bucket, prefix string, capacity uint64, context map[string]string) error {
	inode := capacity / provider.BlockSize
	metaURL := fmt.Sprintf("%s/%s/%s", provider.MetaURL, bucket, prefix)
	args := []string{
		"config",
		metaURL,
		"--inodes", strconv.FormatUint(inode, 10),
		"--capacity", strconv.FormatUint(provider.formatCapacity(capacity), 10),
	}
	cmd := exec.Command("juicefs", args...)

	cmd.Stderr = os.Stderr
	glog.V(3).Infof("juicefs config with command: %s and args: %s and context: %s", "juicefs", args, context)

	input, e := cmd.StdinPipe()
	defer input.Close()
	if e != nil {
		return e
	}
	input.Write([]byte("n"))

	if out, err := cmd.Output(); err != nil {
		return fmt.Errorf("error exec command: %s\nargs: %s\noutput: %s", "juicefs", args, out)
	}
	return nil
}

func (provider *juicefsProvider) formatCapacity(capacity uint64) uint64 {
	capacity = capacity / (1024 * 1024 * 1024)
	if capacity < 1 {
		return 1
	}
	return capacity
}

func (provider *juicefsProvider) formatJuicefs(bucket, prefix string, capacity uint64, context map[string]string) (string, error) {
	endpoint := context["endpoint"]
	accessKey := context["accesskey"]
	secretKey := context["secretkey"]
	inode := capacity / provider.BlockSize
	metaURL := fmt.Sprintf("%s/%s/%s", provider.MetaURL, bucket, prefix)

	if out, err := exec.Command("juicefs", []string{"status", metaURL}...).Output(); err == nil {
		glog.V(3).Infof("%s has been formatted: %s", "juicefs", out)
		return metaURL, nil
	}

	args := []string{
		"format",
		"--inodes", strconv.FormatUint(inode, 10),
		"--block-size", strconv.FormatUint(provider.BlockSize, 10),
		"--capacity", strconv.FormatUint(provider.formatCapacity(capacity), 10),
		"--trash-days", strconv.FormatUint(provider.TrashDays, 10),
		"--storage", "s3",
		"--bucket", fmt.Sprintf("%s/%s/%s", endpoint, bucket, prefix),
		"--access-key", accessKey,
		"--secret-key", secretKey,
		metaURL,
		provider.Name,
	}
	cmd := exec.Command("juicefs", args...)
	cmd.Stderr = os.Stderr
	glog.V(3).Infof("juicefs format with command: %s and args: %s", "juicefs", args)
	if out, err := cmd.Output(); err != nil {
		return metaURL, fmt.Errorf("error exec command: %s\nargs: %s\noutput: %s", "juicefs", args, out)
	}
	return metaURL, nil
}
