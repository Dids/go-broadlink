package broadlink

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

// FIXME: This needs heavy refactoring, and the idea right now is to get
//        the auto-discovery working, which means we'll need to send a special
//        packet to the broadcast address, which the Broadlink device will pickup
//        on and send a message back, which is how we can figure out its IP address
//
//        https://github.com/mjg59/python-broadlink/blob/master/broadlink/__init__.py

// BroadcastAddress is the local (LAN) broadcast address
const BroadcastAddress = "255.255.255.255"

// Device represents a single Broadlink device
type Device struct {
	IP string
}

// New creates and returns a new Broadlink device instance
func New() *Device {
	device := &Device{}

	device.IP = device.discover()
	if device.IP == "" {
		log.Fatal("Unable to find device IP, cannot continue")
	}

	return device
}

// TODO: Read up on this here:
//       https://stackoverflow.com/questions/26028700/write-to-client-udp-socket-in-go

// TODO: Protocol here:
//       https://github.com/mjg59/python-broadlink/blob/master/protocol.md

// TODO: Next we'll need to implement sending arbitrary payloads:
//       https://github.com/mjg59/python-broadlink/blob/master/broadlink/__init__.py#L229

func (device *Device) discover() string {
	localAddress, err := net.ResolveUDPAddr("udp4", getAddressString("0.0.0.0", 0))
	handleErr(err)
	log.Println("Local address:", localAddress.String())

	log.Println("Connecting to server..")
	// connection, err := net.DialUDP("udp4", localAddress, broadcastAddress)
	connection, err := net.ListenPacket("udp4", ":0")
	if connection != nil {
		defer connection.Close()
	}
	handleErr(err)
	log.Println("Connected to server:", connection.LocalAddr().String())

	device.sendPacket(connection)

	log.Println("Waiting to receive a message..")
	response := make([]byte, 1024)
	length, addr, err := connection.ReadFrom(response)
	handleErr(err)
	log.Println("Read length:", length)
	log.Println("Read address:", addr)
	log.Println("Received message:", response)

	deviceAddress := addr.String()
	log.Println("deviceAddress:", deviceAddress)

	deviceMacAddress := response[0x3a:0x40]
	log.Println("deviceMacAddress:", deviceMacAddress)

	deviceType := int(response[0x34]) | int(response[0x35])<<8
	log.Println("deviceType:", fmt.Sprintf("0x%x", deviceType))

	for i := range Devices {
		d := Devices[i]
		if deviceType == d {
			log.Println("Compatible device found:", d)
			return deviceAddress
		}
	}

	return ""
}

func (device *Device) sendPacket(connection net.PacketConn) {
	address := strings.Split(connection.LocalAddr().String(), ":")
	ip := address[0]
	port, _ := strconv.Atoi(address[1])

	packet := make([]byte, 0x30)

	now := time.Now()
	log.Println("now:", now)

	_, timezone := now.Zone()
	log.Println("timezone:", timezone)

	timezone = int(timezone / 3600)
	log.Println("timezone (adjusted):", timezone)

	year := now.Year()
	log.Println("year:", year)

	if timezone < 0 {
		packet[0x08] = byte(0xff + timezone - 1)
		packet[0x09] = 0xff
		packet[0x0a] = 0xff
		packet[0x0b] = 0xff
	} else {
		packet[0x08] = byte(timezone)
		packet[0x09] = 0
		packet[0x0a] = 0
		packet[0x0b] = 0
	}

	packet[0x0c] = byte(year & 0xff)
	packet[0x0d] = byte(year >> 8)
	packet[0x0e] = byte(now.Minute())
	packet[0x0f] = byte(now.Hour())
	subyear, _ := strconv.Atoi(strconv.Itoa(year)[2:])
	log.Println("subyear:", subyear)
	packet[0x10] = byte(subyear)
	packet[0x11] = byte(now.Weekday())
	packet[0x12] = byte(now.Day())
	packet[0x13] = byte(now.Month())
	a, _ := strconv.Atoi(strings.Split(ip, ".")[0])
	packet[0x18] = byte(a)
	b, _ := strconv.Atoi(strings.Split(ip, ".")[1])
	packet[0x19] = byte(b)
	c, _ := strconv.Atoi(strings.Split(ip, ".")[2])
	packet[0x1a] = byte(c)
	d, _ := strconv.Atoi(strings.Split(ip, ".")[3])
	packet[0x1b] = byte(d)
	packet[0x1c] = byte(port & 0xff)
	packet[0x1d] = byte(port >> 8)
	packet[0x26] = 6
	checksum := 0xbeaf
	log.Println("checksum:", checksum)

	for i := 0; i < len(packet); i++ {
		checksum += int(packet[i])
	}

	checksum = checksum & 0xffff
	packet[0x20] = byte(checksum & 0xff)
	packet[0x21] = byte(checksum >> 8)

	log.Println("Sending packet:", packet)

	log.Println("Sending message..")

	broadcastAddress, err := net.ResolveUDPAddr("udp4", getAddressString(BroadcastAddress, 80))
	handleErr(err)
	log.Println("Broadcast address:", broadcastAddress.String())

	// length, err := connection.Write(packet) // FIXME: This might be our issue, so maybe we need to use WriteTo/WriteToUDP?
	// length, err := connection.WriteTo(packet, &net.UDPAddr{IP: net.IP{255, 255, 255, 255}, Port: 80})
	length, err := connection.WriteTo(packet, broadcastAddress)
	handleErr(err)
	log.Println("Sent length:", length)
	log.Println("Sent message:", packet)
}

func getAddressString(ip string, port int) string {
	return strings.Join([]string{ip, strconv.Itoa(port)}, ":")
}

func handleErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
