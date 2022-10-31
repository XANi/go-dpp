package puppet

import (
	"fmt"
	"github.com/op/go-logging"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	//	"os"
	"bufio"
	"io"
)

var log = logging.MustGetLogger("main")

const lastRunSummaryFile = "/var/lib/puppet/state/last_run_summary.yaml"

type LastRunSummaryVersion struct {
	Config string
	Puppet string
}
type LastRunSummary struct {
	Version LastRunSummaryVersion `yaml:"version"`
	// config: 1667193532
	// puppet: 5.5.22
	Resources map[string]int `yaml:"resources"`
	// changed: 8
	// corrective_change: 0
	// failed: 2
	// failed_to_restart: 0
	// out_of_sync: 10
	// restarted: 0
	// scheduled: 0
	// skipped: 2
	// total: 2043

	Time map[string]float64 `yaml:"time"`
	// anchor: 0.0009562040000000001
	// apt_key: 8.169722785
	// archive: 0.064915378
	// catalog_application: 15.424887365000359
	// config_retrieval: 1.68752246
	// cron: 0.005688408
	// exec: 0.048219635000000004
	// file: 0.479023673
	// filebucket: 6.2893e-05
	// host: 0.0013051590000000002
	// notify: 0.002109504
	// package: 2.0101792110000005
	// schedule: 0.000314459
	// service: 0.509904543
	// tidy: 4.8771e-05
	// total: 16.106930851
	// transaction_evaluation: 15.233454602000165
	// user: 0.000224368
	// last_run: 1667193550
	Changes map[string]int `yaml:"changes"`
	// total: 8
	Events map[string]int `yaml:"events"`
	// failure: 2
	// success: 8
	// total: 10
}

type Puppet struct {
	ModulePath   []string
	ManifestPath string
	PuppetPath   string
	sync.Mutex
	ioLock        sync.WaitGroup
	lastrunLock   sync.Mutex
	LastRun       LastRunSummary
	LastRunUpdate time.Time
	LastRunFailed bool
}

//func New(modulePath []string, manifestPath string) (p *Puppet, err error) {
func New(modulePath []string, manifestPath string) (p *Puppet, err error) {
	var pup Puppet
	pup.ModulePath = modulePath
	pup.ManifestPath = manifestPath
	pup.PuppetPath, err = exec.LookPath("puppet")
	return &pup, err
}

func (p *Puppet) Run() (err error) {
	p.Lock()
	defer p.Unlock()
	cmd := exec.Command(p.PuppetPath, "apply", "-v", "--modulepath", strings.Join(p.ModulePath, ":"), p.ManifestPath)
	log.Debugf("Running puppet with command %+v", cmd.Args)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	// Run puppet
	if err := cmd.Start(); err != nil {
		return err
	}
	//attach loggers
	p.ioLock.Add(2)
	go p.logStdout(stdout)
	go p.logStderr(stderr)
	p.ioLock.Wait()
	log.Notice("Puppet run ended")
	err = cmd.Wait()
	if err != nil {
		p.LastRunFailed = true
		return err
	}
	runStatsFileStats, err := os.Stat(lastRunSummaryFile)
	p.lastrunLock.Lock()
	defer p.lastrunLock.Unlock()
	if err != nil {
		p.LastRunFailed = true
	} else {
		if time.Now().Sub(runStatsFileStats.ModTime()).Seconds() > 60 {
			p.LastRunFailed = true
			return fmt.Errorf("last run failed [%s older than 60s after run]", lastRunSummaryFile)
		} else {
			lastRunRaw, err := ioutil.ReadFile(lastRunSummaryFile)
			if err != nil {
				return fmt.Errorf("can't read [%s]", lastRunSummaryFile)
			}
			err = yaml.Unmarshal(lastRunRaw, &p.LastRun)
			if err != nil {
				return fmt.Errorf("error unmarshalling [%s]: %s", lastRunSummaryFile, err)
			}
			p.LastRunFailed = false
			p.LastRunUpdate = runStatsFileStats.ModTime()
		}
	}
	return nil
}

func (p *Puppet) LastRunStats() (success bool, summary LastRunSummary, ts time.Time) {
	p.lastrunLock.Lock()
	defer p.lastrunLock.Unlock()
	return !p.LastRunFailed, p.LastRun, p.LastRunUpdate
}

func (p *Puppet) logStdout(r io.Reader) {
	defer p.ioLock.Done()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log.Notice(scanner.Text()) // Println will add back the final '\n'
	}
}
func (p *Puppet) logStderr(r io.Reader) {
	defer p.ioLock.Done()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log.Warning(scanner.Text()) // Println will add back the final '\n'
	}
}
