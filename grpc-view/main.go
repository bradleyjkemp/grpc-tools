package main

import (
	"encoding/json"
	"fmt"
	"github.com/bradleyjkemp/grpc-tools/internal"
	"github.com/rivo/tview"
	"io"
	"os"
)

func main() {
	previewPane := makePreviewPane()
	rpcTable := makeRPCSelectionTable(previewPane)

	grid := tview.NewGrid().
		SetRows(-1, -2).
		SetColumns(0).
		SetBorders(true)

	grid.AddItem(rpcTable, 0, 0, 1, 1, 0, 0, false).
		AddItem(previewPane, 1, 0, 1, 1, 0, 0, false)

	if err := tview.NewApplication().SetRoot(grid, true).SetFocus(rpcTable).Run(); err != nil {
		panic(err)
	}
}

func makeRPCSelectionTable(previewPane *tview.Table) *tview.Table {
	table := tview.NewTable()
	table.SetSelectable(true, false) // only be able to select entire RPCs

	//// Set up headings
	//table.SetCellSimple(0, 0, "Service")
	//table.SetCellSimple(0, 1, "Method")

	dumpFile, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}

	dumpDecoder := json.NewDecoder(dumpFile)
	var rpcs []internal.RPC
	for rpcCount := 0; ; rpcCount++ {
		rpc := internal.RPC{}
		err := dumpDecoder.Decode(&rpc)
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(fmt.Errorf("failed to decode dump: %s", err))
		}
		table.SetCellSimple(rpcCount, 0, rpc.Service)
		table.SetCellSimple(rpcCount, 1, rpc.Method)
		rpcs = append(rpcs, rpc)
	}

	onchanged := func(rpcNum, _ int) {
		rpc := rpcs[rpcNum]
		for i, message := range rpc.Messages {
			previewPane.SetCellSimple(i, 0, string(message.MessageOrigin))
			var messagePreview string
			if message.Message != nil {
				marshalled, _ := json.Marshal(message.Message)
				messagePreview = string(marshalled)
			} else {
				messagePreview = string(message.RawMessage)
			}

			previewPane.SetCellSimple(i, 1, messagePreview)
		}
	}

	table.SetSelectionChangedFunc(onchanged)
	onchanged(0, 0) // initialise with first RPC previewed
	return table
}

func makePreviewPane() *tview.Table {
	preview := tview.NewTable()
	preview.SetTitle("preview RPC")
	return preview
}
