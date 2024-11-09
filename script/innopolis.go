package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

type NodeIterator struct {
	current *html.Node
}

func NewNodeIterator(node *html.Node) *NodeIterator {
	return &NodeIterator{current: node}
}

func (n *NodeIterator) HasNext() bool {
	return n.current != nil
}

func (n *NodeIterator) Next() *html.Node {
	if n.current == nil {
		return nil
	}

	node := n.current
	n.current = n.current.NextSibling

	return node

}

const (
	endpointURL = "https://5namaz.com/russia/respublika_tatarstan/innopolis/%d-%s/"
)

func main() {
	var (
		years = []int{2024, 2025}
		data  = make([][]string, 0, 365*len(years))
	)

	for _, year := range years {
		for month := range time.Month(12) {
			month++ // month is 1-based

			finalURL := fmt.Sprintf(endpointURL, year, strings.ToLower(month.String()))

			node, err := parseHTML(finalURL)
			checkErr(err)

			text, err := convertCSV(node, int(month), year)
			checkErr(err)

			data = append(data, text...)
		}
	}

	err := storeCSV(data)
	checkErr(err)
}

func parseHTML(url string) (*html.Node, error) {
	const XPATH = `/html/body/div[1]/div/div/div/div[2]/div/div/div/div[2]/div/div/table/tbody`

	doc, err := htmlquery.LoadURL(url)
	checkErr(err)

	node, err := htmlquery.Query(doc, XPATH)
	checkErr(err)

	return node, nil
}

func convertCSV(node *html.Node, month int, year int) ([][]string, error) {
	node = node.FirstChild.NextSibling

	rows := make([][]string, 0, 30)

	nodeIterator := NewNodeIterator(node)

	for nodeIterator.HasNext() {
		innerNode := nodeIterator.Next()
		row := parseRaw(innerNode)
		rows = append(rows, row)
	}

	rows = rows[:len(rows)-2] // remove last 2 row (row-2 is a repeat of the first row of the next month & row-1 is nil)

	for _, row := range rows {
		day := strings.Split(row[0], " ")[0]               // remove day name from the first cell
		row[0] = fmt.Sprintf("%s/%d/%d", day, month, year) // fist cell is the date in format DD/MM/YYYY
	}

	return rows, nil
}

func parseRaw(node *html.Node) []string {
	var (
		column int
		cells  []string
	)

	nodeIterator := NewNodeIterator(node.FirstChild)

	for nodeIterator.HasNext() {
		cell := nodeIterator.Next()

		if column == 4 {
			column++
			continue
		}

		if cell.Data == "td" {
			cells = append(cells, cell.FirstChild.Data)
		}

		column++
	}

	return cells
}

func storeCSV(data [][]string) error {
	const (
		header   = "День,ФАЖР,ВОСХОД,ЗУХР,АСР,МАГРИБ,ИША"
		filename = "innopolis.csv"
	)

	file, err := os.Create(filename)
	checkErr(err)

	defer func(file *os.File) { _ = file.Close() }(file)

	writer := csv.NewWriter(file)
	defer writer.Flush()

	err = writer.Write(strings.Split(header, ","))
	checkErr(err)

	for _, row := range data {
		err = writer.Write(row)
		checkErr(err)
	}

	return nil
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
