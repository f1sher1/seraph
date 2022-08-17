package plugin

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
)

func md5V(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func GetSockPath(name string) string {
	sockSuffix := name
	pos := strings.Index(name, ".")
	if pos > -1 {
		sockSuffix = sockSuffix[:pos]
	}
	sysType := runtime.GOOS
	var sockPath string
	if sysType == "windows" {
		sockPath = path.Join("D:", fmt.Sprintf("plugin_%s_%s.sock", sockSuffix, md5V(sockSuffix)))
	} else {
		sockPath = path.Join(os.TempDir(), fmt.Sprintf("plugin_%s_%s.sock", sockSuffix, md5V(sockSuffix)))
	}
	return fmt.Sprintf("unix://%s", sockPath)
}
