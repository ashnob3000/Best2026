package cloudflare

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sync"
)

type TunnelManager struct {
	mu      sync.RWMutex
	vlessURL string
	trojanURL string
}

func NewTunnelManager() *TunnelManager {
	return &TunnelManager{}
}

func (tm *TunnelManager) StartVLESS() {
	tm.start("vless", 2097)
}

func (tm *TunnelManager) StartTrojan() {
	tm.start("trojan", 2098)
}

func (tm *TunnelManager) start(protocol string, port int) {
	ctx := context.Background()

	cmd := exec.CommandContext(
		ctx,
		"cloudflared",
		"tunnel",
		"--url",
		fmt.Sprintf("http://127.0.0.1:%d", port),
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("%s tunnel error: %v\n", protocol, err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("%s tunnel error: %v\n", protocol, err)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("%s cloudflared start error: %v\n", protocol, err)
		return
	}

	go tm.readOutput(protocol, stdout)
	go tm.readOutput(protocol, stderr)
}

func (tm *TunnelManager) readOutput(protocol string, output interface {
	Read([]byte) (int, error)
}) {
	scanner := bufio.NewScanner(output)

	re := regexp.MustCompile(`https://[a-zA-Z0-9.-]+\.trycloudflare\.com`)

	for scanner.Scan() {
		line := scanner.Text()

		fmt.Println(line)

		match := re.FindString(line)
		if match == "" {
			continue
		}

		tm.mu.Lock()

		if protocol == "vless" {
			tm.vlessURL = match
		} else {
			tm.trojanURL = match
		}

		tm.mu.Unlock()

		fmt.Printf("%s tunnel URL: %s\n", protocol, match)
	}
}

func (tm *TunnelManager) VLESSURL() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.vlessURL
}

func (tm *TunnelManager) TrojanURL() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.trojanURL
}
