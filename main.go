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
	IntervalSeconds time.Duration `arg`
	Pid             []int         `arg:"required,positional"`
}

type FileInfo struct {
	Pid       int       `json:"pid"`
	Name      string    `json:"file"`
	Size      *int64    `json:"size,omitempty"`
	Timestamp time.Time `json:"time"`
	Deleted   *bool     `json:"deleted,omitempty"`
}

var files map[string]*FileInfo

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	args.IntervalSeconds = 1
	arg.MustParse(&args)

	files = make(map[string]*FileInfo, 100)

	ticker := time.NewTicker(args.IntervalSeconds * time.Second)

	for _ = range ticker.C {
		fs, err := procfs.NewFS("/proc")
		if err != nil {
			log.Println(err)
			break
		}
		for i := 0; i < len(args.Pid); i++ {
			p, err := fs.Proc(args.Pid[i])
			if err != nil {
				log.Fatal(err)
			}

			fds, err := p.FileDescriptorTargets()
			if err != nil {
				log.Fatal("could not get process stat", err)
			}

			//log.Println(fds)
			for j := 0; j < len(fds); j++ {
				filename := fds[j]
				if strings.HasPrefix(filename, "/") {
					if !specialFile(filename) {
						handleFile(filename, args.Pid[i])
					}
				}
			}
		}
		err = checkIfFilesHaveDisappeared()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func handleFile(filename string, pid int) error {
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
			return err
		}
		fi.Timestamp = time.Now()
	} else {
		// Existing file
		timestamp := time.Now()
		filesize, err := getFileSize(filename)
		if err != nil {
			return err
		}
		if *filesize != *fi.Size {
			//log.Println(filename, filesize, fi.Size)
			b, err := json.Marshal(*fi)
			if err != nil {
				return err
			}
			os.Stdout.Write(b)
			os.Stdout.Write([]byte("\n"))
			fi.Size = filesize
			fi.Timestamp = timestamp
		}
	}
	return nil
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

func checkIfFilesHaveDisappeared() error {
	for _, fi := range files {
		_, err := os.Stat(fi.Name)
		if os.IsNotExist(err) {
			fi.Timestamp = time.Now()
			delete(files, fi.Name)
			fi.Deleted = new(bool)
			*fi.Deleted = true
			b, err := json.Marshal(*fi)
			if err != nil {
				return err
			}
			os.Stdout.Write(b)
			os.Stdout.Write([]byte("\n"))
		} else {
			return err
		}

	}
	return nil
}
