

package nrf905

import (
	"log"
	"fmt"
	"errors"

	"periph.io/x/periph/conn"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/conn/spi"
	"periph.io/x/periph/conn/gpio"
	"time"
	)

func New(p spi.Port, trx_ce gpio.PinOut, pwr_up gpio.PinOut, tx_en gpio.PinOut, am gpio.PinIn, dr gpio.PinIn, cd gpio.PinIn, opts *Opts) (*Dev, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	c, err := p.Connect(10*physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		return nil, err
	}
	if err := trx_ce.Out(gpio.Low); err != nil {
		return nil, err
	}
	if err := pwr_up.Out(gpio.Low); err != nil {
		return nil, err
	}
	if err := tx_en.Out(gpio.Low); err != nil {
		return nil, err
	}

	if am != nil {
		if err := am.In(gpio.PullDown, gpio.NoEdge); err != nil {
			return nil, err
		}
	}
	if dr != nil {
		if err := dr.In(gpio.PullDown, gpio.RisingEdge); err != nil {
			return nil, err
		}
	}

	d := &Dev{  c: c,
				trx_ce: trx_ce,
				pwr_up: pwr_up,
				tx_en: tx_en,
				am: am,
				dr: dr,
				cd: cd,
				state: powerDownState}
	d.PowerDown()
	d.writeRFConfig(opts)
	return d, nil
}

type Dev struct {
	c               conn.Conn

	trx_ce          gpio.PinOut
	pwr_up          gpio.PinOut
	tx_en           gpio.PinOut

	am              gpio.PinIn
	dr              gpio.PinIn
	cd              gpio.PinIn

	state           operatingState
}

type Power uint8

const (
	PowerM10dBm  Power = 0
	PowerM2dBm   Power = 1
	Power6dBm    Power = 2
	Power10dBm   Power = 3
)

type AddressWidth uint8

const (
	AddressWidth1 = 1
	AddressWidth2 = 2
	AddressWidth3 = 3
	AddressWidth4 = 4
)

type Address []byte

type PayloadWidth uint8

type CRCMode uint8

const (
	NoCRC       CRCMode = iota
	CRC8Bit
	CRC16Bit
)

type CrystalFrequency uint8

const (
	Crystal4MHz   CrystalFrequency = 0
	Crystal8MHz   CrystalFrequency = 1
	Crystal12MHz  CrystalFrequency = 2
	Crystal16MHz  CrystalFrequency = 3
	Crystal20MHz  CrystalFrequency = 4
)

type Opts struct {
	CenterFrequency     physic.Frequency
	OutputPower         Power
	ReducedRXCurrent    bool
	AutoRetransmit      bool
	TXAddressWidth      AddressWidth
	RXAddress           Address
	RXPayloadWidth      PayloadWidth
	TXPayloadWidth      PayloadWidth
	CrystalFrequency    CrystalFrequency
	CRCMode             CRCMode
}

var DefaultOpts = Opts{
	OutputPower: PowerM10dBm,
	ReducedRXCurrent: false,
	AutoRetransmit: false,
	TXAddressWidth: AddressWidth4,
	RXPayloadWidth: 32,
	TXPayloadWidth: 32,
	CrystalFrequency: Crystal16MHz,
	CRCMode: CRC16Bit,
}

func b2i(b bool) byte {
	if b {
		return 1
	}
	return 0
}

const (
	// RFConfig[1]
	AutoRetransmitOffset    = 5
	ReducedRXCurrentOffset  = 4
	OutputPowerOffset       = 2
	PllOffset               = 1
	// RFConfig[2]
	TXAddressWidthOffset    = 4
	RXAddressWidthOffset    = 0
	// RFConfig[9]
	CRCModeOffset           = 7
	CRCEnableOffset         = 6
	CrystalFrequencyOffset  = 3
)

func getAddressWidth(address Address) (AddressWidth, error) {
	if len(address) < 1 && len(address) > 4 {
	    return AddressWidth1, errors.New("invalid address width")
	}
	return AddressWidth(len(address)), nil
}

