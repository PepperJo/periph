

package nrf905

import (
	"periph.io/x/periph/conn"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/conn/spi"
)

func New(p spi.Port, trx_ce gpio.PinOut, pwr_up gpio.PinOut, tx_en gpio.PinOut, am gpio.PinIn, dr gpio.PinIn) {
    c, err := p.Connect(10*physic.MegaHertz, spi.Mode0, 8)
    if err != nil {
        return nil, err
    }
}

type Dev struct {
    c       conn.Conn

}
