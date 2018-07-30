

package nrf905

import (
    "fmt"

    "periph.io/x/periph/conn"
    "periph.io/x/periph/conn/physic"
    "periph.io/x/periph/conn/spi"
    "periph.io/x/periph/conn/gpio"
)

func New(p spi.Port, trx_ce gpio.PinOut, pwr_up gpio.PinOut, tx_en gpio.PinOut, am gpio.PinIn, dr gpio.PinIn) (*Dev, error) {
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

func (d *Dev) ReadRF() error {
    cmd := [11]byte{0x10}
    var rf [11]byte
    if err := d.c.Tx(cmd[:], rf[:]); err != nil {
        return err
    }

    fmt.Println("rf = %h", rf)

    return nil
}
