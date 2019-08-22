package main

import (
	"github.com/prometheus/procfs"
	"log"
	"strings"
)

func findDescendents(pid int) ([]int, error) {
	//log.Println("**", pid)
	descendents := make([]int, 0)

	children, err := findChildren(pid)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	descendents = children

	// Recursively find children's children
	for i := 0; i < len(children); i++ {
		childPid := children[i]
		childDescendents, err := findDescendents(childPid)
		if err != nil {
			pidStillExists, err2 := pidStillExists(childPid)
			if err2 != nil {
				log.Println(err2)
				return nil, err2
			}
			if !pidStillExists {
				continue
			}
			log.Println(err)
			return nil, err
		}
		descendents = append(descendents, childDescendents...)
	}

	return descendents, nil
}

func pidStillExists(pid int) (bool, error) {
	fs, err := procfs.NewFS(LINIX_PROC)
	if err != nil {
		return false, err
	}
	_, err = fs.Proc(pid)
	if err != nil {
		log.Println(err)
		return false, nil
	}
	return true, nil
}
func findChildren(pid int) ([]int, error) {
	all, err := procfs.AllProcs()
	if err != nil {
		log.Println(err)
		return nil, err
	} else {
		children := make([]int, 0)
		for i := 0; i < len(all); i++ {
			proc := all[i]

			procStat, err := proc.Stat()
			if err != nil {
				if strings.HasSuffix(err.Error(), "no such file or directory") {
					continue
				}
				// Is the error that the proc no longer exists? That is OK (child processes go away)
				pidStillExists, err2 := pidStillExists(pid)
				if err2 != nil {
					log.Println(err2)
					return nil, err2
				}
				if !pidStillExists {
					continue
				}
				log.Println(err)
				return nil, err

			}
			if procStat.PPID == pid {
				children = append(children, procStat.PID)
				//log.Println("\n**", proc)
				//log.Println("  ", procStat)
			}

		}
		return children, err
	}
}
