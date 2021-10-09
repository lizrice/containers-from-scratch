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

var homePath string

func init(){
	homePath = os.Getenv("HOME")
	if homePath == ""{
		panic("HOME env must be specified")
	}
}

// go run main.go run <cmd> <args>
func main() {
	switch os.Args[1] {
	case "run":
		run()
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
	rootPath := fmt.Sprintf("%s/ubuntufs", homePath)
	must(syscall.Mount(rootPath, rootPath, "bind", syscall.MS_BIND, ""))
	// bind mount
	testBindPath := filepath.Join(rootPath, "test")
	os.Mkdir(testBindPath, 0755)
	must(syscall.Mount(fmt.Sprintf("%s/test", homePath), testBindPath, "", syscall.MS_BIND, ""))

	// jail rootfs with pivot_root syscall
	// ref: https://github.com/opencontainers/runc/blob/v1.0.2/libcontainer/rootfs_linux.go#L817
	putOldPath := filepath.Join(rootPath, "put_old")
	os.Mkdir(putOldPath, 0755)
	must(syscall.PivotRoot(rootPath, putOldPath))
	// lazy unmount
	must(syscall.Unmount("/put_old", syscall.MNT_DETACH))
	if err := os.Remove("/put_old"); err != nil{
		panic(err)
	}
	//must(syscall.Chroot("fmt.Sprintf("%s/ubuntufs", homePath))
	must(os.Chdir("/"))
	must(syscall.Mount("proc", "proc", "proc", 0, ""))
	must(syscall.Mount("thing", "mytemp", "tmpfs", 0, ""))
 
	must(cmd.Run())

	must(syscall.Unmount("proc", 0))
	must(syscall.Unmount("mytemp", 0))
	must(syscall.Unmount("/test", 0))
}

func cg() {
	cgroups := "/sys/fs/cgroup/"
	pids := filepath.Join(cgroups, "pids")
	os.Mkdir(filepath.Join(pids, "liz"), 0755)
	// Add process limition for 20
	must(ioutil.WriteFile(filepath.Join(pids, "liz/pids.max"), []byte("20"), 0700))
	// Removes the new cgroup in place after the container exits
	must(ioutil.WriteFile(filepath.Join(pids, "liz/notify_on_release"), []byte("1"), 0700))
	// Add current process in the cgroup
	must(ioutil.WriteFile(filepath.Join(pids, "liz/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))

	// Add cpu limitation for 0.3 core
	cpu := filepath.Join(cgroups, "cpu")
	os.Mkdir(filepath.Join(cpu, "liz"), 0755)
	must(ioutil.WriteFile(filepath.Join(cpu, "liz/cpu.cfs_period_us"), []byte("100000"), 0700))
	must(ioutil.WriteFile(filepath.Join(cpu, "liz/cpu.cfs_quota_us"), []byte("30000"), 0700))
	must(ioutil.WriteFile(filepath.Join(cpu, "liz/notify_on_release"), []byte("1"), 0700))
	must(ioutil.WriteFile(filepath.Join(cpu, "liz/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))

	// Add memory limitation for 100M
	mem := filepath.Join(cgroups, "memory")
	os.Mkdir(filepath.Join(mem, "liz"), 0755)
	must(ioutil.WriteFile(filepath.Join(mem, "liz/memory.limit_in_bytes"), []byte("100M"), 0700))
	must(ioutil.WriteFile(filepath.Join(mem, "liz/memory.swappiness"), []byte("0"), 0700))
	must(ioutil.WriteFile(filepath.Join(mem, "liz/notify_on_release"), []byte("1"), 0700))
	must(ioutil.WriteFile(filepath.Join(mem, "liz/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