const (
	PllLowMinFrequency physic.Frequency     = 422400*physic.KiloHertz
	PllLowMaxFrequency physic.Frequency     = 473500*physic.KiloHertz
	PllHighMinFrequency physic.Frequency    = 844800*physic.KiloHertz
	PllHighMaxFrequency physic.Frequency    = 947000*physic.KiloHertz
)

func freqToChannelPll(frequency physic.Frequency) (byte, bool, error) {
	//  fRF = ( 422.4 + CH_NOd /10)*(1+HFREQ_PLLd) MHz
	if frequency >= PllLowMinFrequency && frequency <= PllLowMaxFrequency {
		return byte((frequency - PllLowMinFrequency) / (100 * physic.KiloHertz)), false, nil
	} else if frequency >= PllHighMinFrequency && frequency <= PllHighMaxFrequency {
		return byte((frequency - PllHighMinFrequency) / (200 * physic.KiloHertz)), true, nil
	} else {
		return 0, false, fmt.Errorf("center Frequency %d not supported", frequency)
	}
}

func (opts *Opts) validate() error {
	_, err := getAddressWidth(opts.RXAddress)
	if err != nil {
		return err
	}
	if opts.RXPayloadWidth > 32 {
		return errors.New("RX payload width to large")
	}
	if opts.TXPayloadWidth > 32 {
		return errors.New("TX payload width to large")
	}
	_, _, err = freqToChannelPll(opts.CenterFrequency)
	if err != nil {
		return err
	}
	return nil
}

func (opts *Opts) encode(data []byte) {
	channel, pll, _ := freqToChannelPll(opts.CenterFrequency)
	data[0] = byte(channel & 0xFF)
	data[1] = b2i(opts.AutoRetransmit) << AutoRetransmitOffset |
			  b2i(opts.ReducedRXCurrent) << ReducedRXCurrentOffset |
			  byte(opts.OutputPower << OutputPowerOffset) |
			  b2i(pll) << PllOffset | byte(channel >> 8)
	RXAddressWidth, _ := getAddressWidth(opts.RXAddress)
	data[2] = byte(opts.TXAddressWidth << TXAddressWidthOffset) |
			  byte(RXAddressWidth)
	data[3] = byte(opts.RXPayloadWidth)
	data[4] = byte(opts.TXPayloadWidth)
	for i := 0; i < int(RXAddressWidth); i++ {
        data[5 + i] = opts.RXAddress[i]
    }
	data[9] = byte(opts.CrystalFrequency << CrystalFrequencyOffset)
	switch opts.CRCMode {
	case CRC8Bit:
		data[9] = data[9] | 1 << CRCEnableOffset
	case CRC16Bit:
		data[9] = data[9] | 1 << CRCEnableOffset | 1 << CRCModeOffset
	}
}

type readInstruction uint8
type writeInstruction uint8

const (
	writeRFInstruction         writeInstruction    = 0
	readRFInstruction          readInstruction     = 0x10
	writeTXPayloadInstruction  writeInstruction    = 0x20
	readTXPayloadInstruction   readInstruction     = 0x21
	writeTXAddressInstruction  writeInstruction    = 0x22
	readTXAddressInstruction   readInstruction     = 0x23
	readRXPayloadInstruction   readInstruction     = 0x24
	channelConfigInstruction   writeInstruction    = 0x80
	statusRegisterInstruction  readInstruction     = 0xFF
)

type statusRegister struct {
	addressMatch    bool
	dataReady       bool
}

const (
	addressMatchOffset  = 7
	dataReadyOffset     = 5
)

func byteToStatusRegister(data byte) *statusRegister {
	return &statusRegister{
		dataReady: (data >> dataReadyOffset) & 0x1 == 0x1,
		addressMatch: (data >> addressMatchOffset) & 0x1 == 0x1}
}

var writeBuffer [33 /* max SPI data size  + 1 */]byte
var readBuffer [33 /* max SPI data size + 1 */]byte

func (d *Dev) writeCommand(instruction writeInstruction, data []byte) (*statusRegister, error) {
	writeBuffer[0] = byte(instruction)
	end := len(data) + 1
	copy(writeBuffer[1:end], data)
	if err := d.c.Tx(writeBuffer[:end], readBuffer[:end]); err != nil {
		return nil, err
	}
	return byteToStatusRegister(readBuffer[0]), nil
}

