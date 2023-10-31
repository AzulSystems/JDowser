// Copyright 2020 Azul Systems, Inc. All rights reserved.
// Use of this source code is governed by the 3-Clause BSD
// license that can be found in the LICENSE file.

package jdowser

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/md5"
	"debug/elf"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type JVMVersionInfo struct {
	Version        string `json:"java_version"`
	RuntimeName    string `json:"runtime_name"`
	RuntimeVendor  string `json:"java_runtime_vendor"`
	RuntimeVersion string `json:"java_runtime_version"`
	VMName         string `json:"java_vm_name"`
	VMVendor       string `json:"java_vm_vendor"`
	VMVersion      string `json:"java_vm_version"`
}

type JVMInstallation struct {
	Host             string `json:"host"`
	JavaHome         string `json:"java_home"`
	IsJDK            bool   `json:"is_jdk"`
	LibJVM           string `json:"libjvm"`
	LibJVMHash       string `json:"libjvm_hash"`
	rt_jar           string
	base_jmod        string
	VersionInfo      JVMVersionInfo `json:"version_info"`
	RunningInstances int            `json:"running_instances"`
}

func InitJVMInstallation(libjvm string, config *Config) *JVMInstallation {
	var inst JVMInstallation

	hostname, _ := os.Hostname()
	inst.Host = hostname
	inst.LibJVM = libjvm
	inst.LibJVMHash, _ = md5sum(libjvm)
	inst.JavaHome = findJavaHome(libjvm)
	inst.VersionInfo = JVMVersionInfo{}

	if inst.JavaHome != "" {
		if _, err := os.Stat(path.Join(inst.JavaHome, "bin/javac")); err == nil || os.IsExist(err) {
			inst.IsJDK = true
		}

		var found bool

		wf := func(p string, info os.FileInfo, err error) error {
			if found {
				return filepath.SkipDir
			}
			if err == nil {
				fname := path.Base(p)
				if fname == "rt.jar" {
					inst.rt_jar = p
					found = true
					return filepath.SkipDir
				}
				if fname == "java.base.jmod" {
					inst.base_jmod = p
					found = true
					return filepath.SkipDir
				}
			}
			return nil
		}

		filepath.Walk(inst.JavaHome, wf)
	}

	for {
		if !config.nojvmrun && inst.JavaHome != "" && readVersionInfoFromOutput(&inst) {
			break
		}
		if readVersionInfoFromStrings(&inst) {
			break
		}
		if inst.rt_jar != "" && readVersionInfoFromRtJar(&inst) {
			break
		}
		if inst.base_jmod != "" && readVersionInfoFromBaseJmod(&inst) {
			break
		}
		break
	}

	return &inst
}

func md5sum(path string) (string, error) {
	var md5sum string
	file, err := os.Open(path)
	if err == nil {
		defer closeFile(file)
		hash := md5.New()
		if _, err := io.Copy(hash, file); err != nil {
			return md5sum, err
		}
		hashInBytes := hash.Sum(nil)[:16]
		md5sum = hex.EncodeToString(hashInBytes)
	}
	return md5sum, nil
}

func (inst *JVMInstallation) Dump(out *os.File) {
	fmt.Fprintln(out, "host:", inst.Host)
	fmt.Fprintln(out, "libjvm:", inst.LibJVM)
	fmt.Fprintln(out, "libjvm_hash:", inst.LibJVMHash)
	fmt.Fprintln(out, "java_home:", inst.JavaHome)
	fmt.Fprintln(out, "is_jdk:", inst.IsJDK)
	fmt.Fprintln(out, "java_version:", inst.VersionInfo.Version)
	fmt.Fprintln(out, "java_runtime_name:", inst.VersionInfo.RuntimeName)
	fmt.Fprintln(out, "java_runtime_version:", inst.VersionInfo.RuntimeVersion)
	fmt.Fprintln(out, "java_runtime_vendor:", inst.VersionInfo.RuntimeVendor)
	fmt.Fprintln(out, "java_vm_name:", inst.VersionInfo.VMName)
	fmt.Fprintln(out, "java_vm_version:", inst.VersionInfo.VMVersion)
	fmt.Fprintln(out, "java_vm_vendor:", inst.VersionInfo.VMVendor)
	fmt.Fprintln(out, "running_instances:", inst.RunningInstances)
	fmt.Fprintln(out)
}

