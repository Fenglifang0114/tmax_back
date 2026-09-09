//go:build !windows

package util

import (
	"bufio"
	"fmt"
	"os/exec"
	"runtime"
)

func SetHideWindow(cmd *exec.Cmd) {}

func RunCommand(output chan<- string, done chan<- error, command string, args ...string) {
	cmd := exec.Command(command, args...)
	//隐藏窗口
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("osascript", "-e", "tell app \"Terminal\" to set miniaturized of window 1 to true", "-e", "tell application \"System Events\" to keystroke \"m\" using {command down, option down}")
	} else if runtime.GOOS == "linux" {
		cmd = exec.Command("xdotool", "getactivewindow", "windowminimize")
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		done <- fmt.Errorf("failed to create stdout pipe: %v", err)
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		done <- fmt.Errorf("failed to start command: %v", err)
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		output <- line // Send each line of output to the output channel
	}

	err = cmd.Wait()
	if err != nil {
		done <- fmt.Errorf("command failed: %v", err)
		return
	}

	done <- nil
}

func KillApp(appName string) error {
	cmd := exec.Command("taskkill", "/F", "/IM", appName)
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error killing process:", err)
		return err
	}

	fmt.Println("Process killed")
	return nil
}
