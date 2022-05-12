package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/itrepablik/itrlog"
	"github.com/kardianos/service"
	"golang.org/x/sys/windows"
)

// TH32CS_SNAPPROCESS is described in https://msdn.microsoft.com/de-de/libr...
const TH32CS_SNAPPROCESS = 0x00000002

// Config is the runner app config structure.
type Config struct {
	Name, DisplayName, Description string

	Dir  string
	Exec string
	Args []string
	Env  []string

	Stderr, Stdout string
}

var logger service.Logger

type program struct {
	exit    chan struct{}
	service service.Service

	*Config

	cmd *exec.Cmd
}

func (p *program) Start(s service.Service) error {
	fullExec, err := exec.LookPath(p.Exec)
	if err != nil {
		return fmt.Errorf("failed to find executable %q: %v", p.Exec, err)
	}

	p.cmd = exec.Command(fullExec, p.Args...)
	p.cmd.Dir = p.Dir
	p.cmd.Env = append(os.Environ(), p.Env...)

	go p.run()
	return nil
}

func (p *program) run() {
	logger.Info("Starting ", p.DisplayName)
	itrlog.Info("Starting ", p.DisplayName)
	defer func() {
		if service.Interactive() {
			p.Stop(p.service)
		} else {
			p.service.Stop()
		}
	}()

	if p.Stderr != "" {
		f, err := os.OpenFile(p.Stderr, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
		if err != nil {
			logger.Warningf("Failed to open std err %q: %v", p.Stderr, err)
			itrlog.Warnf("Failed to open std err %q: %v", p.Stderr, err)
			return
		}
		defer f.Close()
		p.cmd.Stderr = f
	}
	if p.Stdout != "" {
		f, err := os.OpenFile(p.Stdout, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
		if err != nil {
			logger.Warningf("Failed to open std out %q: %v", p.Stdout, err)
			itrlog.Warnf("Failed to open std out %q: %v", p.Stdout, err)
			return
		}
		defer f.Close()
		p.cmd.Stdout = f
	}

	err := p.cmd.Run()
	if err != nil {
		logger.Warningf("Error running: %v", err)
		itrlog.Warnf("Error running: %v", err)
	}

}

func (p *program) Stop(s service.Service) error {
	defer close(p.exit)
	logger.Info("Stopping ", p.DisplayName)
	itrlog.Info("Stopping ", p.DisplayName)
	WindowsKillProcessByPID()

	if !p.cmd.ProcessState.Exited() {
		p.cmd.Process.Kill()
	}
	if service.Interactive() {
		os.Exit(0)
	}
	return nil
}

// WindowsKillProcessByPID is the kill a current process by PID.
func WindowsKillProcessByPID() {
	// Kill the 'gokopy.exe' process as well.
	procs, err := processes()
	if err != nil {
		logger.Error(err)
		itrlog.Error(err)
	}
	explorer := findProcessByName(procs, "fiberseries.exe")
	if explorer != nil {
		kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(explorer.ProcessID))
		err := kill.Run()
		if err != nil {
			logger.Error(err)
			itrlog.Error(err)
		}
	}
}

// WindowsProcess is an implementation of Process for Windows.
type WindowsProcess struct {
	ProcessID       int
	ParentProcessID int
	Exe             string
}

func processes() ([]WindowsProcess, error) {
	handle, err := windows.CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(handle)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	// get the first process
	err = windows.Process32First(handle, &entry)
	if err != nil {
		return nil, err
	}

	results := make([]WindowsProcess, 0, 50)
	for {
		results = append(results, newWindowsProcess(&entry))

		err = windows.Process32Next(handle, &entry)
		if err != nil {
			// windows sends ERROR_NO_MORE_FILES on last process
			if err == syscall.ERROR_NO_MORE_FILES {
				return results, nil
			}
			return nil, err
		}
	}
}

func findProcessByName(processes []WindowsProcess, name string) *WindowsProcess {
	for _, p := range processes {
		if strings.EqualFold(p.Exe, name) {
			// if strings.ToLower(p.Exe) == strings.ToLower(name) {
			return &p
		}
	}
	return nil
}

func newWindowsProcess(e *windows.ProcessEntry32) WindowsProcess {
	// Find when the string ends for decoding
	end := 0
	for {
		if e.ExeFile[end] == 0 {
			break
		}
		end++
	}

	return WindowsProcess{
		ProcessID:       int(e.ProcessID),
		ParentProcessID: int(e.ParentProcessID),
		Exe:             syscall.UTF16ToString(e.ExeFile[:end]),
	}
}

func main() {
	// Call the kardianos OS service library here
	svcFlag := flag.String("service", "", "Control the system service.")
	flag.Parse()

	svcConfig := &service.Config{
		Name:        "fiberseries2",
		DisplayName: "fiberseries2",
		Description: "A lightweight automated backup file software2.",
	}

	// get the currect .exe full path
	fullexepath, err := os.Executable()
	if err != nil {
		fmt.Println(err)
		logger.Error(err)
		itrlog.Error(err)
		return

	}

	dir, _ := filepath.Split(fullexepath)
	execCMD := ""

	// identify os platform
	switch os := runtime.GOOS; os {
	case "darwin":
		fmt.Println("OS X.")
	case "linux":
		fmt.Println("Linux")
	default:
		// freebsd, openbsd, plan9, windows ...
		execCMD = "C:\\windows\\system32\\cmd.exe" // Set to this path for windows os
	}

	configValues := &Config{
		Dir:    filepath.FromSlash(dir),
		Exec:   execCMD,
		Args:   []string{"/C", "C:\\fiberseries\\fiberseries.exe"},
		Env:    []string{},
		Stderr: "",
		Stdout: "",
	}
	prg := &program{
		exit:   make(chan struct{}),
		Config: configValues,
	}

	s, err := service.New(prg, svcConfig)

	if err != nil {
		log.Fatal(err)
		itrlog.Fatal(err)
	}

	prg.service = s

	errs := make(chan error, 5)
	logger, err = s.Logger(errs)
	if err != nil {
		log.Fatal(err)
		itrlog.Fatal(err)
	}

	go func() {
		for {
			err := <-errs
			if err != nil {
				log.Print(err)
			}
		}
	}()

	if len(*svcFlag) != 0 {
		err := service.Control(s, *svcFlag)
		if err != nil {
			log.Printf("Valid actions: %q\n", service.ControlAction)
			log.Fatal(err)
			itrlog.Error(err)
		}
		return
	}

	err = s.Run()
	if err != nil {
		logger.Error(err)
		log.Fatal(err)
		itrlog.Error(err)
	}
}
