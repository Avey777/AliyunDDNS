package main

import (
	// log "Goproject/src/Study/logger"
	// "aliyunddns/pkg/aliyungo/dns" //第三方库
	log "aliyunddns/pkg/logger"
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"
	"time"

	// log "github.com/Avey777/Goproject/src/Study/logger"

	"github.com/denverdino/aliyungo/dns"
	"gopkg.in/yaml.v2"
)

func main() {

	// a := getMyIPV6()
	// fmt.Printf(a)

	// var c conf
	// conf := c.getConf()
	// a := conf.RR
	// fmt.Printf(a)

	// ddnsipv6() //允许DDNS

	// for {
	// 	//调试
	// 	ipv6 := ReadLineFile("..//AliyunDDNS/readipv6.txt")
	// 	//实际使用
	// 	// ipv6 := ReadLineFile("readipv6.txt")
	// 	fmt.Printf("yamlipv6:", ipv6)
	// 	if ipv6 != getMyIPV6() {
	// 		WriteLine(getMyIPV6())
	// 		ddnsipv6()
	// 	} else {
	// 		log.Info("ipv6地址未改变", time.Now())
	// 	}
	// 	time.Sleep(time.Second * 3)
	// }

	// ipv6 := ReadLineFile("readipv6.txt")
	// fmt.Printf(ipv6)
	// if ipv6 != getMyIPV6() {
	// 	WriteLine(getMyIPV6())
	// 	ddnsipv6()
	// } else {
	// 	log.Info("ipv6地址未改变", time.Now())
	// }

	// yamlarr()
	tickTimerChan()

}

//Conf 定义域名相关配置信息
type conf struct {
	AccessKeyID     string `yaml:"AccessKeyID"`
	AccessKeySecret string `yaml:"AccessKeySecret"`
	DomainName      string `yaml:"DomainName"` //主域名
	RR              string `yaml:"RR"`         //子域名前缀
}

//yaml读取函数
func (c *conf) getConf() *conf {
	// yamlFile, err := ioutil.ReadFile("../AliyunDDNS/AliyunDDNS/conf.yaml") //调试
	yamlFile, err := ioutil.ReadFile("conf.yaml") //实际使用
	if err != nil {
		fmt.Println(err.Error())
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		fmt.Println(err.Error())
	}
	return c
}

//获取ipv6地址
func getMyIPV6() string {
	s, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range s {
		i := regexp.MustCompile(`(\w+:){7}\w+`).FindString(a.String())
		if strings.Count(i, ":") == 7 {
			return i
		}
	}
	return ""
}

//定时打印时间
func tickTimerChan() {
	ticker := time.NewTicker(time.Second * time.Duration(3))
	var tickChan = make(chan int)
	func() {
		for t := range ticker.C {
			// fmt.Println("im doing some biz")
			fmt.Println("Tick ar:", t.Format("2006.01.02 15:04:05"))
			// ipv6 := ReadLineFile("../AliyunDDNS/AliyunDDNS/readipv6.txt") //调试
			ipv6 := ReadLineFile("readipv6.txt") // 正式
			log.Info("readipv6:", ipv6)
			if ipv6 == getMyIPV6() {

				log.Info("ipv6地址未改变", time.Now())
			} else {
				WriteLine(getMyIPV6())
				yamlarr()
				log.Info("ipv6地址更新", time.Now())
			}
		}
		tickChan <- 0
	}()
	defer ticker.Stop()
	select {
	case <-tickChan:
		fmt.Println("tickChan")
	}
	fmt.Println("ticker stopped")
}

//处理yaml文件信息, 转换字符串为数组
func yamlarr() {
	var con conf
	yaml := con.getConf()
	// fmt.Println(yaml)
	//逗号分隔字符串
	sep := ","
	arr := strings.Split(yaml.RR, sep)
	// fmt.Println(arr)

	for _, arr := range arr {
		ddnsipv6(arr)
		// fmt.Println(RRNEW)

		fmt.Printf("v2的数据类型为：%T\n", arr)

	}

	// RRNEW := []string{arr}
	// for i := 0; i < len(RRNEW); i++ {
	// 	// ddnsipv6()
	// 	fmt.Println(RRNEW)
	// }

}

