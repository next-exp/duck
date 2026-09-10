//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/magefile/mage/mg"
)

// Default target to run when none is specified
// If not set, running mage will list available targets
var Default = Build

// Generate runs sqlc code generation
func Generate() error {
	fmt.Println("Generating sqlc code...")
	sqlcPath := "$(go env GOPATH)/bin/sqlc"
	cmd := exec.Command("sh", "-c", sqlcPath+" generate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// A build step that requires additional params, or platform specific steps for example
func Build() error {
	mg.Deps(Generate)
	mg.Deps(BuildRateReader)
	mg.Deps(BuildTestDataBuilder)
	mg.Deps(BuildLDCServer)
	mg.Deps(BuildGDCServer)
	mg.Deps(BuildDeviceSimulator)
	mg.Deps(BuildAPIServer)
	mg.Deps(BuildReadLogs)
	mg.Deps(BuildEventDump)
	fmt.Println("Compilation finished")
	return nil
}

// A custom install step if you need your bin someplace other than go/bin
func Install() error {
	mg.Deps(Build)
	fmt.Println("Installing...")
	return os.Rename("./MyApp", "/usr/bin/MyApp")
}

func BuildGDCServer() error {
	fmt.Println("Building gdcRPC executable...")
	ldflags := os.Getenv("CGO_LDFLAGS")
	cflags := os.Getenv("CGO_CFLAGS")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", "./bin/gdcRPC", "./gdcRPC")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("CGO_ENABLED=1"),
		fmt.Sprintf("CGO_LDFLAGS=%s", ldflags),
		fmt.Sprintf("CGO_CFLAGS=%s", cflags))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func BuildLDCServer() error {
	fmt.Println("Building ldcRPC executable...")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", "./bin/ldcRPC", "./ldcRPC")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func BuildAPIServer() error {
	fmt.Println("Building API executable...")
	args := []string{"build", "-buildvcs=false"}
	if release := os.Getenv("DUCK_RELEASE"); release != "" {
		args = append(args, "-ldflags", "-X main.release="+release)
	}
	args = append(args, "-o", "./bin/duckAPI", "./api")
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func BuildReadLogs() error {
	fmt.Println("Building readLogs executable...")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", "./bin/readLogs", "./readLogs")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func BuildRateReader() error {
	fmt.Println("Building rateReader executable...")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", "./bin/rateReader", "./rateReader")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func BuildTestDataBuilder() error {
	fmt.Println("Building testDataBuilder executable...")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", "./bin/testDataBuilder", "./testDataBuilder")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func BuildEventDump() error {
	fmt.Println("Building eventDump executable...")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", "./bin/eventDump", "./eventDump")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func BuildDeviceSimulator() error {
	fmt.Println("Building deviceSimulator executable...")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", "./bin/deviceSimulator", "./deviceSimulator")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