func (inst *JVMInstallation) DumpCSV(out *os.File) {
	w := csv.NewWriter(out)
	w.Write([]string{
		inst.Host, inst.LibJVM,
		inst.LibJVMHash, inst.JavaHome,
		strconv.FormatBool(inst.IsJDK), inst.VersionInfo.Version,
		inst.VersionInfo.RuntimeName, inst.VersionInfo.RuntimeVersion,
		inst.VersionInfo.RuntimeVendor, inst.VersionInfo.VMName,
		inst.VersionInfo.VMVersion, inst.VersionInfo.VMVendor,
		strconv.FormatInt(int64(inst.RunningInstances), 10)})
	w.Flush()
}

func DumpCSVHeader(out *os.File) {
	w := csv.NewWriter(out)
	w.Write([]string{
		"host", "libjvm",
		"libjvm_hash", "java_home",
		"is_jdk", "java_version",
		"java_runtime_name", "java_runtime_version",
		"java_runtime_vendor", "java_vm_name",
		"java_vm_version", "java_vm_vendor",
		"running_instances"})
	w.Flush()
}

func readVersionInfoFromRtJar(inst *JVMInstallation) bool {
	output, err := extractVersionClass(inst.rt_jar)
	if err == nil && len(output) != 0 {
		extractVersionStringsFromClassFileBytes(output, &inst.VersionInfo)
		return true
	}
	return false
}

func readVersionInfoFromBaseJmod(inst *JVMInstallation) bool {
	output, _ := exec.Command("unzip", "-cpq", inst.base_jmod, "classes/java/lang/VersionProps.class").Output()
	if len(output) != 0 {
		extractVersionStringsFromClassFileBytes(output, &inst.VersionInfo)
		return true
	}
	return false
}

func processStringsFromFile(fileName string, offset int, length int, callback func(str string) bool) error {
	f, e := os.Open(fileName)
	if e != nil {
		return e
	}
	_, e = f.Seek(int64(offset), 0)
	if e != nil {
		return e
	}
	defer closeFile(f)

	r := bufio.NewReader(f)

readLoop:
	for size := length; size > 0; {
		str, e := r.ReadString(0)
		if e != nil {
			if e == io.EOF {
				return nil
			}
			return e
		}
		strLen := len(str)
		size -= strLen

		if strLen > 10 {
			for i := 0; i < strLen-1; i++ {
				if str[i] < ' ' || str[i] > '~' {
					continue readLoop
				}
			}
			if !callback(str[:strLen-1]) {
				break
			}
		}
	}
	return nil
}

func readVersionInfoFromStrings(inst *JVMInstallation) bool {
	offset := 0
	size := math.MaxInt64

	// Only process .rodata section for elf files
	f, e := elf.Open(inst.LibJVM)
	if e == nil {
		s := f.Section(".rodata")
		_ = f.Close()
		if s != nil {
			offset = int(s.Offset)
			size = int(s.Size)
		}
	}

	re := regexp.MustCompile(`^(?P<name>OpenJDK.* VM) \((?P<ver>.*)\) for .* JRE \((?P<re_name>.*)\) \((?P<re_ver>.*)\), built`)
	re2 := regexp.MustCompile(`^(?P<name>OpenJDK.* VM) \((?P<ver>.*)\) for .* JRE \((?P<re_name>.*)\), built`)
	re3 := regexp.MustCompile(`^(?P<name>Java HotSpot\(TM\).* VM) \((?P<ver>.*)\) for .* JRE \((?P<re_name>.*)\), built`)
	var zing bool

	e = processStringsFromFile(inst.LibJVM, offset, size, func(str string) bool {
		if strings.Contains(str, "Azul Systems") {
			inst.VersionInfo.VMVendor = "Azul Systems, Inc."
			inst.VersionInfo.RuntimeVendor = "Azul Systems, Inc."
			inst.VersionInfo.RuntimeName = "Zing Runtime Environment for Java Applications"
			inst.VersionInfo.VMName = "Zing 64-Bit Tiered VM"
			zing = true
		} else if strings.Contains(str, "AdoptOpenJDK") {
			inst.VersionInfo.VMVendor = "AdoptOpenJDK"
			inst.VersionInfo.RuntimeVendor = "AdoptOpenJDK"
		} else if re.MatchString(str) {
			match := re.FindAllStringSubmatch(str, 5)
			inst.VersionInfo.VMName = match[0][1]
			inst.VersionInfo.VMVersion = match[0][2]
			inst.VersionInfo.RuntimeName = match[0][3]
			inst.VersionInfo.RuntimeVersion = match[0][4]
		} else if re2.MatchString(str) {
			match := re2.FindAllStringSubmatch(str, 4)
			inst.VersionInfo.VMName = match[0][1]
			inst.VersionInfo.VMVersion = match[0][2]
			inst.VersionInfo.RuntimeName = match[0][1]
			inst.VersionInfo.RuntimeVersion = match[0][3]
		} else if re3.MatchString(str) {
			match := re3.FindAllStringSubmatch(str, 4)
			inst.VersionInfo.VMName = match[0][1]
			inst.VersionInfo.VMVersion = match[0][2]
			inst.VersionInfo.RuntimeName = match[0][1]
			inst.VersionInfo.RuntimeVersion = match[0][3]
			inst.VersionInfo.VMVendor = "Oracle Corporation"
			inst.VersionInfo.RuntimeVendor = "Oracle Corporation"
		} else if zing && strings.Contains(str, "-zing_") {
			inst.VersionInfo.VMVersion = str
			inst.VersionInfo.RuntimeVersion = str
		}

		return inst.VersionInfo.VMVersion == "" || inst.VersionInfo.VMVendor == ""
	})

	if e == nil {
		v := &inst.VersionInfo.RuntimeVersion
		i := strings.IndexAny(*v, "-+_")
		if i > 0 {
			inst.VersionInfo.Version = (*v)[:i]
			return true
		} else if *v != "" {
			inst.VersionInfo.Version = *v
			return true
		}
	}

	return false
}

