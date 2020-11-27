package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FileLogger struct {
	sync.RWMutex
	FileWriter *os.File

	Filename   string `json:"filename"`
	Append     bool   `json:"append"`
	MaxLines   int    `json:"maxlines"`
	MaxSize    int    `json:"maxsize"`
	Daily      bool   `json:"daily"`
	MaxDays    int64  `json:"maxdays"`
	Level      string `json:"level"`
	PermitMask string `json:"permit"`

	LogLevel             int
	MaxSizeCurSize       int
	MaxLinesCurLines     int
	DailyOpenDate        int
	DailyOpenTime        time.Time
	FileNameOnly, suffix string
}

// Init file logger with json config.
// jsonConfig like:
//	{
//	"filename":"log/app.log",
//	"maxlines":10000,
//	"maxsize":1024,
//	"daily":true,
//	"maxdays":15,
//	"rotate":true,
//  	"permit":"0600"
//	}
func (f *FileLogger) Init(jsonConfig string) error {
	fmt.Printf("FileLogger Init:%s\n", jsonConfig)
	if len(jsonConfig) == 0 {
		return nil
	}
	err := json.Unmarshal([]byte(jsonConfig), f)
	if err != nil {
		return err
	}
	if len(f.Filename) == 0 {
		return errors.New("jsonconfig must have filename")
	}
	f.suffix = filepath.Ext(f.Filename)
	f.FileNameOnly = strings.TrimSuffix(f.Filename, f.suffix)
	f.MaxSize *= 1024 * 1024 // 将单位转换成MB
	if f.suffix == "" {
		f.suffix = ".log"
	}
	if l, ok := LevelMap[f.Level]; ok {
		f.LogLevel = l
	}
	err = f.NewFile()
	return err
}

func (f *FileLogger) NeedCreateFresh(size int, day int) bool {
	return (f.MaxLines > 0 && f.MaxLinesCurLines >= f.MaxLines) ||
		(f.MaxSize > 0 && f.MaxSizeCurSize+size >= f.MaxSize) ||
		(f.Daily && day != f.DailyOpenDate)

}

// WriteMsg write logger message into file.
func (f *FileLogger) LogWrite(when time.Time, msgText interface{}, level int) error {
	msg, ok := msgText.(string)
	if !ok {
		return nil
	}
	if level > f.LogLevel {
		return nil
	}

	day := when.Day()
	msg += "\n"
	if f.Append {
		f.RLock()
		if f.NeedCreateFresh(len(msg), day) {
			f.RUnlock()
			f.Lock()
			if f.NeedCreateFresh(len(msg), day) {
				if err := f.CreateFreshFile(when); err != nil {
					fmt.Fprintf(os.Stderr, "CreateFreshFile(%q): %s\n", f.Filename, err)
				}
			}
			f.Unlock()
		} else {
			f.RUnlock()
		}
	}

	f.Lock()
	_, err := f.FileWriter.Write([]byte(msg))
	if err == nil {
		f.MaxLinesCurLines++
		f.MaxSizeCurSize += len(msg)
	}
	f.Unlock()
	return err
}

func (f *FileLogger) CreateLogFile() (*os.File, error) {
	// Open the log file
	perm, err := strconv.ParseInt(f.PermitMask, 8, 64)
	if err != nil {
		return nil, err
	}
	fd, err := os.OpenFile(f.Filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.FileMode(perm))
	if err == nil {
		// Make sure file perm is user set perm cause of `os.OpenFile` will obey umask
		os.Chmod(f.Filename, os.FileMode(perm))
	}
	return fd, err
}

func (f *FileLogger) NewFile() error {
	file, err := f.CreateLogFile()
	if err != nil {
		return err
	}
	if f.FileWriter != nil {
		f.FileWriter.Close()
	}
	f.FileWriter = file

	fInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("get stat err: %s", err)
	}
	f.MaxSizeCurSize = int(fInfo.Size())
	f.DailyOpenTime = time.Now()
	f.DailyOpenDate = f.DailyOpenTime.Day()
	f.MaxLinesCurLines = 0
	if f.MaxSizeCurSize > 0 {
		count, err := f.lines()
		if err != nil {
			return err
		}
		f.MaxLinesCurLines = count
	}
	return nil
}

