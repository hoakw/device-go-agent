// The healthtype package defines structures for health information.
package healthType

import (
	"log"
	insp_net "net"
	"time"
)

// Struct definition for NetworkIP information.
type NetworkInterface struct {
	PrivateIP string
	PublicIP  string
}

// Struct defining the format of logs.
//   - Warn: Warning log type.
//   - Info: General information log type.
//   - Error: Error type information.
type Logger struct {
	Warn  *log.Logger
	Info  *log.Logger
	Error *log.Logger
}

// ConfigInfo struct defines the configuration file used by SDT Cloud on the device.
//   - AssetCode: Device serial number.
//   - MqttUrl: MQTT URL of SDT Cloud.
//   - ProjectCode: ID of the project to which the device belongs.
//   - ServiceCode: Service code of SDT Cloud.
//   - ServiceType: Service type of SDT Cloud.
type ConfigInfo struct {
	AssetCode   string `json:"assetcode"`
	MqttUrl     string `json:"mqtturl"`
	ProjectCode string `json:"projectcode"`
	ServiceCode string `json:"servicecode"`
	ServiceType string `json:"servicetype"`
	ServerIp    string `json:"serverip"`
}

// Struct defining the environment information of the Device-Health agent.
//   - MqttType: MQTT service type used by the agent.
//   - ArchType: Architecture type of the device.
//   - RootPath: Root path of the BWC.
type HealthService struct {
	MqttType string
	ArchType string
	RootPath string
}

// Struct definition for CPU information.
//   - Cpu: CPU usage percentage.
//   - Total: Total number of CPU cores.
//   - Time: Time of collection.
type NodeCpu struct {
	Cpu   float64
	Total int
	Time  time.Time
}

// Struct definition for memory information.
//   - Mem: Memory usage percentage.
//   - Total: Total memory capacity.
//   - Time: Time of collection.
type NodeMem struct {
	Mem   float64
	Total string
	Time  time.Time
}

// This struct defines the process information.
//   - Id: Process ID.
//   - Cpu: CPU usage percentage.
//   - Memory: Memory usage percentage.
//   - Context: Process name.
//   - Time: Timestamp of collection.
type ProcInfo struct {
	Id      string
	Cpu     float32
	Memory  float32
	Context string
	Time    time.Time
}

// This is a Struct defining disk information.
//   - Name: Disk name.
//   - Totalsize: Total size of the disk.
//   - Used: Disk usage.
//   - UsedPercent: Disk usage percentage.
//   - Mountpoint: Disk mount point.
//   - Time: Time of collection.
type DiskInfo struct {
	Name        string
	Totalsize   string
	Used        float64
	UsedPercent string
	Mountpoint  string
	Time        time.Time
}

// This is a Struct defining network information.
//   - Index: Network index.
//   - Name: Network name.
//   - Address: Network IP address.
//   - Mtu: Network MTU value.
//   - HardwareAddr: Hardware address.
//   - Time: Time of collection.
type NetInfo struct {
	Index        int
	Name         string
	Address      string
	Mtu          int
	HardwareAddr insp_net.HardwareAddr
	Time         time.Time
}

// This is a Struct defining serial information.
//   - Index: Serial index.
//   - Uart: Serial Uart.
//   - Port: Serial port.
//   - Irq: Serial IRQ value.
//   - Tx: Serial transmit value.
//   - Rx: Serial receive value.
//   - Time: Time of collection.
type SerialInfo struct {
	Index string
	Uart  string
	Port  string
	Irq   int
	Tx    int
	Rx    int
	Time  time.Time
}