func readVersionInfoFromOutput(inst *JVMInstallation) bool {
	b, _ := exec.Command(path.Join(inst.JavaHome, "bin/java"), "-XshowSettings:all", "-version").CombinedOutput()
	s := bufio.NewScanner(bytes.NewReader(b))
	s.Split(bufio.ScanLines)
	var res bool
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if idx := strings.Index(line, " = "); idx >= 0 {
			name := line[0:idx]
			value := strings.TrimSpace(line[idx+3:])

			switch name {
			case "java.version":
				inst.VersionInfo.Version = value
			case "java.runtime.name":
				inst.VersionInfo.RuntimeName = value
			case "java.vendor":
				inst.VersionInfo.RuntimeVendor = value
			case "java.runtime.version":
				inst.VersionInfo.RuntimeVersion = value
			case "java.vm.name":
				inst.VersionInfo.VMName = value
			case "java.vm.vendor":
				inst.VersionInfo.VMVendor = value
			case "java.vm.version":
				inst.VersionInfo.VMVersion = value
			default:
			}
		}
	}
	return res
}
func findJavaHome(libjvm string) string {
	if libjvm == "/" {
		return ""
	}
	p := path.Join(libjvm, "bin/java")
	if _, err := os.Stat(p); err == nil || os.IsExist(err) {
		p = path.Join(path.Dir(libjvm), "bin/java")
		if _, err := os.Stat(p); err == nil || os.IsExist(err) {
			return path.Dir(libjvm)
		}
		return libjvm
	}
	return findJavaHome(path.Dir(libjvm))
}

func extractVersionStringsFromClassFileBytes(bytes []byte, info *JVMVersionInfo) {
	classFile := ParseClassFile(bytes)
	cp := classFile.constantPool
	fields := classFile.fields

	for _, f := range fields {
		if name, ok := cp.GetConstantEntry(f.nameIndex).(*ConstantUtf8Entry); ok {
			attrs := f.attributes
			for _, attr := range attrs {
				if attr, ok := attr.(*ConstantValueAttribute); ok {
					if e, ok := cp.GetConstantEntry(attr.value).(*ConstantStringEntry); ok {
						value := fmt.Sprintf("%s", e.Value(cp))
						if value != "" {
							switch strings.ToLower(name.String()) {
							case "Version":
								info.Version = value
							case "RuntimeName":
								info.RuntimeName = value
								info.VMName = value
							case "vendor":
								info.RuntimeVendor = value
								info.VMVendor = value
							case "RuntimeVersion":
								info.RuntimeVersion = value
								info.VMVersion = value
							default:
							}
						}
					}
				}
			}
		}
	}
}

func extractVersionClass(filename string) ([]byte, error) {
	archive, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = archive.Close() }()

	var data []byte
	for _, archiveEntry := range archive.File {
		if archiveEntry.Name == "sun/misc/Version.class" {
			fc, err := archiveEntry.Open()
			if err != nil {
				log.Fatal("Error reading archive entry")
			}
			data, err = io.ReadAll(fc)
			_ = fc.Close()
			if err != nil {
				return nil, err
			} else {
				return data, err
			}
		}
	}

	return nil, errors.New("entry sun/misc/Version.class not found")
}
