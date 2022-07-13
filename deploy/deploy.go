package deploy

import (
	//	"github.com/op/go-logging"
	"archive/tar"
	"compress/gzip"
	"io/ioutil"
	"os"
	"time"
)

//var log = logging.MustGetLogger("main")

type Config struct {
	Files []string
}

type deployer struct {
	cfg *Config
}

func NewDeployer(c Config) (*deployer, error) {
	var d deployer
	if len(c.Files) < 1 {
		c.Files = []string{
			`/etc/puppet/hiera.yaml`,
			`/etc/puppet/puppet.conf`,
			`/etc/dpp/config.yaml`,
		}
	}
	d.cfg = &c
	return &d, nil
}

// PrepareDeployPackage packs all required files into a package
func (d *deployer) PrepareDeployPackage(outfile string) error {
	// Create a buffer to write our archive to.
	f, err := os.Create(outfile)
	if err != nil {
		return err
	}
	gw := gzip.NewWriter(f)
	// Create a new tar archive.
	tw := tar.NewWriter(gw)
	for _, filename := range d.cfg.Files {
		body, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		ts := time.Now()
		hdr := &tar.Header{
			Name:       filename,
			Mode:       0600,
			Size:       int64(len(body)),
			AccessTime: ts,
			ModTime:    ts,
			ChangeTime: ts,
			Uname:      `root`,
			Gname:      `root`,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(body); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	err = gw.Close()
	if err != nil {
		return err
	}
	return f.Close()
}
