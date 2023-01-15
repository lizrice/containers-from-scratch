package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

var cgroups = "/sys/fs/cgroup"
var custom_cgroup = filepath.Join(cgroups, "liz")

// go run main.go run <cmd> <args>
func main() {
	switch os.Args[1] {
	case "run":
		run()
		// Removes the instantiated cgroup after container exit
		cgCleanup()
	case "child":
		child()
	default:
		panic("help")
	}
}

func run() {
	fmt.Printf("Running %v \n", os.Args[2:])

	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	must(cmd.Run())
}

func child() {
	fmt.Printf("Running %v \n", os.Args[2:])

	cg()

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	must(syscall.Sethostname([]byte("container")))
	// cd /tmp/containers-from-scratch
	// skopeo copy docker://ubuntu oci:ubuntu
	// mkdir ubuntufs
	// tar -xf ../ubuntu/blobs/sha256/somesha.... -C ubuntufs && rm -rf ubuntu
	// mkdir ubuntufs/mytemp
	must(syscall.Chroot("/tmp/containers-from-scratch/ubuntufs/"))
	must(os.Chdir("/"))
	must(syscall.Mount("proc", "proc", "proc", 0, ""))
	must(syscall.Mount("thing", "mytemp", "tmpfs", 0, ""))

	must(cmd.Run())

	must(syscall.Unmount("/proc", 0))
	must(syscall.Unmount("/mytemp", 0))
}

func cg() {
	cgroups := "/sys/fs/cgroup/"
	pids := filepath.Join(cgroups, "pids")
	os.Mkdir(filepath.Join(pids, "liz"), 0755)
	must(ioutil.WriteFile(filepath.Join(pids, "liz/pids.max"), []byte("20"), 0700))
	must(ioutil.WriteFile(filepath.Join(pids, "liz/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
func cgCleanup() error {
	alive, err := ioutil.ReadFile(filepath.Join(custom_cgroup, "pids.current"))
	if err != nil { // or must(err).. but then it'll look weird..
		panic(err)
	}

	if alive[0] != uint8(48) {
		must(ioutil.WriteFile(filepath.Join(custom_cgroup, "cgroup.kill"), []byte("1"), 0644))
	}
	must(os.Remove(custom_cgroup))

	return nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
