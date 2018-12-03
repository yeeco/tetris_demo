/*
 * // Copyright (C) 2017 gyee authors
 * //
 * // This file is part of the gyee library.
 * //
 * // the gyee library is free software: you can redistribute it and/or modify
 * // it under the terms of the GNU General Public License as published by
 * // the Free Software Foundation, either version 3 of the License, or
 * // (at your option) any later version.
 * //
 * // the gyee library is distributed in the hope that it will be useful,
 * // but WITHOUT ANY WARRANTY; without even the implied warranty of
 * // MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * // GNU General Public License for more details.
 * //
 * // You should have received a copy of the GNU General Public License
 * // along with the gyee library.  If not, see <http://www.gnu.org/licenses/>.
 *
 *
 */

package main

import (
	"os"
	"runtime"
	"sync"

	"fmt"
	"math/rand"
	//"os/signal"
	"reflect"
	"strconv"
	//"syscall"
	"time"

	"github.com/urfave/cli"
	"github.com/yeeco/gyee/utils/logging"
	"github.com/yeeco/tetris_demo/node"
)

var (
	app         = cli.NewApp()
	nodeNumber  uint
	txsNumber   uint
	crashNumber uint
	rps         uint
	nodes       []*node.Node
	wg          sync.WaitGroup
)

func init() {
	app.Flags = []cli.Flag{
		cli.UintFlag{
			Name:        "nodenum, n",
			Usage:       "Demo node number",
			Value:       10,
			Destination: &nodeNumber,
		},
		cli.UintFlag{
			Name:        "txsnum, tx",
			Usage:       "Demo transactions number",
			Value:       50000,
			Destination: &txsNumber,
		},
		cli.UintFlag{
			Name:        "crash, c",
			Usage:       "Node crash number",
			Value:       0,
			Destination: &crashNumber,
		},
		cli.UintFlag{
			Name:        "request, rps",
			Usage:       "Demo transactions requests per second",
			Value:       5000,
			Destination: &rps,
		},
	}
	app.Action = demo
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	if err := app.Run(os.Args); err != nil {
		logging.Logger.Fatal(err)
	}
}

func demo(ctx *cli.Context) error {
	nodes = make([]*node.Node, nodeNumber)

	for i := uint(0); i < nodeNumber; i++ {
		nodes[i], _ = node.NewNode(i, nodeNumber)
		nodes[i].Start(&wg)
		nodes[i].BroadcastTransactions(rps, txsNumber)
	}

	go func() {
		cases := make([]reflect.SelectCase, nodeNumber)
		for i := uint(0); i < nodeNumber; i++ {
			cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(nodes[i].Report)}
		}

		fmt.Printf("\n%14s", "")
		for i := uint(0); i < nodeNumber; i++ {
			fmt.Printf("node%-6d", i)
		}
		fmt.Println()
		remaining := len(cases)
		for remaining > 0 {
			chosen, value, ok := reflect.Select(cases)
			if !ok {
				cases[chosen].Chan = reflect.ValueOf(nil)
				remaining -= 1
				continue
			}
			if value.Interface().(bool) {
				fmt.Print("\rHeight(Txs):")
				for i := uint(0); i < nodeNumber; i++ {
					h := len(nodes[i].Blockchain)
					if h > 0 {
						str := fmt.Sprint("(", len(nodes[i].Blockchain[h-1].Tansactions), ")")
						fmt.Printf("%3d%-7s", h, str)
					}
				}
			}
		}
		fmt.Println()
	}()

	go func() { //模拟crash几个节点
		for c := uint(0); c < crashNumber; c++ {
			time.Sleep(time.Duration(rand.Intn(2000)+1000) * time.Millisecond)
			nodes[rand.Intn(int(nodeNumber))].Stop()
		}
	}()

	//go func() {
	//	sigc := make(chan os.Signal, 1)
	//	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	//	defer signal.Stop(sigc)
	//	<-sigc
	//	logging.Logger.Info("Got interrupt, shutting down...")
	//	for c := uint(0); c < nodeNumber; c++ {
	//		fmt.Println(
	//			"node", c,
	//			" eventCh", len(nodes[c].Tetris.EventCh),
	//			" sendCh",  len(nodes[c].Tetris.SendEventCh),
	//			" txCh",    len(nodes[c].Tetris.TxsCh))
	//	}
	//}()

	wg.Wait()

	fmt.Println("\n")
	fmt.Print("                      ")
	for j := uint(0); j < nodeNumber; j++ {
		fmt.Print("balance", j, "  ")
	}
	fmt.Println("")

	for i := uint(0); i < nodeNumber; i++ {
		states := nodes[i].States
		h := len(nodes[i].Blockchain)
		fmt.Printf("%s%-2d%s%3d   ", "Node:", i, " Height:", h)
		for j := uint(0); j < nodeNumber; j++ {
			state := states[strconv.Itoa(int(j))]
			fmt.Printf("%7d   ", state.Balance)
		}
		fmt.Println()
	}

	h := len(nodes[0].Blockchain)
	if h == 0 {
		return nil
	}

	txsum := 0
	for i := 0; i < h; i++ {
		txsum += len(nodes[0].Blockchain[i].Tansactions)
	}
	start := nodes[0].Blockchain[0].Timestamp
	stop := nodes[0].Blockchain[h-1].Timestamp
	interval := stop.Sub(start)

	fmt.Println()
	fmt.Println("Txs confirmed:", txsum, " Time:", interval.Seconds(), " Tps:", int(float64(txsum)/interval.Seconds()), "\n")

	return nil
}
