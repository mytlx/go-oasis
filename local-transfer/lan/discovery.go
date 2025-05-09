package lan

import (
	"encoding/json"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type DeviceInfo struct {
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	DeviceType string `json:"device_type"`
	IP         string `json:"ip"`
	Timestamp  string `json:"timestamp"`
}
type DiscoveredDevice struct {
	DeviceName string    `json:"deviceName"`
	DeviceType string    `json:"deviceType"`
	IP         string    `json:"ip"`
	LastSeen   time.Time `json:"lastSeen"`
}

var (
	discoveredDevices   = make(map[string]DiscoveredDevice)
	discoveredDevicesMu sync.RWMutex
)

var (
	devices   = make(map[string]DeviceInfo)
	devicesMu sync.Mutex
)

// StartBroadcaster 启动设备广播
func StartBroadcaster(info DeviceInfo, interval time.Duration) {
	go func() {
		addr, _ := net.ResolveUDPAddr("udp", "255.255.255.255:9999")
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			log.Println("[UDP] 广播失败:", err)
			return
		}
		defer conn.Close()

		for {
			info.Timestamp = time.Now().Format(time.RFC3339)
			b, _ := json.Marshal(info)
			conn.Write(b)
			time.Sleep(interval)
		}
	}()
}

// StartListener 启动监听器
func StartListener(onDiscover func(DeviceInfo)) {
	go func() {
		// // 获取本机的广播地址
		// broadcastAddr := GetBroadcastAddress()
		// if broadcastAddr == "" {
		// 	log.Fatal("无法获取广播地址")
		// 	return
		// }
		// addr := broadcastAddr + ":9999" // 使用计算得到的广播地址
		// udpAddr, err := net.ResolveUDPAddr("udp", addr)
		// if err != nil {
		// 	log.Fatal("无法解析 UDP 地址:", err)
		// 	return
		// }

		addr, _ := net.ResolveUDPAddr("udp", ":9999")
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			log.Fatal("无法监听 UDP 端口:", err)
			return
		}
		defer conn.Close()

		log.Println("正在监听 UDP 广播...")

		// 循环接收数据包
		buf := make([]byte, 2048)
		for {
			n, src, err := conn.ReadFromUDP(buf)
			if err != nil {
				log.Println("接收数据包错误:", err)
				continue
			}

			// 解析设备信息
			var deviceInfo DeviceInfo
			if err := json.Unmarshal(buf[:n], &deviceInfo); err != nil {
				log.Println("解析设备信息失败:", err)
				continue
			}
			deviceInfo.IP = src.IP.String()
			devicesMu.Lock()
			devices[deviceInfo.DeviceID] = deviceInfo
			devicesMu.Unlock()
			onDiscover(deviceInfo)

			// 更新已发现设备列表
			UpdateDiscoveredDevice(deviceInfo.IP, deviceInfo.DeviceName, deviceInfo.DeviceType)
			log.Printf("接收到来自 %s 的设备信息: %s, %s, %s\n",
				addr, deviceInfo.DeviceName, deviceInfo.DeviceType, deviceInfo.IP)
		}
	}()
}

// StartCleaner 定期清理
func StartCleaner(timeout time.Duration) {
	go func() {
		for {
			time.Sleep(10 * time.Second)
			now := time.Now()
			devicesMu.Lock()
			for id, d := range devices {
				t, err := time.Parse(time.RFC3339, d.Timestamp)
				if err != nil || now.Sub(t) > timeout {
					delete(devices, id)
				}
			}
			devicesMu.Unlock()
		}
	}()
}

func GetBroadcastAddress() string {
	// 获取本机所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatal("无法获取网络接口:", err)
	}

	for _, iface := range interfaces {
		// 排除 loopback 和不启用的网络接口
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			log.Println("无法获取接口地址:", err)
			continue
		}

		// 找到第一个合适的 IP 地址并计算广播地址
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				log.Println("无法解析地址:", err)
				continue
			}

			// 这里假设 IP 地址属于 192.168.x.x 网段
			// 可以根据需要优化计算广播地址的算法
			if strings.HasPrefix(ip.String(), "192.168") {
				// 使用网段后进行广播地址计算
				ip = ip.Mask(net.CIDRMask(24, 32))                                                  // 使用 255.255.255.0 子网掩码
				broadcast := net.IPv4(ip[0]|^uint8(255), ip[1]|^uint8(255), ip[2]|^uint8(255), 255) // 计算广播地址
				return broadcast.String()
			}
		}
	}
	return ""
}

func GetOnlineDevices() []DeviceInfo {
	devicesMu.Lock()
	defer devicesMu.Unlock()
	var list []DeviceInfo
	for _, d := range devices {
		list = append(list, d)
	}
	return list
}

// UpdateDiscoveredDevice 更新设备信息
func UpdateDiscoveredDevice(ip, name, deviceType string) {
	discoveredDevicesMu.Lock()
	defer discoveredDevicesMu.Unlock()
	discoveredDevices[ip] = DiscoveredDevice{
		DeviceName: name,
		DeviceType: deviceType,
		IP:         ip,
		LastSeen:   time.Now(),
	}
}

// GetDiscoveredDevices 获取设备列表（只返回 30 秒内活跃的）
func GetDiscoveredDevices() []DiscoveredDevice {
	discoveredDevicesMu.RLock()
	defer discoveredDevicesMu.RUnlock()
	var list []DiscoveredDevice
	cutoff := time.Now().Add(-30 * time.Second)
	for _, d := range discoveredDevices {
		if d.LastSeen.After(cutoff) {
			list = append(list, d)
		}
	}
	return list
}
