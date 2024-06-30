package usbSerial

import (
	"SplitFlapApp/utils"
	"bufio"
	"fmt"
	"go.bug.st/serial"
)

const DEFAULT_BAUDRATE = 230400
const retryTimeout float32 = 0.25

type SerialConnection interface {
	Open(portName string) error
	Write(data []byte) error
	Read() ([]byte, error)
	Close() error
}

func NewSerialConnection() *Serial {
	list, err := serial.GetPortsList()
	if err != nil {
		utils.Log(utils.LogLevel_Error, "Failed to get port list", err)
		return nil
	}

	if len(list) == 0 {
		utils.Log(utils.LogLevel_Error, "No ports available", err)
		return nil
	}

	return NewSerialConnectionOnPort(list[0])
}

func NewSerialConnectionOnPort(port string) *Serial {
	s := Serial{}
	err := s.Open(port)
	if err != nil {
		utils.Log(utils.LogLevel_Error, "Failed to open port", err)
		return nil
	}

	utils.Log(utils.LogLevel_Info, fmt.Sprintf("Connecting to port %s", port), nil)
	return &s
}

type Serial struct {
	serial *serial.Port
}

func (s *Serial) getSerial() serial.Port {
	return *s.serial
}

func (s *Serial) Open(portName string) error {
	mode := serial.Mode{
		BaudRate: DEFAULT_BAUDRATE,
		DataBits: 8,
	}

	port, err := serial.Open(portName, &mode)
	if err != nil {
		return err
	}

	s.serial = &port
	return nil
}

func (s *Serial) Write(data []byte) error {
	w, err := s.getSerial().Write(data)
	if err != nil {
		utils.Log(utils.LogLevel_Error, "Failed to write to serial", err)
	}

	utils.Log(utils.LogLevel_Debug, fmt.Sprintf("Bytes written %d", w), nil)
	return err
}

func (s *Serial) Read() ([]byte, error) {
	var buffer []byte

	reader := bufio.NewReader(s.getSerial())
	reply, err := reader.ReadBytes(byte(0))
	if err != nil {
		utils.Log(utils.LogLevel_Error, "Failed to read from serial", err)
		return buffer, err
	}

	return reply, err
}

func (s *Serial) Close() error {
	return s.getSerial().Close()
}
