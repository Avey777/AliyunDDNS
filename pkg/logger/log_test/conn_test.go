package log_test

import (
	"testing"

	. "github.com/Avey777/Goproject/src/Study/logger"
)

func TestConn(t *testing.T) {
	log := NewLogger()
	log.SetLogger("conn", `{"net":"tcp","addr":"192.168.1.9:81"}`)
	log.Info("this is informational to net")
}