func (d *Dev) readCommand(instruction readInstruction, data []byte) (*statusRegister, error) {
	writeBuffer[0] = byte(instruction)
	end := len(data) + 1
	for i := 1; i <= end; i++ {
		writeBuffer[i] = 0xFF
	}
	if err := d.c.Tx(writeBuffer[:end], readBuffer[:end]); err != nil {
		return nil, err
	}
	copy(data, readBuffer[1:end])

	return byteToStatusRegister(readBuffer[0]), nil
}

func (d *Dev) writeRFConfig(opts *Opts) error {
	var rfConfig [10]byte
	opts.encode(rfConfig[:])
	if _, err := d.writeCommand(writeRFInstruction, rfConfig[:]); err != nil {
		return err
	}
	printRawRFConfig(rfConfig[:])
	return nil
}

func printRawRFConfig(data []byte) {
	log.Printf("RFConfig:")
	log.Printf("RFConfig[0] - CH_NO[7:0]: %02x", data[0])
	log.Printf("RFConfig[1] - AUTO_RETRAN,RX_RED_PWR,PA_PWR[1:0],HFREQ_PLL,CH_NO[8]: %02x", data[1])
	log.Printf("RFConfig[2] - TX_AFW[2:0],RX_AFW[2:0]: %02x", data[2])
	log.Printf("RFConfig[3] - RX_PW[5:0]: %02x", data[3])
	log.Printf("RFConfig[4] - TX_PW[5:0]: %02x", data[4])
	log.Printf("RFConfig[5] - RX_ADDRESS: %02x", data[5])
	log.Printf("RFConfig[6] - RX_ADDRESS: %02x", data[6])
	log.Printf("RFConfig[7] - RX_ADDRESS: %02x", data[7])
	log.Printf("RFConfig[8] - RX_ADDRESS: %02x", data[8])
	log.Printf("RFConfig[9] - CRC_MODE,CRC_EN,XOF[2:0],UP_CLK_EN,UP_CLK_FREQ[1:0]: %02x", data[9])
}

func (d *Dev) addressMatch() (bool, error) {
	if d.am != nil {
		return bool(d.am.Read()), nil
	} else {
		var tmp [0]byte
		statusRegister, err := d.readCommand(statusRegisterInstruction, tmp[:])
		if err != nil {
			return false, err
		}
		return statusRegister.addressMatch, nil
	}
}

func (d *Dev) DataReady() (bool, error) {
	if d.dr != nil {
		return bool(d.dr.Read()), nil
	} else {
		var tmp [0]byte
		statusRegister, err := d.readCommand(statusRegisterInstruction, tmp[:])
		if err != nil {
			return false, err
		}
		return statusRegister.dataReady, nil
	}
}

func (d *Dev) CarrierDetect() (bool, error) {
	if d.cd != nil {
		return bool(d.cd.Read()), nil
	}
	return false, errors.New("carrier detect not connected")
}

type operatingState uint

const (
	powerDownState  operatingState = iota
	standbyState
	radioRXState
	radioTXState
)

func (d *Dev) PowerDown() {
	d.pwr_up.Out(gpio.Low)
	d.trx_ce.Out(gpio.Low)
	d.tx_en.Out(gpio.Low)
	d.state = powerDownState
}

func (d *Dev) Standby() {
	d.trx_ce.Out(gpio.Low)
	d.tx_en.Out(gpio.Low)
	d.pwr_up.Out(gpio.High)
	d.state = standbyState
}

func (d *Dev) EnableReceive() error {
	if d.state != radioRXState {
		if d.state != standbyState {
			return errors.New("not in standby or RX operating state")
		} else {
			d.trx_ce.Out(gpio.High)
			d.state = radioRXState
		}
	}
	return nil
}

func (d *Dev) Receive(timeout time.Duration, payload []byte) error {
	if err := d.EnableReceive(); err != nil {
		return err
	}
	if !d.dr.WaitForEdge(timeout) {
		return errors.New("time out waiting for data ready")
	}
	// TODO remove
	dr, _ := d.DataReady()
	log.Printf("Receive: dr = %t", dr)

	if _, err := d.readCommand(readRXPayloadInstruction, payload); err != nil {
		return err
	}
	return nil
}

