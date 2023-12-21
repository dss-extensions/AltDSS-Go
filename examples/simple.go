package main

import (
	"log"

	"github.com/dss-extensions/altdss-go/altdss"
)

func main() {
	dss := altdss.IDSS{}
	dss.Init(nil)
	err := dss.Text.Set_Command("redirect ../electricdss-tst/Version8/Distrib/IEEETestCases/13Bus/IEEE13Nodeckt.dss")
	if err != nil {
		log.Fatal(err)
	}

	numNodes, err := dss.ActiveCircuit.NumNodes()
	if err != nil {
		log.Fatal(err)
	}
	println("Number of nodes:", numNodes)
	names, err := dss.ActiveCircuit.AllNodeNames()
	if err != nil {
		log.Fatal(err)
	}
	vmag, err := dss.ActiveCircuit.AllBusVmag()
	if err != nil {
		log.Fatal(err)
	}
	cvolts, err := dss.ActiveCircuit.AllBusVolts()
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < len(names); i++ {
		println(i, names[i], vmag[i]/1000, cvolts[i])
	}
	dss2, err := dss.NewContext()
	if err != nil {
		log.Fatal(err)
	}
	err = dss2.Text.Set_Command("new circuit.test2")
	if err != nil {
		log.Fatal(err)
	}

	circName, err := dss.ActiveCircuit.Name()
	if err != nil {
		log.Fatal(err)
	}
	println("Circuit name:", circName)

	circName2, err := dss2.ActiveCircuit.Name()
	if err != nil {
		log.Fatal(err)
	}
	println("First instance (Prime/Default), circuit name:", circName)
	println("Second instance, circuit name:", circName2)

	// Test bus activation
	println()
	println("Selecting bus 632")
	busNum, err := dss.ActiveCircuit.SetActiveBus("632")
	if err != nil {
		log.Fatal(err)
	}
	busName, err := dss.ActiveCircuit.ActiveBus.Name()
	println("Active Bus:", busName, "number", busNum)
	println("Selecting bus 671")
	busNum, err = dss.ActiveCircuit.SetActiveBus("671")
	if err != nil {
		log.Fatal(err)
	}
	busName, err = dss.ActiveCircuit.ActiveBus.Name()
	println("Active Bus:", busName, "number", busNum)

}
