package haproxy

import (
	"os"
	"os/exec"
	"syscall"
)

type haproxy struct {
	ConfigPath string
	cmd        *exec.Cmd
}

func New(configPath string, flags ...string) *haproxy {
	flags = append(flags, "-W", "-f", configPath)
	cmd := exec.Command("haproxy", flags...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return &haproxy{cmd: cmd, ConfigPath: configPath}
}

func (h *haproxy) Validate() error {
	cmd := exec.Command("haproxy", "-c", "-f", h.ConfigPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (h *haproxy) Start() error {
	err := h.cmd.Start()
	if err != nil {
		return err
	}
	return nil
}

func (h *haproxy) Stop() {
	h.cmd.Process.Signal(os.Interrupt)
}

func (h *haproxy) Reload() {
	h.cmd.Process.Signal(syscall.SIGUSR2)
}
