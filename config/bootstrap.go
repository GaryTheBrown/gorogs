package config

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func loopedMapWrite(cmap map[string]any, keys []string, value any) bool {
	if len(keys) == 0 {
		return false
	}

	if len(keys) == 1 {
		cmap[keys[0]] = value
		return true
	}

	var nextMap map[string]any

	if existingValue, exists := cmap[keys[0]]; exists {
		var ok bool
		if nextMap, ok = existingValue.(map[string]any); !ok {
			return false
		}
	} else {
		nextMap = make(map[string]any)
		cmap[keys[0]] = nextMap
	}

	return loopedMapWrite(nextMap, keys[1:], value)
}

func init() { //_getEnvironFillMap() {}
	prefix := "gorogs."

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, prefix) {
			parts := strings.SplitN(env, "=", 2)
			fullKeys := strings.Split(parts[0], ".")
			if len(parts) == 2 {
				loopedMapWrite(massConfigMap, fullKeys[1:], autoParseType(parts[1]))
			} else if len(parts) == 1 {
				loopedMapWrite(massConfigMap, fullKeys[1:], true)
			}
		}
	}
}

func init() { //_getDisabled() {
	if disabledStr, ok := massConfigMap["disabled"].(string); ok {
		disabledSlice := strings.SplitSeq(strings.ToLower(disabledStr), ",")
		for val := range disabledSlice {
			cleanVal := strings.TrimSpace(val)
			disabled[cleanVal] = true
		}
	}
}

func init() { //_getEnabled() {
	if enabledStr, ok := massConfigMap["enabled"].(string); ok {
		enabledSlice := strings.SplitSeq(strings.ToLower(enabledStr), ",")
		for val := range enabledSlice {
			cleanVal := strings.TrimSpace(val)
			enabled[cleanVal] = true
		}
	}
}

func init() { //_getHostnameAndDomain() {
	tmpHostname := os.Getenv("HOSTNAME")
	if tmpHostname != "" {
		if strings.Contains(tmpHostname, ".") {
			parts := strings.Split(tmpHostname, ".")
			Hostname = parts[0]
			DomainName = strings.Join(parts[1:], ".")
			return
		} else {
			Hostname = tmpHostname
		}
	}
	if Hostname == "" {
		// Attempt to pull from native Linux Kernel Uname maps directly first (PID 1 Safe)
		var buf unix.Utsname
		if err := unix.Uname(&buf); err == nil {
			// Clean out the raw kernel null-terminated byte array mapping elements
			fields := strings.Split(string(buf.Nodename[:]), "\x00")
			Hostname = strings.TrimSpace(fields[0])
		}
	}
	if Hostname == "" {
		if osHost, err := os.Hostname(); err == nil {
			Hostname = osHost
		}
	}
	if Hostname == "" {
		Hostname = "gorogs"
	}
	if file, errFile := os.Open("/etc/resolv.conf"); errFile == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			rLine := scanner.Text()
			if strings.HasPrefix(rLine, "search ") || strings.HasPrefix(rLine, "domain ") {
				fields := strings.Fields(rLine)
				if len(fields) > 1 {
					DomainName = strings.TrimPrefix(strings.TrimSpace(fields[1]), ".")
					return
				}
			}
		}
		if err := scanner.Err(); err != nil {
		}

	}
	DomainName = "local"
}

func init() { //_getSharedRoot() {
	ShareRoot = massConfigMap.Get("shareroot", ConstOriginalShareRoot)
}

func init() { //_getWorkgroup() {
	Workgroup = massConfigMap.Get("workgroup", ConstOriginalWorkgroup)
}

func init() { //_getSystemIP() {
	var err error
	if waitForSystemIP(5*time.Second) == nil {
		return
	}

	ip, err := net.LookupIP(Hostname)
	if err == nil && len(ip) > 0 {
		SystemIP = ip[0]
		return
	}

}

func init() { // _getActiveInterfaces() {
	if interfaceStr, ok := massConfigMap["networkinterfaces"].(string); ok {
		interfaceSlice := strings.SplitSeq(strings.ToLower(interfaceStr), ",")
		for val := range interfaceSlice {
			cleanVal := strings.TrimSpace(val)
			NetworkInterfaces = append(NetworkInterfaces, cleanVal)
		}
		return
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		NetworkInterfaces = []string{"eth0"}
		return
	}

	for _, iface := range interfaces {
		if (iface.Flags & net.FlagUp) == 0 {
			continue
		}

		if (iface.Flags & net.FlagLoopback) != 0 {
			continue
		}

		NetworkInterfaces = append(NetworkInterfaces, iface.Name)
	}
}

func waitForSystemIP(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		interfaces, err := net.Interfaces()
		if err != nil {
			return err
		}

		for _, iface := range interfaces {
			if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
				continue
			}

			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}

			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}

				if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
					SystemIP = ip
					return nil
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for valid non-loopback IP assignment")
}
