

package nrf905

import (
    "log"
    "errors"

    "periph.io/x/periph/conn"
    "periph.io/x/periph/conn/physic"
    "periph.io/x/periph/conn/spi"
    "periph.io/x/periph/conn/gpio"
)

func New(p spi.Port, trx_ce gpio.PinOut, pwr_up gpio.PinOut, tx_en gpio.PinOut, am gpio.PinIn, dr gpio.PinIn, opts *Opts) (*Dev, error) {
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

    if err := am.In(gpio.PullDown, gpio.NoEdge); err != nil {
        return nil, err
    }
    if err := dr.In(gpio.PullDown, gpio.RisingEdge); err != nil {
        return nil, err
    }

    d := &Dev{c: c,
                trx_ce: trx_ce,
                pwr_up: pwr_up,
                tx_en: tx_en,
                am: am,
                dr: dr}
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
    var rxAddressWidth AddressWidth
    switch len(address) {
    case 1:
        rxAddressWidth = AddressWidth1
    case 4:
        rxAddressWidth = AddressWidth4
    default:
        return AddressWidth1, errors.New("invalid address width")
    }
    return rxAddressWidth, nil
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
    return nil
}

func (opts *Opts) encode(data []byte) {
    //  fRF = ( 422.4 + CH_NOd /10)*(1+HFREQ_PLLd) MHz
    var channel = 128 * (78125 * opts.CenterFrequency - 33)
    var pll byte = 0
    if channel & 0x1FF != channel {
        channel = 64 * (78125 * opts.CenterFrequency - 66)
        pll = 1
    }
    data[0] = byte(channel & 0xFF)
    data[1] = b2i(opts.AutoRetransmit) << AutoRetransmitOffset |
              b2i(opts.ReducedRXCurrent) << ReducedRXCurrentOffset |
              byte(opts.OutputPower << OutputPowerOffset) |
              pll << PllOffset | byte(channel >> 8)
    RXAddressWidth, _ := getAddressWidth(opts.RXAddress)
    data[2] = byte(opts.TXAddressWidth << TXAddressWidthOffset) |
              byte(RXAddressWidth)
    data[3] = byte(opts.RXPayloadWidth)
    data[4] = byte(opts.TXPayloadWidth)
    switch RXAddressWidth {
    case AddressWidth1:
        data[5] = opts.RXAddress[0]
    case AddressWidth4:
        data[5] = opts.RXAddress[0]
        data[6] = opts.RXAddress[1]
        data[7] = opts.RXAddress[2]
        data[8] = opts.RXAddress[3]
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
    for i := range writeBuffer[1:end] {
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
    printRawRFConfig(rfConfig[:])
    _, err := d.writeCommand(writeRFInstruction, rfConfig[:])
    if err != nil {
        return err
    }
    var rfConfig2 [10]byte
    _, err = d.readCommand(readRFInstruction, rfConfig2[:])
    if err != nil {
        return err
    }
    printRawRFConfig(rfConfig2[:])
    if rfConfig != rfConfig2 {
        log.Printf("RFConfig not equal")
    }
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