//阿里云DDNS
func ddnsipv6(RRNEW string) {
	var c conf
	yaml := c.getConf()

	//本地ipv6地址
	publicIP := getMyIPV6()
	//DomainName := "amiemall.com"
	DomainName := yaml.DomainName

	// // RR := "ipv6"
	// RRNEW := yaml.RR
	// log.Info("RRNEW", RRNEW)

	AccessKeyID := yaml.AccessKeyID
	AccessKeySecret := yaml.AccessKeySecret
	fmt.Printf(AccessKeyID, AccessKeySecret)

	// 连接阿里云服务器，获取DNS信息
	client := dns.NewClient(AccessKeyID, AccessKeySecret) //阿里云key值
	client.SetDebug(false)
	domainInfo := new(dns.DescribeDomainRecordsArgs)
	domainInfo.DomainName = DomainName
	oldRecord, err := client.DescribeDomainRecords(domainInfo)
	if err != nil {
		fmt.Println("链接错误")
		fmt.Print(err)
		return
	}

	var exsitRecordID string
	for _, record := range oldRecord.DomainRecords.Record {
		if record.DomainName == DomainName && record.RR == RRNEW {
			if record.Value == publicIP {
				fmt.Println("当前配置解析地址与公网IP相同，不需要修改。")
				return
			}
			exsitRecordID = record.RecordId
		}
	}

	if 0 < len(exsitRecordID) {
		// 有配置记录，则匹配配置文件，进行更新操作
		updateRecord := new(dns.UpdateDomainRecordArgs)
		updateRecord.RecordId = exsitRecordID
		updateRecord.RR = RRNEW
		updateRecord.Value = publicIP
		updateRecord.Type = dns.AAAARecord //ipv6 使用ARecord
		rsp := new(dns.UpdateDomainRecordResponse)
		rsp, err := client.UpdateDomainRecord(updateRecord)
		if nil != err {
			fmt.Println("修改解析失败", err)
		} else {
			fmt.Println("修改解析成功", rsp)
		}
	} else {
		// 没有找到配置记录，那么就新增一个
		newRecord := new(dns.AddDomainRecordArgs)
		newRecord.DomainName = DomainName
		newRecord.RR = RRNEW
		newRecord.Value = publicIP
		newRecord.Type = dns.AAAARecord //ipv6 使用ARecord
		rsp := new(dns.AddDomainRecordResponse)
		rsp, err = client.AddDomainRecord(newRecord)
		if nil != err {
			fmt.Println("添加DNS解析失败", err)
		} else {
			fmt.Println("添加DNS解析成功", rsp)
		}
	}

}

//向文件中写入内容
func WriteLine(content string) {
	//调试
	// ioutil.WriteFile("../AliyunDDNS/AliyunDDNS/readipv6.txt", []byte(content), 0777) //如果文件a.txt已经存在那么会忽略权限参数，清空文件内容。文件不存在会创建文件赋予权限
	//实际使用
	ioutil.WriteFile("readipv6.txt", []byte(content), 0777) //如果文件a.txt已经存在那么会忽略权限参数，清空文件内容。文件不存在会创建文件赋予权限
}

//读取txt文件
func ReadLineFile(fileName string) string {
	var connect string
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("打开文件失败")
	} else {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fmt.Println("打印文件内容(ipv6)：", scanner.Text())
			connect = scanner.Text()
		}
	}
	return connect
}

//获取当前文件所在路径
func CurrentfilePath() string {
	_, filename, _, ok := runtime.Caller(1)
	var cwdPath string
	if ok {
		cwdPath = path.Join(path.Dir(filename), "") // the the main function file directory
	} else {
		cwdPath = "./"
	}
	return cwdPath
}
