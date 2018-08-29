// Copyright 2016 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

//go:generate go run gen.go

// ssd1306 writes to a display driven by a ssd1306 controler.
package main

import (
	"fmt"
	"flag"
	"os"
	"errors"
	"io/ioutil"
	"log"
	"time"

	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/conn/physic"
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
	cdName := flag.String("cd", "", "CD")

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
	cd := gpioreg.ByName(*cdName)

	opts := nrf905.Opts{
		CenterFrequency: 868*physic.MegaHertz + 200*physic.KiloHertz,
		OutputPower: nrf905.PowerM10dBm,
		ReducedRXCurrent: false,
		AutoRetransmit: false,
		TXAddressWidth: nrf905.AddressWidth3,
		RXAddress: []byte{0x93, 0x9a, 0x0c},
		//RXAddress: []byte{0x0, 0x0, 0x0},
		RXPayloadWidth: 24,
		TXPayloadWidth: 24,
		CrystalFrequency: nrf905.Crystal16MHz,
		CRCMode: nrf905.CRC16Bit,
	}

	d, err := nrf905.New(c, trx_en, pwr_up, tx_en, am, dr, cd, &opts)
	if err != nil {
		return err
	}

	d.Standby()
	//err = d.EnableReceive()
	//if err != nil {
	//	return err
	//}
	//for true {
	//	cd, err := d.CarrierDetect()
	//	if err != nil {
	//		return nil
	//	}
	//	log.Printf("Carrier Detect = %t", cd)
	//	dr, err := d.DataReady()
	//	if err != nil {
	//		return nil
	//	}
	//	log.Printf("Data Ready = %t", dr)
	//	if dr {
	//		log.Printf("Payload")
	//		var payload [24]byte
	//		d.Rx(time.Minute, payload[:])
	//		for _, x := range payload {
	//			log.Printf("02x ", x)
	//		}
	//	}
	//	time.Sleep(time.Second)
	//}

	d.SetTxAddress([]byte{0x1a, 0x2a, 0x3a})
	var payload [24]byte
	err = d.Tx(time.Minute, payload[:], false)
	if err != nil {
		return err
	}
	for true {
		;
	}

	return nil
}

func main() {
	if err := mainImpl(); err != nil {
		fmt.Fprintf(os.Stderr, "nrf905: %s.\n", err)
		os.Exit(1)
	}
}
