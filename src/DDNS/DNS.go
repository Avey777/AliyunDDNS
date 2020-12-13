package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

//My ip info
type IpInfo struct {
	IpAdd    string
	IsUpdate bool
	GetTime  time.Time
}

var MYIP [1]*IpInfo

func getMyIp(urls string) string {
	myip := ""
	reqest, err := http.Get(urls)
	if err == nil {
		defer reqest.Body.Close()
		b, _ := ioutil.ReadAll(reqest.Body)
		return string(b)
	}
	return myip
}

func checkIP(newIP string) {
	if newIP != MYIP[0].IpAdd {
		MYIP[0].IpAdd = newIP
		MYIP[0].IsUpdate = false
		MYIP[0].GetTime = time.Now()
	}

	if !MYIP[0].IsUpdate {
		updateIP()
	}
}

func updateIP() {
	url := fmt.Sprintf("%s%s", "https://xxxx@xxxx.com:xxxx@www.dnsdynamic.org/api/?hostname=xxxx.dnsget.org&myip=", MYIP[0].IpAdd)
	reqest, err := http.Get(url)
	if err != nil {
		return
	}
	defer reqest.Body.Close()
	b, _ := ioutil.ReadAll(reqest.Body)
	spiltB := strings.Split(string(b), " ")
	if spiltB[0] == "good" {
		MYIP[0].IsUpdate = true
	}
}

func main1() {
	//Initialization
	MYIP[0] = new(IpInfo)
	for {
		newIP := getMyIp("http://myip.dnsdynamic.org")
		checkIP(newIP)
		time.Sleep(300 * time.Second)
	}

}
