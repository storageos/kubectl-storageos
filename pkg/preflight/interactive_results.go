package preflight

import (
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/pkg/errors"
	ui "github.com/replicatedhq/termui/v3"
	"github.com/replicatedhq/termui/v3/widgets"
	"github.com/replicatedhq/troubleshoot/cmd/util"
	analyzerunner "github.com/replicatedhq/troubleshoot/pkg/analyze"
)

var (
	selectedResult = 0
	table          = widgets.NewTable()
	isShowingSaved = false
)

func showInteractiveResults(preflightName string, analyzeResults []*analyzerunner.AnalyzeResult) error {
	if err := ui.Init(); err != nil {
		return errors.Wrap(err, "failed to create terminal ui")
	}
	defer ui.Close()

	drawUI(preflightName, analyzeResults)

	for {
		e := <-ui.PollEvents()
		switch e.ID {
		case "<C-c>":
			return nil
		case "q":
			if !isShowingSaved {
				return nil
			}
			isShowingSaved = false
			ui.Clear()
			drawUI(preflightName, analyzeResults)
		case "s":
			filename, err := save(preflightName, analyzeResults)
			if err == nil {
				showSaved(filename)
				go func() {
					time.Sleep(time.Second * 5)
					isShowingSaved = false
					ui.Clear()
					drawUI(preflightName, analyzeResults)
				}()
			}
		case "<Resize>":
			ui.Clear()
			drawUI(preflightName, analyzeResults)
		case "<Down>":
			if selectedResult < len(analyzeResults)-1 {
				selectedResult++
			} else {
				selectedResult = 0
				table.SelectedRow = 0
			}
			table.ScrollDown()
			ui.Clear()
			drawUI(preflightName, analyzeResults)
		case "<Up>":
			if selectedResult > 0 {
				selectedResult--
			} else {
				selectedResult = len(analyzeResults) - 1
				table.SelectedRow = len(analyzeResults)
			}
			table.ScrollUp()
			ui.Clear()
			drawUI(preflightName, analyzeResults)
		}
	}
}

func drawUI(preflightName string, analyzeResults []*analyzerunner.AnalyzeResult) {
	drawGrid(analyzeResults)
	drawHeader(preflightName)
	drawFooter()
}

func drawGrid(analyzeResults []*analyzerunner.AnalyzeResult) {
	drawPreflightTable(analyzeResults)
	drawDetails(analyzeResults[selectedResult])
}

func drawHeader(preflightName string) {
	termWidth, _ := ui.TerminalDimensions()

	title := widgets.NewParagraph()
	title.Text = fmt.Sprintf("%s Preflight Checks", util.AppName(preflightName))
	title.TextStyle.Fg = ui.ColorWhite
	title.TextStyle.Bg = ui.ColorClear
	title.TextStyle.Modifier = ui.ModifierBold
	title.Border = false

	left := termWidth/2 - 2*len(title.Text)/3
	right := termWidth/2 + (termWidth/2 - left)

	title.SetRect(left, 0, right, 1)
	ui.Render(title)
}

func drawFooter() {
	termWidth, termHeight := ui.TerminalDimensions()

	instructions := widgets.NewParagraph()
	instructions.Text = "[q] quit    [s] save    [↑][↓] scroll"
	instructions.Border = false

	left := 0
	right := termWidth
	top := termHeight - 1
	bottom := termHeight

	instructions.SetRect(left, top, right, bottom)
	ui.Render(instructions)
}

func drawPreflightTable(analyzeResults []*analyzerunner.AnalyzeResult) {
	termWidth, termHeight := ui.TerminalDimensions()

	table.SetRect(0, 3, termWidth/2, termHeight-6)
	table.FillRow = true
	table.Border = true
	table.Rows = [][]string{}
	table.ColumnWidths = []int{termWidth}

	for i, analyzeResult := range analyzeResults {
		title := analyzeResult.Title

		switch {
		case analyzeResult.IsPass:
			title = fmt.Sprintf("✔  %s", title)
		case analyzeResult.IsWarn:
			title = fmt.Sprintf("⚠️  %s", title)
		case analyzeResult.IsFail:
			title = fmt.Sprintf("✘  %s", title)
		}
		table.Rows = append(table.Rows, []string{
			title,
		})

		switch {
		case analyzeResult.IsPass:
			table.RowStyles[i] = ui.NewStyle(ui.ColorGreen, ui.ColorClear)
			if i == selectedResult {
				table.RowStyles[i] = ui.NewStyle(ui.ColorGreen, ui.ColorClear, ui.ModifierReverse)
			}
		case analyzeResult.IsWarn:
			table.RowStyles[i] = ui.NewStyle(ui.ColorYellow, ui.ColorClear)
			if i == selectedResult {
				table.RowStyles[i] = ui.NewStyle(ui.ColorYellow, ui.ColorClear, ui.ModifierReverse)
			}
		case analyzeResult.IsFail:
			table.RowStyles[i] = ui.NewStyle(ui.ColorRed, ui.ColorClear)
			if i == selectedResult {
				table.RowStyles[i] = ui.NewStyle(ui.ColorRed, ui.ColorClear, ui.ModifierReverse)
			}
		}

	}

	ui.Render(table)
}

