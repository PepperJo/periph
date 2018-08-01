

package nrf905

import (
    "fmt"
    "log"

    "periph.io/x/periph/conn"
    "periph.io/x/periph/conn/physic"
    "periph.io/x/periph/conn/spi"
    "periph.io/x/periph/conn/gpio"
)

func New(p spi.Port, trx_ce gpio.PinOut, pwr_up gpio.PinOut, tx_en gpio.PinOut, am gpio.PinIn, dr gpio.PinIn, opts *Opts) (*Dev, error) {
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

    return &Dev{c: c,
                trx_ce: trx_ce,
                pwr_up: pwr_up,
                tx_en: tx_en,
                am: am,
                dr: dr}, nil
}

type Dev struct {
    c       conn.Conn

    trx_ce  gpio.PinOut
    pwr_up  gpio.PinOut
    tx_en   gpio.PinOut

    am      gpio.PinIn
    dr      gpio.PinIn
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

func (opts *Opts) encode(data [10]byte) {
    //  fRF = ( 422.4 + CH_NOd /10)*(1+HFREQ_PLLd) MHz
    var channel = 128 * (78125 * opts.CenterFrequency - 33)
    var pll = 0
    if channel & 0x1F != channel {
        channel = 64 * (78125 * opts.CenterFrequency - 66)
        pll = 1
    }
}

func (opts *Opts) decode(data [10]byte) error {
    log.Printf("Decode RFConfig:")
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

    return nil
}

func (d *Dev) ReadRF() error {
    cmd := [11]byte{0x10,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF}
    var rf [11]byte
    if err := d.c.Tx(cmd[:], rf[:]); err != nil {
        return err
    }

    for _, e := range rf {
        fmt.Printf("%x ", e)
    }
    fmt.Println()

    return nil
}
