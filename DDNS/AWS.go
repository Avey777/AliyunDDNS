package main1

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const GETIPURL string = "http://ddns.oray.com/checkip"
const UPDATEIP string = "http://%s:%s@ddns.oray.com/ph/update?hostname=%s&myip=%s"

var logger *log.Logger = nil

func getMyIp(url string) (string, int) {
	reqest, err := http.Get(url)
	if err == nil {
		defer reqest.Body.Close()
		b, _ := ioutil.ReadAll(reqest.Body)
		reg := regexp.MustCompile(`\d{1,3}.\d{1,3}.\d{1,3}.\d{1,3}`)
		IpTemp := reg.FindString(string(b))
		return IpTemp, 0
	} else {
		return "", 1
	}
}

func updateDDNS(IpAdd string) int {
	url := fmt.Sprintf(UPDATEIP, os.Args[1], os.Args[2], os.Args[3], IpAdd)
	if logger != nil {
		logger.Println(url, os.Getpid())
	}
	reqest, err := http.Get(url)
	if err != nil {
		return 1
	} else {
		defer reqest.Body.Close()
		b, _ := ioutil.ReadAll(reqest.Body)
		spiltB := strings.Split(string(b), " ")
		fmt.Print(spiltB[0])
		if spiltB[0] == "good" {
			return 0
		} else if spiltB[0] == "nochg" {
			return 0
		} else {
			return 1
		}
	}
}

func main() {

	if 4 != len(os.Args) {
		fmt.Printf("the input param: user password url\n")
		return
	}

	logfile, logerr := os.OpenFile("/var/log/ddnsupdate", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0)
	if nil != logerr {
		fmt.Printf("%s\n", logerr.Error())
		return
	}
	defer logfile.Close()

	if os.Getppid() != 1 {
		filePath, _ := filepath.Abs(os.Args[0])
		fmt.Printf("filePath=%s\n", filePath)
		cmd := exec.Command(filePath, os.Args[1:]...)
		cmd.Start()
		return
	}
	logger = log.New(logfile, "", log.Ldate|log.Ltime|log.Llongfile)
	logger.Println("Start ddnsupdate", os.Getpid())

	for {
		newIP, errorflg := getMyIp(GETIPURL)
		if errorflg == 0 {
			if 0 == updateDDNS(newIP) {
				logger.Println("updateDDNS OK", os.Getpid())
				break
			} else {
				logger.Println("updateDDNS error", os.Getpid())
			}
		} else {
			logger.Println("getMyIp error", os.Getpid())
		}

		time.Sleep(60 * 60 * time.Second)
	}
	logger.Println("exit ddnsupdate", os.Getpid())
}