func (f *FileLogger) lines() (int, error) {
	fd, err := os.Open(f.Filename)
	if err != nil {
		return 0, err
	}
	defer fd.Close()

	buf := make([]byte, 32768) // 32k
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := fd.Read(buf)
		if err != nil && err != io.EOF {
			return count, err
		}

		count += bytes.Count(buf[:c], lineSep)

		if err == io.EOF {
			break
		}
	}

	return count, nil
}

// new file name like  xx.2013-01-01.001.log
func (f *FileLogger) CreateFreshFile(logTime time.Time) error {
	// file exists
	// Find the next available number
	num := 1
	fName := ""
	rotatePerm, err := strconv.ParseInt(f.PermitMask, 8, 64)
	if err != nil {
		return err
	}

	_, err = os.Lstat(f.Filename)
	if err != nil {
		// 初始日志文件不存在，无需创建新文件
		goto RESTART_LOGGER
	}
	// 日期变了， 说明跨天，重命名时需要保存为昨天的日期
	if f.DailyOpenDate != logTime.Day() {
		for ; err == nil && num <= 999; num++ {
			fName = f.FileNameOnly + fmt.Sprintf(".%s.%03d%s", f.DailyOpenTime.Format("2006-01-02"), num, f.suffix)
			_, err = os.Lstat(fName)
		}
	} else { //如果仅仅是文件大小或行数达到了限制，仅仅变更后缀序号即可
		for ; err == nil && num <= 999; num++ {
			fName = f.FileNameOnly + fmt.Sprintf(".%s.%03d%s", logTime.Format("2006-01-02"), num, f.suffix)
			_, err = os.Lstat(fName)
		}
	}

	if err == nil {
		return fmt.Errorf("Cannot find free log number to rename %s", f.Filename)
	}
	f.FileWriter.Close()

	// 当创建新文件标记为true时
	// 当日志文件超过最大限制行
	// 当日志文件超过最大限制字节
	// 当日志文件隔天更新标记为true时
	// 将旧文件重命名，然后创建新文件
	err = os.Rename(f.Filename, fName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "os.Rename %s to %s err:%s\n", f.Filename, fName, err.Error())
		goto RESTART_LOGGER
	}

	err = os.Chmod(fName, os.FileMode(rotatePerm))

RESTART_LOGGER:

	startLoggerErr := f.NewFile()
	go f.DeleteOldLog()

	if startLoggerErr != nil {
		return fmt.Errorf("Rotate StartLogger: %s", startLoggerErr)
	}
	if err != nil {
		return fmt.Errorf("Rotate: %s", err)
	}
	return nil
}

func (f *FileLogger) DeleteOldLog() {
	dir := filepath.Dir(f.Filename)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) (returnErr error) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Unable to delete old log '%s', error: %v\n", path, r)
			}
		}()

		if info == nil {
			return
		}

		if f.MaxDays != -1 && !info.IsDir() && info.ModTime().Add(24*time.Hour*time.Duration(f.MaxDays)).Before(time.Now()) {
			if strings.HasPrefix(filepath.Base(path), filepath.Base(f.FileNameOnly)) &&
				strings.HasSuffix(filepath.Base(path), f.suffix) {
				os.Remove(path)
			}
		}
		return
	})
}

func (f *FileLogger) Destroy() {
	f.FileWriter.Close()
}

func init() {
	Register(AdapterFile, &FileLogger{
		Daily:      true,
		MaxDays:    7,
		Append:     true,
		LogLevel:   LevelDebug,
		PermitMask: "0777",
		MaxLines:   10,
		MaxSize:    10 * 1024 * 1024,
	})
}
