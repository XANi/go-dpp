package puppet

import (
	"sync"
	"os/exec"
	"github.com/op/go-logging"
	"strings"
//	"os"
	"bufio"
	"io"
)
var log = logging.MustGetLogger("main")

type Puppet struct {
	ModulePath []string
	ManifestPath string
	PuppetPath string
	sync.Mutex
	ioLock sync.WaitGroup
}


//func New(modulePath []string, manifestPath string) (p *Puppet, err error) {
func New(modulePath []string, manifestPath string) (p *Puppet, err error) {
	var pup Puppet
	pup.ModulePath = modulePath
	pup.ManifestPath = manifestPath
	pup.PuppetPath, err = exec.LookPath("puppet")
	return &pup, err
}

func (p *Puppet)Run() (err error) {
	p.Lock()
	defer p.Unlock()
	cmd :=  exec.Command(p.PuppetPath,"apply","-v", "--modulepath", strings.Join(p.ModulePath,":"), p.ManifestPath)
	log.Debugf("Running puppet with command %+v", cmd.Args)
	stdout, err := cmd.StdoutPipe()
	if err != nil { return err }
	stderr, err := cmd.StderrPipe()
	if err != nil {	return err	}
	// Run puppet
	if err := cmd.Start(); err != nil {	return err	}
	//attach loggers
	p.ioLock.Add(2)
	go p.handleStdout(stdout)
	go p.handleStderr(stderr)
	p.ioLock.Wait()
	log.Notice("Puppet run ended")
	return err
}

func(p *Puppet) handleStdout(r io.Reader) {
	defer p.ioLock.Done()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log.Notice(scanner.Text()) // Println will add back the final '\n'
	}
}
func(p *Puppet) handleStderr(r io.Reader) {
	defer p.ioLock.Done()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log.Warning(scanner.Text()) // Println will add back the final '\n'
	}
}