func drawDetails(analysisResult *analyzerunner.AnalyzeResult) {
	termWidth, _ := ui.TerminalDimensions()

	currentTop := 4
	title := widgets.NewParagraph()
	title.Text = analysisResult.Title
	title.Border = false

	switch {
	case analysisResult.IsPass:
		title.TextStyle = ui.NewStyle(ui.ColorGreen, ui.ColorClear, ui.ModifierBold)
	case analysisResult.IsWarn:
		title.TextStyle = ui.NewStyle(ui.ColorYellow, ui.ColorClear, ui.ModifierBold)
	case analysisResult.IsFail:
		title.TextStyle = ui.NewStyle(ui.ColorRed, ui.ColorClear, ui.ModifierBold)
	}

	height := estimateNumberOfLines(title.Text, termWidth/2)
	title.SetRect(termWidth/2, currentTop, termWidth, currentTop+height)
	ui.Render(title)
	currentTop = currentTop + height + 1

	message := widgets.NewParagraph()
	message.Text = analysisResult.Message
	message.Border = false
	height = estimateNumberOfLines(message.Text, termWidth/2) + 2
	message.SetRect(termWidth/2, currentTop, termWidth, currentTop+height)
	ui.Render(message)
	currentTop = currentTop + height + 1

	if analysisResult.URI != "" {
		uri := widgets.NewParagraph()
		uri.Text = fmt.Sprintf("For more information: %s", analysisResult.URI)
		uri.Border = false
		height = estimateNumberOfLines(uri.Text, termWidth/2)
		uri.SetRect(termWidth/2, currentTop, termWidth, currentTop+height)
		ui.Render(uri)
	}
}

func estimateNumberOfLines(text string, width int) int {
	lines := len(text)/width + 1
	return lines
}

func save(preflightName string, analyzeResults []*analyzerunner.AnalyzeResult) (string, error) {
	results := fmt.Sprintf("%s Preflight Checks\n\n", util.AppName(preflightName))
	for _, analyzeResult := range analyzeResults {
		result := ""

		switch {
		case analyzeResult.IsPass:
			result = "Check PASS\n"
		case analyzeResult.IsWarn:
			result = "Check WARN\n"
		case analyzeResult.IsFail:
			result = "Check FAIL\n"
		}

		result = result + fmt.Sprintf("Title: %s\n", analyzeResult.Title)
		result = result + fmt.Sprintf("Message: %s\n", analyzeResult.Message)

		if analyzeResult.URI != "" {
			result = result + fmt.Sprintf("URI: %s\n", analyzeResult.URI)
		}

		result = result + "\n------------\n"

		results = results + result
	}

	// Overwrite any previous data.
	filename := path.Join(util.HomeDir(), fmt.Sprintf("%s-results.txt", preflightName))
	if err := ioutil.WriteFile(filename, []byte(results), 0644); err != nil {
		return "", errors.Wrap(err, "failed to save preflight results")
	}

	return filename, nil
}

func showSaved(filename string) {
	termWidth, termHeight := ui.TerminalDimensions()

	savedMessage := widgets.NewParagraph()
	savedMessage.Text = fmt.Sprintf("Preflight results saved to\n\n%s", filename)
	savedMessage.WrapText = true
	savedMessage.Border = true

	left := termWidth/2 - 20
	right := termWidth/2 + 20
	top := termHeight/2 - 4
	bottom := termHeight/2 + 4

	savedMessage.SetRect(left, top, right, bottom)
	ui.Render(savedMessage)

	isShowingSaved = true
}
