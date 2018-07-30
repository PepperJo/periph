// Copyright 2016 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

//go:generate go run gen.go

// ssd1306 writes to a display driven by a ssd1306 controler.
package main

import (
    "periph.io/x/periph/conn/gpio/gpioreg"
    "periph.io/x/periph/conn/spi"
    "periph.io/x/periph/conn/spi/spireg"
    "periph.io/x/periph/devices/nrf905"
)

func mainImpl() error {
    spiID := flag.String("spi", "", "SPI port to use")
    trxceName := flag.String("trxce", "", "TRX_CE")
    pwrupName := flag.String("pwrup", "", "PWR_UP")
    txenName := flag.String("txen", "", "TX_EN")
    amName := flag.String("am", "", "AM")
    drName := flag.String("dr", "", "DR")

    verbose := flag.Bool("v", false, "verbose mode")
    flag.Parse()
    if !*verbose {
        log.SetOutput(ioutil.Discard)
    }
    log.SetFlags(log.Lmicroseconds)
    if flag.NArg() != 0 {
        return errors.New("unexpected argument, try -help")
    }

    if _, err := hostInit(); err != nil {
        return err
    }

    c, err := spireg.Open(*spiID)
    if err != nil {
        return err
    }
    defer c.Close()
    if p, ok := c.(spi.Pins); ok {
        log.Printf("Using pins CLK: %s  MOSI: %s MISO: %s  CS: %s", p.CLK(), p.MOSI(), p.MISO(), p.CS())
    }

    trx_en := gpioreg.ByName(*trxceName)
    pwr_up := gpioreg.ByName(*pwrupName)
    tx_en := gpioreg.ByName(*txenName)
    am := gpioreg.ByName(*amName)
    dr := gpioreg.ByName(*drName)

    s, err := nrf905.New(c, trx_en, pwr_up, tx_en, am, dr)
    if err != nil {
        return err
    }

    return nil
}

func main() {
    if err := mainImpl(); err != nil {
        fmt.Fprintf(os.Stderr, "nrf905: %s.\n", err)
        os.Exit(1)
    }
}
