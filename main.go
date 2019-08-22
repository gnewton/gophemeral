// gophemeral

package main

import (
	//"fmt"
	"encoding/json"
	"github.com/alexflint/go-arg"
	"github.com/prometheus/procfs"
	"log"
	"os"
	"strings"
	"time"
)

var args struct {
	IntervalSeconds time.Duration `arg:""`
	Pid             []int         `arg:"required,positional"`
}

type FileInfo struct {
	Pid       int       `json:"pid"`
	ParentPid int       `json:"parent_pid,omitempty"`
	Name      string    `json:"file"`
	Size      *int64    `json:"size,omitempty"`
	Timestamp time.Time `json:"time"`
	Deleted   *bool     `json:"deleted,omitempty"`
	Opened    *bool     `json:"opened,omitempty"`
}

const LINIX_PROC = "/proc"

func main() {
	//var files map[string]*FileInfo
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	args.IntervalSeconds = 5
	arg.MustParse(&args)

	files := make(map[string]*FileInfo, 100)

	// Run immediately
	err := checkFiles(args.Pid, files)
	if err != nil {
		log.Fatal(err)
	}

	ticker := time.NewTicker(args.IntervalSeconds * time.Second)

	for _ = range ticker.C {
		err := checkFiles(args.Pid, files)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func checkFiles(pids []int, files map[string]*FileInfo) error {
	fs, err := procfs.NewFS(LINIX_PROC)
	if err != nil {
		log.Println(err)
		return err
	}

	descendentPids := make([]int, 0)
	for i := 0; i < len(pids); i++ {
		pid := pids[i]
		p, err := fs.Proc(pid)
		if err != nil {
			log.Println(err)
			return err
		}

		fds, err := p.FileDescriptorTargets()
		if err != nil {
			log.Println(err)
			return err
		}

		//log.Println(fds)
		for j := 0; j < len(fds); j++ {
			filename := fds[j]
			if strings.HasPrefix(filename, "/") {
				if !specialFile(filename) {
					err := checkIfFileHasChangedSize(filename, pid, files)
					if err != nil {
						return err
					}
					//log.Println("\n\n********************************************************")
					descendents, err := findDescendents(pid)
					if err != nil {
						log.Println(err)
						return err
					}
					descendentPids = append(descendentPids, descendents...)
				}
			}
		}
	}
	err = checkIfFilesHaveDisappeared(files)
	if err != nil {
		log.Println(err)
		return err
	}

	// Add the (new) descendent PIDs to be checked next ticker
	pids = append(pids, descendentPids...)
	return nil
}

func checkIfFileHasChangedSize(filename string, pid int, files map[string]*FileInfo) error {
	var err error
	var fi *FileInfo
	var ok bool

	// New file
	if fi, ok = files[filename]; !ok {
		fi = new(FileInfo)
		files[filename] = fi
		fi.Name = filename
		fi.Pid = pid
		fi.Size, err = getFileSize(filename)
		if err != nil {
			if strings.HasSuffix(err.Error(), "no such file or directory") {
				log.Println(err)
				fi.Deleted = new(bool)
				*fi.Deleted = true
				log.Println("printing")
				outputRecord(fi)
				return nil
			}
			log.Println(err)
			return err
		}
		fi.Opened = new(bool)
		*fi.Opened = true
		outputRecord(fi)
		fi.Opened = nil
		fi.Timestamp = time.Now()
	} else {
		// Existing file
		timestamp := time.Now()
		filesize, err := getFileSize(filename)
		if err != nil {
			// File no longer exists
			if strings.HasSuffix(err.Error(), "no such file or directory") {
				fi.Deleted = new(bool)
				*fi.Deleted = true
				log.Println("printing")
				outputRecord(fi)
				return nil
			}
			return err
		}
		if *filesize != *fi.Size {
			log.Println("printing")
			err = outputRecord(fi)
			if err != nil {
				return err
			}

			fi.Size = filesize
			fi.Timestamp = timestamp
		}
	}
	return nil
}

func outputRecord(fi *FileInfo) error {
	b, err := json.Marshal(*fi)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(b)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write([]byte("\n"))
	return err

}

func getFileSize(filename string) (size *int64, err error) {
	fi, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	tmp := fi.Size()
	size = &tmp
	return size, nil
}

func checkIfFilesHaveDisappeared(files map[string]*FileInfo) error {
	for _, fi := range files {
		_, err := os.Stat(fi.Name)
		if os.IsNotExist(err) {
			fi.Timestamp = time.Now()
			delete(files, fi.Name)
			fi.Deleted = new(bool)
			*fi.Deleted = true
			log.Println("printing")
			err = outputRecord(fi)
			if err != nil {
				return err
			}
		} else {
			return err
		}

	}
	return nil
}
